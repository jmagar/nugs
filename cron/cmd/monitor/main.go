package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jmagar/nugs/cron/internal/api"
	"github.com/jmagar/nugs/cron/internal/catalog"
	"github.com/jmagar/nugs/cron/internal/models"
)

func main() {
	// Load main config
	config, err := loadConfig("configs/config.json")
	if err != nil {
		log.Fatal("Error loading config:", err)
	}

	// Load monitor config
	monitorConfig, err := loadMonitorConfig("configs/monitor_config.json")
	if err != nil {
		log.Fatal("Error loading monitor config:", err)
	}

	// Load shows data
	showsData := loadShowsData()

	// Create catalog manager (no authentication needed for catalog lookups)
	catalogManager := catalog.NewCatalogManager()

	log.Printf("Checking monitored artists for new shows...")

	// Check each monitored artist for new shows
	for _, artist := range monitorConfig.Artists {
		if !artist.Monitor {
			continue
		}
		log.Printf("\nChecking %s (ID: %d)...", artist.Artist, artist.ID)

		shows, err := catalogManager.GetShowsForArtist(artist.Artist)
		if err != nil {
			log.Printf("Error getting shows for %s: %v", artist.Artist, err)
			continue
		}

		// Check all shows to find missing ones (no date restriction)
		var newShows []catalog.ShowContainer

		for _, show := range shows {
			// Check if show is not already downloaded
			if !isShowDownloaded(artist.Artist, show.ContainerID, showsData) {
				newShows = append(newShows, show)
			}
		}

		if len(newShows) == 0 {
			log.Printf("No new shows found for %s", artist.Artist)
			continue
		}

		log.Printf("Found %d new shows for %s", len(newShows), artist.Artist)

		// Download new shows
		for _, show := range newShows {
			log.Printf("Downloading: %s - %s, %s %s",
				show.PerformanceDateShort, show.VenueName, show.VenueCity, show.VenueState)

			// Create API client only when we need to download
			apiClient := api.NewSafeAPIClient()
			err := apiClient.Authenticate(config.Email, config.Password)
			if err != nil {
				log.Printf("Authentication failed for download: %v", err)
				continue
			}

			releaseURL := fmt.Sprintf("https://play.nugs.net/release/%d", show.ContainerID)

			// Create artist-specific output directory
			artistPath := filepath.Join(config.OutPath, sanitizeFilename(artist.Artist))

			// Run nugs-dl command
			cmd := exec.Command("bin/nugs-dl",
				"-f", fmt.Sprintf("%d", config.Format),
				"-o", artistPath,
				releaseURL)

			output, err := cmd.CombinedOutput()
			if err != nil {
				log.Printf("Error downloading show %d: %v\nOutput: %s\n",
					show.ContainerID, err, string(output))
				continue
			}

			log.Printf("Successfully downloaded show %d", show.ContainerID)

			// Rsync to tootie
			err = rsyncToTootie(artistPath, artist.ArtistFolder)
			if err != nil {
				log.Printf("Error syncing show %d to tootie: %v", show.ContainerID, err)
				continue
			}

			log.Printf("Successfully synced show %d to tootie", show.ContainerID)

			// Clean up local files
			err = cleanupLocalFiles(artistPath)
			if err != nil {
				log.Printf("Warning: Could not cleanup local files: %v", err)
			}

			// Mark as downloaded
			markShowDownloaded(artist.Artist, show.ContainerID, showsData)
		}
	}

	// Save updated shows data
	saveShowsData(showsData)
	log.Println("\nAll checks complete!")
}

func loadConfig(filename string) (*models.Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config models.Config
	err = json.Unmarshal(data, &config)
	return &config, err
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
		// File doesn't exist, return empty struct
		return &models.ShowsData{
			Artists: make(map[string]models.ArtistShowData),
		}
	}

	var shows models.ShowsData
	if err := json.Unmarshal(data, &shows); err != nil {
		log.Printf("Warning: failed to unmarshal shows data: %v", err)
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

func saveShowsData(shows *models.ShowsData) {
	data, err := json.MarshalIndent(shows, "", "  ")
	if err != nil {
		fmt.Printf("Warning: failed to marshal shows data: %v\n", err)
		return
	}

	// Ensure the data directory exists
	if err := os.MkdirAll("data", 0755); err != nil {
		fmt.Printf("Warning: failed to create data directory: %v\n", err)
		return
	}

	// Create a temporary file in the data directory
	tempFile, err := os.CreateTemp("data", "shows.json.tmp.*")
	if err != nil {
		fmt.Printf("Warning: failed to create temp file: %v\n", err)
		return
	}
	tempPath := tempFile.Name()

	// Write data to temp file
	if _, err := tempFile.Write(data); err != nil {
		tempFile.Close()
		os.Remove(tempPath)
		fmt.Printf("Warning: failed to write to temp file: %v\n", err)
		return
	}

	// Flush data to disk
	if err := tempFile.Sync(); err != nil {
		tempFile.Close()
		os.Remove(tempPath)
		fmt.Printf("Warning: failed to sync temp file: %v\n", err)
		return
	}

	// Close temp file
	if err := tempFile.Close(); err != nil {
		os.Remove(tempPath)
		fmt.Printf("Warning: failed to close temp file: %v\n", err)
		return
	}

	// Set correct permissions
	if err := os.Chmod(tempPath, 0644); err != nil {
		os.Remove(tempPath)
		fmt.Printf("Warning: failed to set permissions on temp file: %v\n", err)
		return
	}

	// Atomically replace the target file
	if err := os.Rename(tempPath, "data/shows.json"); err != nil {
		os.Remove(tempPath)
		fmt.Printf("Warning: failed to rename temp file to final location: %v\n", err)
		return
	}
}

func isShowDownloaded(artistName string, containerID int, shows *models.ShowsData) bool {
	artistData, exists := shows.Artists[artistName]
	if !exists {
		return false
	}

	for _, id := range artistData.Downloaded {
		if id == containerID {
			return true
		}
	}
	return false
}

func markShowDownloaded(artistName string, containerID int, shows *models.ShowsData) {
	if shows.Artists == nil {
		shows.Artists = make(map[string]models.ArtistShowData)
	}

	artistData := shows.Artists[artistName]
	artistData.Downloaded = append(artistData.Downloaded, containerID)
	shows.Artists[artistName] = artistData
}

func rsyncToTootie(localPath, remotePath string) error {
	cmd := exec.Command("rsync", "-avP", "--remove-source-files",
		localPath+"/",
		fmt.Sprintf("tootie:%s/", remotePath))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("rsync failed: %v\nOutput: %s", err, string(output))
	}

	return nil
}

func cleanupLocalFiles(localPath string) error {
	// Remove empty directories after rsync
	cmd := exec.Command("find", localPath, "-type", "d", "-empty", "-delete")
	_, err := cmd.CombinedOutput()
	return err
}

func sanitizeFilename(name string) string {
	// Remove or replace characters that might cause issues in filenames
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "-",
		"?", "-",
		"\"", "-",
		"<", "-",
		">", "-",
		"|", "-",
	)
	return replacer.Replace(name)
}
