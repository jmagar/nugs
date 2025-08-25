package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jmagar/nugs/cron/internal/catalog"
	"github.com/jmagar/nugs/cron/internal/models"
)

func main() {
	log.Println("Starting missing shows detection...")

	// Load monitor configuration
	// Note: We no longer need the main config since we don't authenticate here

	monitorConfig, err := loadMonitorConfig("configs/monitor_config.json")
	if err != nil {
		log.Fatal("Error loading monitor config:", err)
	}

	// Load shows data
	showsData := loadShowsData()

	// Create catalog manager
	catalogManager := catalog.NewCatalogManager()

	log.Println("Loading catalog...")

	// Process each monitored artist
	for _, artist := range monitorConfig.Artists {
		if !artist.Monitor {
			continue
		}

		log.Printf("\nProcessing %s (ID: %d)...", artist.Artist, artist.ID)

		// Get available shows from cached catalog
		availableShows, err := catalogManager.GetShowsForArtist(artist.Artist)
		if err != nil {
			log.Printf("Error getting shows for %s: %v", artist.Artist, err)
			continue
		}

		// Get show IDs
		availableIDs := make([]int, len(availableShows))
		for i, show := range availableShows {
			availableIDs[i] = show.ContainerID
		}

		// Get downloaded shows from tootie filesystem
		downloadedIDs, err := getDownloadedShows(artist.ArtistFolder, artist.Artist)
		if err != nil {
			log.Printf("Error scanning downloaded shows for %s: %v", artist.Artist, err)
			continue
		}

		// Calculate missing shows
		missingIDs := findMissingShows(availableIDs, downloadedIDs)

		// Update shows data
		if showsData.Artists == nil {
			showsData.Artists = make(map[string]models.ArtistShowData)
		}

		showsData.Artists[artist.Artist] = models.ArtistShowData{
			ArtistID:   artist.ID,
			Downloaded: downloadedIDs,
			Available:  availableIDs,
			Missing:    missingIDs,
		}

		// Update catalog metadata from catalog manager
		if catalogStats, err := catalogManager.GetCatalogStats(); err == nil {
			showsData.LastCatalogUpdate = catalogStats.LastUpdate
			showsData.CatalogTotalShows = catalogStats.TotalShows
			showsData.CatalogTotalArtists = catalogStats.TotalArtists
		}
		showsData.LastAnalysisTime = time.Now().Format(time.RFC3339)

		// Report results
		log.Printf("Available shows: %d", len(availableIDs))
		log.Printf("Downloaded shows: %d", len(downloadedIDs))
		log.Printf("Missing shows: %d", len(missingIDs))

		if len(missingIDs) > 0 {
			log.Printf("Missing show IDs: %v", missingIDs[:min(10, len(missingIDs))])
			if len(missingIDs) > 10 {
				log.Printf("... and %d more", len(missingIDs)-10)
			}
		}
	}

	// Save updated shows data
	err = saveShowsData(showsData)
	if err != nil {
		log.Fatal("Error saving shows data:", err)
	}

	log.Println("\nMissing shows detection complete!")
	log.Println("Check shows.json for detailed results.")
}

func loadMonitorConfig(filename string) (*models.MonitorConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config models.MonitorConfig
	err = json.Unmarshal(data, &config)
	return &config, err
}

func loadShowsData() *models.ShowsData {
	data, err := os.ReadFile("data/shows.json")
	if err != nil {
		return &models.ShowsData{
			Artists: make(map[string]models.ArtistShowData),
		}
	}

	var shows models.ShowsData
	if err := json.Unmarshal(data, &shows); err != nil {
		fmt.Printf("Warning: failed to unmarshal shows data: %v\n", err)
	}
	if shows.Artists == nil {
		shows.Artists = make(map[string]models.ArtistShowData)
	}

	// Initialize metadata fields if they don't exist
	if shows.LastCatalogUpdate == "" {
		shows.LastCatalogUpdate = "unknown"
	}
	if shows.LastAnalysisTime == "" {
		shows.LastAnalysisTime = "unknown"
	}

	return &shows
}

