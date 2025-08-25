package catalog

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/jmagar/nugs/cron/internal/api"
)

// CatalogManager handles the full Nugs catalog
type CatalogManager struct {
	catalogFile string
	maxAge      time.Duration
}

// ShowContainer represents a show from the catalog
type ShowContainer struct {
	ContainerID              int    `json:"containerID"`
	ArtistName               string `json:"artistName"`
	VenueName                string `json:"venueName"`
	VenueCity                string `json:"venueCity"`
	VenueState               string `json:"venueState"`
	PerformanceDate          string `json:"performanceDate"`
	PerformanceDateShort     string `json:"performanceDateShort"`
	PerformanceDateFormatted string `json:"performanceDateFormatted"`
	ContainerInfo            string `json:"containerInfo"`
	AvailabilityType         int    `json:"availabilityType"`
	AvailabilityTypeStr      string `json:"availabilityTypeStr"`
	ActiveState              string `json:"activeState"`
}

// CatalogResponse represents the full API response
type CatalogResponse struct {
	MethodName                  string `json:"methodName"`
	ResponseAvailabilityCode    int    `json:"responseAvailabilityCode"`
	ResponseAvailabilityCodeStr string `json:"responseAvailabilityCodeStr"`
	Response                    struct {
		Containers []ShowContainer `json:"containers"`
	} `json:"Response"`
}

// CatalogCache represents our cached catalog with metadata
type CatalogCache struct {
	LastUpdate    string                     `json:"last_update"`
	TotalShows    int                        `json:"total_shows"`
	TotalArtists  int                        `json:"total_artists"`
	ShowsByArtist map[string][]ShowContainer `json:"shows_by_artist"`
	AllShows      []ShowContainer            `json:"all_shows"`
}

// NewCatalogManager creates a new catalog manager
func NewCatalogManager() *CatalogManager {
	return &CatalogManager{
		catalogFile: "data/catalog_cache.json",
		maxAge:      24 * time.Hour, // Refresh daily
	}
}

// GetCatalog returns the current catalog, refreshing if needed
func (cm *CatalogManager) GetCatalog() (*CatalogCache, error) {
	// Check if we need to refresh
	if cm.needsRefresh() {
		log.Println("Catalog needs refresh, fetching from API...")
		if err := cm.refreshCatalog(); err != nil {
			log.Printf("Failed to refresh catalog: %v", err)
			// Try to load existing cache even if refresh failed
		}
	}

	return cm.loadCatalogCache()
}

// GetShowsForArtist returns all shows for a specific artist
func (cm *CatalogManager) GetShowsForArtist(artistName string) ([]ShowContainer, error) {
	catalog, err := cm.GetCatalog()
	if err != nil {
		return nil, err
	}

	shows, exists := catalog.ShowsByArtist[artistName]
	if !exists {
		return []ShowContainer{}, nil
	}

	return shows, nil
}

// GetAllArtists returns a list of all artists in the catalog
func (cm *CatalogManager) GetAllArtists() ([]string, error) {
	catalog, err := cm.GetCatalog()
	if err != nil {
		return nil, err
	}

	artists := make([]string, 0, len(catalog.ShowsByArtist))
	for artist := range catalog.ShowsByArtist {
		artists = append(artists, artist)
	}

	sort.Strings(artists)
	return artists, nil
}

// GetShowByID finds a specific show by container ID
func (cm *CatalogManager) GetShowByID(containerID int) (*ShowContainer, error) {
	catalog, err := cm.GetCatalog()
	if err != nil {
		return nil, err
	}

	for _, show := range catalog.AllShows {
		if show.ContainerID == containerID {
			return &show, nil
		}
	}

	return nil, fmt.Errorf("show with ID %d not found", containerID)
}

// GetCatalogStats returns statistics about the catalog
func (cm *CatalogManager) GetCatalogStats() (*CatalogCache, error) {
	return cm.GetCatalog()
}

// ForceRefresh forces a catalog refresh regardless of age
func (cm *CatalogManager) ForceRefresh() error {
	log.Println("Forcing catalog refresh...")
	return cm.refreshCatalog()
}

// needsRefresh checks if the catalog cache is stale
func (cm *CatalogManager) needsRefresh() bool {
	// Check if cache file exists
	if _, err := os.Stat(cm.catalogFile); os.IsNotExist(err) {
		return true
	}

	// Check file age
	fileInfo, err := os.Stat(cm.catalogFile)
	if err != nil {
		return true
	}

	age := time.Since(fileInfo.ModTime())
	return age > cm.maxAge
}