func saveShowsData(shows *models.ShowsData) error {
	data, err := json.MarshalIndent(shows, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile("data/shows.json", data, 0644)
}

func getDownloadedShows(artistFolder, artistName string) ([]int, error) {
	// Use SSH to list directories on tootie
	cmd := exec.Command("ssh", "tootie", "ls", "-1", fmt.Sprintf("'%s'", artistFolder))

	output, err := cmd.CombinedOutput()
	if err != nil {
		// If directory doesn't exist or SSH fails, return empty list
		return []int{}, nil
	}

	// Parse directory names to log found folders
	folders := strings.Split(strings.TrimSpace(string(output)), "\n")

	// Regular expressions to match different folder name patterns
	// Pattern 1: MM_DD_YY format (newer shows)
	datePattern1 := regexp.MustCompile(`^(\d{2})_(\d{2})_(\d{2})`)
	// Pattern 2: Artist Name - MM_DD_YY format (older shows)
	datePattern2 := regexp.MustCompile(`^` + regexp.QuoteMeta(artistName) + ` - (\d{2})_(\d{2})_(\d{2})`)

	foundCount := 0
	for _, folder := range folders {
		if folder == "" || strings.HasPrefix(folder, ".") || strings.HasSuffix(folder, ".nfo") ||
			strings.HasSuffix(folder, ".jpg") || strings.HasSuffix(folder, ".png") ||
			strings.HasSuffix(folder, ".md") {
			continue
		}

		if datePattern1.MatchString(folder) || datePattern2.MatchString(folder) {
			foundCount++
		}
	}

	if foundCount > 0 {
		log.Printf("Found %d show folders for %s", foundCount, artistName)
	}

	// Parse folder names and match them to container IDs in the catalog
	catalogManager := catalog.NewCatalogManager()
	downloadedIDs := []int{}

	// Get shows for this artist from catalog ONCE (not per folder)
	shows, err := catalogManager.GetShowsForArtist(artistName)
	if err != nil {
		log.Printf("Error getting catalog shows for %s: %v", artistName, err)
		return []int{}, nil
	}

	// Create a map of dates to container IDs for fast lookup
	dateToContainerID := make(map[string]int)
	for _, show := range shows {
		dateToContainerID[show.PerformanceDateShort] = show.ContainerID
	}

	// Parse each folder name and try to match to a container ID
	for _, folder := range folders {
		if folder == "" || strings.HasPrefix(folder, ".") || strings.HasSuffix(folder, ".nfo") ||
			strings.HasSuffix(folder, ".jpg") || strings.HasSuffix(folder, ".png") ||
			strings.HasSuffix(folder, ".md") {
			continue
		}

		var month, day, year string

		// Try Pattern 1: MM_DD_YY format (newer shows)
		if matches := datePattern1.FindStringSubmatch(folder); len(matches) == 4 {
			month, day, year = matches[1], matches[2], matches[3]
		} else if matches := datePattern2.FindStringSubmatch(folder); len(matches) == 4 {
			// Pattern 2: Artist Name - MM_DD_YY format (older shows)
			month, day, year = matches[1], matches[2], matches[3]
		} else {
			// Folder doesn't match expected patterns
			continue
		}

		// Convert MM_DD_YY to MM/DD/YY format to match catalog
		dateToMatch := fmt.Sprintf("%s/%s/%s", month, day, year)

		// Fast lookup in map instead of looping through all shows
		if containerID, exists := dateToContainerID[dateToMatch]; exists {
			downloadedIDs = append(downloadedIDs, containerID)
		}
	}

	log.Printf("Successfully matched %d folders to container IDs for %s", len(downloadedIDs), artistName)
	return downloadedIDs, nil
}

func findMissingShows(available, downloaded []int) []int {
	// Convert downloaded to map for faster lookup
	downloadedMap := make(map[int]bool)
	for _, id := range downloaded {
		downloadedMap[id] = true
	}

	// Find missing shows
	var missing []int
	for _, id := range available {
		if !downloadedMap[id] {
			missing = append(missing, id)
		}
	}

	// Sort for consistent output
	sort.Ints(missing)
	return missing
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