// refreshCatalog fetches the full catalog from the API
func (cm *CatalogManager) refreshCatalog() error {
	log.Println("Fetching full catalog from Nugs.net...")

	// Use our safe API client for consistency (even though this endpoint doesn't need auth)
	apiClient := api.NewSafeAPIClient()
	body, err := apiClient.GetFullCatalog()
	if err != nil {
		return fmt.Errorf("failed to fetch catalog: %v", err)
	}

	// Parse the response
	var response CatalogResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse catalog response: %v", err)
	}

	log.Printf("Fetched %d shows from API", len(response.Response.Containers))

	// Organize shows by artist
	showsByArtist := make(map[string][]ShowContainer)
	artistCounts := make(map[string]int)

	for _, show := range response.Response.Containers {
		artistName := strings.TrimSpace(show.ArtistName)
		showsByArtist[artistName] = append(showsByArtist[artistName], show)
		artistCounts[artistName]++
	}

	// Sort shows for each artist by date (newest first)
	for artistName := range showsByArtist {
		shows := showsByArtist[artistName]
		sort.Slice(shows, func(i, j int) bool {
			// Try to parse dates and sort by performance date
			dateI, errI := time.Parse("1/2/2006", shows[i].PerformanceDate)
			dateJ, errJ := time.Parse("1/2/2006", shows[j].PerformanceDate)

			if errI != nil || errJ != nil {
				// Fall back to container ID if date parsing fails
				return shows[i].ContainerID > shows[j].ContainerID
			}

			return dateI.After(dateJ)
		})
		showsByArtist[artistName] = shows
	}

	// Create cache structure
	cache := CatalogCache{
		LastUpdate:    time.Now().Format(time.RFC3339),
		TotalShows:    len(response.Response.Containers),
		TotalArtists:  len(showsByArtist),
		ShowsByArtist: showsByArtist,
		AllShows:      response.Response.Containers,
	}

	// Save to cache file
	if err := cm.saveCatalogCache(&cache); err != nil {
		return fmt.Errorf("failed to save catalog cache: %v", err)
	}

	log.Printf("Catalog updated: %d shows from %d artists", cache.TotalShows, cache.TotalArtists)
	return nil
}

// loadCatalogCache loads the cached catalog from disk
func (cm *CatalogManager) loadCatalogCache() (*CatalogCache, error) {
	data, err := os.ReadFile(cm.catalogFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read catalog cache: %v", err)
	}

	var cache CatalogCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("failed to parse catalog cache: %v", err)
	}

	return &cache, nil
}

// saveCatalogCache saves the catalog cache to disk
func (cm *CatalogManager) saveCatalogCache(cache *CatalogCache) error {
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cm.catalogFile, data, 0644)
}

// PrintCatalogStats prints statistics about the catalog
func (cm *CatalogManager) PrintCatalogStats() {
	cache, err := cm.GetCatalog()
	if err != nil {
		log.Printf("Error loading catalog: %v", err)
		return
	}

	fmt.Println("=== Nugs.net Catalog Statistics ===")
	fmt.Printf("Last Updated: %s\n", cache.LastUpdate)
	fmt.Printf("Total Shows: %d\n", cache.TotalShows)
	fmt.Printf("Total Artists: %d\n", cache.TotalArtists)
	fmt.Println("")

	// Show top artists by show count
	type artistStat struct {
		Name      string
		ShowCount int
	}

	var artistStats []artistStat
	for name, shows := range cache.ShowsByArtist {
		artistStats = append(artistStats, artistStat{
			Name:      name,
			ShowCount: len(shows),
		})
	}

	sort.Slice(artistStats, func(i, j int) bool {
		return artistStats[i].ShowCount > artistStats[j].ShowCount
	})

	fmt.Println("Top 10 Artists by Show Count:")
	for i, stat := range artistStats {
		if i >= 10 {
			break
		}
		fmt.Printf("%2d. %s (%d shows)\n", i+1, stat.Name, stat.ShowCount)
	}
}

// RunCLI provides the CLI functionality for catalog_manager
func RunCLI() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: catalog_manager <command>")
		fmt.Println("Commands:")
		fmt.Println("  stats    - Show catalog statistics")
		fmt.Println("  refresh  - Force catalog refresh")
		fmt.Println("  artist <name> - Show all shows for an artist")
		return
	}

	cm := NewCatalogManager()
	command := os.Args[1]

	switch command {
	case "stats":
		cm.PrintCatalogStats()
	case "refresh":
		if err := cm.ForceRefresh(); err != nil {
			log.Fatal("Refresh failed:", err)
		}
		fmt.Println("Catalog refreshed successfully")
	case "artist":
		if len(os.Args) < 3 {
			fmt.Println("Usage: catalog_manager artist <artist_name>")
			return
		}
		artistName := strings.Join(os.Args[2:], " ")
		shows, err := cm.GetShowsForArtist(artistName)
		if err != nil {
			log.Fatal("Error:", err)
		}
		fmt.Printf("Shows for %s:\n", artistName)
		for _, show := range shows {
			fmt.Printf("  %d - %s at %s, %s %s\n",
				show.ContainerID, show.PerformanceDateShort,
				show.VenueName, show.VenueCity, show.VenueState)
		}
	default:
		fmt.Printf("Unknown command: %s\n", command)
	}
}
