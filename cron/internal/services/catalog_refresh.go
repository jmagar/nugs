package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jmagar/nugs/cron/internal/models"
	_ "github.com/mattn/go-sqlite3"
)

type CatalogRefreshService struct {
	DB         *sql.DB
	JobManager *models.JobManager
}

type RefreshResult struct {
	TotalShows      int64  `json:"total_shows"`
	ProcessedShows  int64  `json:"processed_shows"`
	ImportedShows   int64  `json:"imported_shows"`
	SkippedShows    int64  `json:"skipped_shows"`
	ErrorShows      int64  `json:"error_shows"`
	TotalArtists    int64  `json:"total_artists"`
	ImportedArtists int64  `json:"imported_artists"`
	Duration        string `json:"duration"`
}

type CatalogResponse struct {
	Response struct {
		Containers []json.RawMessage `json:"containers"`
	} `json:"Response"`
}

type Show struct {
	ContainerID              int    `json:"containerID"`
	ArtistID                 int    `json:"artistID"`
	ArtistName               string `json:"artistName"`
	LicensorName             string `json:"licensorName"`
	VenueName                string `json:"venueName"`
	VenueCity                string `json:"venueCity"`
	VenueState               string `json:"venueState"`
	PerformanceDate          string `json:"performanceDate"`
	PerformanceDateFormatted string `json:"performanceDateFormatted"`
	PerformanceDateShort     string `json:"performanceDateShort"`
	ContainerInfo            string `json:"containerInfo"`
	AvailabilityType         int    `json:"availabilityType"`
	AvailabilityTypeStr      string `json:"availabilityTypeStr"`
	ActiveState              string `json:"activeState"`
	PageURL                  string `json:"pageURL"`
	ContainerCode            string `json:"containerCode"`
	ExtImage                 string `json:"extImage"`
}

type CatalogCache struct {
	LastUpdate    string            `json:"last_update"`
	TotalShows    int               `json:"total_shows"`
	TotalArtists  int               `json:"total_artists"`
	ShowsByArtist map[string][]Show `json:"shows_by_artist"`
	AllShows      []Show            `json:"all_shows"`
}

func NewCatalogRefreshService(db *sql.DB, jobManager *models.JobManager) *CatalogRefreshService {
	return &CatalogRefreshService{
		DB:         db,
		JobManager: jobManager,
	}
}

func (s *CatalogRefreshService) StartRefresh(force bool) *models.Job {
	job := s.JobManager.CreateJob(models.JobTypeCatalogRefresh)

	// Start refresh in background
	go s.runRefresh(job, force)

	return job
}

func (s *CatalogRefreshService) runRefresh(job *models.Job, force bool) {
	startTime := time.Now()

	if err := s.JobManager.UpdateJob(job.ID, func(j *models.Job) {
		j.Status = models.JobStatusRunning
		j.StartedAt = startTime
		j.Message = "Starting catalog refresh..."
	}); err != nil {
		log.Printf("Error updating job status: %v", err)
		return
	}

	// Check if we should skip refresh based on last update time
	if !force {
		lastRefresh, err := s.getLastRefreshTime()
		if err == nil && time.Since(lastRefresh) < 4*time.Hour {
			if err := s.JobManager.UpdateJob(job.ID, func(j *models.Job) {
				j.Status = models.JobStatusCompleted
				j.Progress = 100
				j.Message = "Catalog is already up to date"
				completedAt := time.Now()
				j.CompletedAt = &completedAt
			}); err != nil {
				log.Printf("Error updating job status: %v", err)
			}
			return
		}
	}

	result := &RefreshResult{}

	// Use existing catalog_manager command
	err := s.refreshUsingCatalogManager(job, result)
	if err != nil {
		if updateErr := s.JobManager.UpdateJob(job.ID, func(j *models.Job) {
			j.Status = models.JobStatusFailed
			j.Error = err.Error()
			j.Message = "Catalog refresh failed"
			completedAt := time.Now()
			j.CompletedAt = &completedAt
		}); updateErr != nil {
			log.Printf("Error updating job status: %v", updateErr)
		}
		return
	}

	// Update job with final results
	result.Duration = time.Since(startTime).String()
	completedAt := time.Now()

	if err := s.JobManager.UpdateJob(job.ID, func(j *models.Job) {
		j.Status = models.JobStatusCompleted
		j.Progress = 100
		j.Message = fmt.Sprintf("Refresh completed: %d shows from %d artists", result.TotalShows, result.TotalArtists)
		j.Result = result
		j.CompletedAt = &completedAt
	}); err != nil {
		log.Printf("Warning: failed to update job status: %v", err)
	}

	// Update last refresh time
	if err := s.setLastRefreshTime(time.Now()); err != nil {
		log.Printf("Warning: failed to update last refresh time: %v", err)
	}
}

func (s *CatalogRefreshService) refreshUsingCatalogManager(job *models.Job, result *RefreshResult) error {
	// Update progress
	if err := s.JobManager.UpdateJob(job.ID, func(j *models.Job) {
		j.Progress = 10
		j.Message = "Fetching catalog from Nugs.net..."
	}); err != nil {
		log.Printf("Warning: failed to update job status: %v", err)
	}

	// Check for cancellation
	select {
	case <-job.Cancel:
		return fmt.Errorf("refresh cancelled")
	default:
	}

	// Execute catalog_manager refresh
	cmd := exec.Command("./bin/catalog_manager", "refresh")
	cmd.Dir = "/home/jmagar/code/nugs/cron"

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("catalog_manager failed: %v, output: %s", err, string(output))
	}

	// Update progress
	if err := s.JobManager.UpdateJob(job.ID, func(j *models.Job) {
		j.Progress = 50
		j.Message = "Processing catalog data..."
	}); err != nil {
		log.Printf("Warning: failed to update job status: %v", err)
	}

	// Read and process catalog cache file
	outputStr := string(output)
	if strings.Contains(outputStr, "Catalog refreshed successfully") {
		err = s.importCatalogData(job, result)
		if err != nil {
			return fmt.Errorf("failed to import catalog data: %v", err)
		}
	}

	// Update final progress
	if err := s.JobManager.UpdateJob(job.ID, func(j *models.Job) {
		j.Progress = 90
		j.Message = "Finalizing catalog update..."
	}); err != nil {
		log.Printf("Warning: failed to update job status: %v", err)
	}

	return nil
}

func (s *CatalogRefreshService) importCatalogData(job *models.Job, result *RefreshResult) error {
	// Read the catalog cache file
	catalogPath := filepath.Join("data", "catalog_cache.json")
	data, err := os.ReadFile(catalogPath)
	if err != nil {
		return fmt.Errorf("failed to read catalog cache file: %v", err)
	}

	// Parse the JSON
	var catalog CatalogCache
	err = json.Unmarshal(data, &catalog)
	if err != nil {
		return fmt.Errorf("failed to parse catalog JSON: %v", err)
	}

	// Update progress
	if err := s.JobManager.UpdateJob(job.ID, func(j *models.Job) {
		j.Progress = 60
		j.Message = "Clearing existing data..."
	}); err != nil {
		log.Printf("Warning: failed to update job status: %v", err)
	}

	// Clear existing data (removing seed data)
	_, err = s.DB.Exec("DELETE FROM shows")
	if err != nil {
		return fmt.Errorf("failed to clear shows table: %v", err)
	}

	_, err = s.DB.Exec("DELETE FROM artists")
	if err != nil {
		return fmt.Errorf("failed to clear artists table: %v", err)
	}

	// Update progress
	if err := s.JobManager.UpdateJob(job.ID, func(j *models.Job) {
		j.Progress = 70
		j.Message = "Importing artists..."
	}); err != nil {
		log.Printf("Warning: failed to update job status: %v", err)
	}

	// Extract and insert unique artists
	artistMap := make(map[string]int)
	artistCounter := 1

	for artistName := range catalog.ShowsByArtist {
		if artistName == "" {
			continue
		}

		// Create slug from artist name
		slug := strings.ToLower(strings.ReplaceAll(artistName, " ", "-"))
		slug = strings.ReplaceAll(slug, "&", "and")

		// Insert artist
		_, err = s.DB.Exec(`
			INSERT INTO artists (id, name, slug, show_count, is_active, nugs_artist_id, created_at, updated_at) 
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			artistCounter, artistName, slug, len(catalog.ShowsByArtist[artistName]), true, nil, time.Now(), time.Now())

		if err != nil {
			log.Printf("Failed to insert artist %s: %v", artistName, err)
			continue
		}

		artistMap[artistName] = artistCounter
		artistCounter++
	}

	// Update progress
	if err := s.JobManager.UpdateJob(job.ID, func(j *models.Job) {
		j.Progress = 80
		j.Message = "Importing shows..."
	}); err != nil {
		log.Printf("Warning: failed to update job status: %v", err)
	}

	// Insert shows
	showCounter := 0
	for artistName, shows := range catalog.ShowsByArtist {
		artistID, exists := artistMap[artistName]
		if !exists {
			continue
		}

		for _, show := range shows {
			// Parse the performance date
			performanceDate, err := time.Parse("1/2/2006", show.PerformanceDate)
			if err != nil {
				// Try alternative format
				performanceDate, err = time.Parse("2006/01/02", show.PerformanceDateFormatted)
				if err != nil {
					log.Printf("Failed to parse date for show %d: %v", show.ContainerID, err)
					performanceDate = time.Now()
				}
			}

			_, err = s.DB.Exec(`
				INSERT INTO shows (container_id, artist_id, date, venue, city, state, country, 
					duration_minutes, is_available, created_at, updated_at) 
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				show.ContainerID, artistID, performanceDate, show.VenueName,
				show.VenueCity, show.VenueState, "USA", 0,
				show.ActiveState == "AVAILABLE", time.Now(), time.Now())

			if err != nil {
				log.Printf("Failed to insert show %d: %v", show.ContainerID, err)
				continue
			}

			showCounter++
		}
	}

	// Update result statistics
	result.TotalShows = int64(catalog.TotalShows)
	result.ImportedShows = int64(showCounter)
	result.TotalArtists = int64(len(artistMap))
	result.ImportedArtists = int64(len(artistMap))

	log.Printf("Successfully imported %d shows from %d artists", showCounter, len(artistMap))
	return nil
}

func (s *CatalogRefreshService) getLastRefreshTime() (time.Time, error) {
	var lastRefresh string
	err := s.DB.QueryRow(`
		SELECT value FROM system_config 
		WHERE key = 'last_catalog_refresh'
	`).Scan(&lastRefresh)

	if err != nil {
		return time.Time{}, err
	}

	return time.Parse(time.RFC3339, lastRefresh)
}

func (s *CatalogRefreshService) setLastRefreshTime(t time.Time) error {
	_, err := s.DB.Exec(`
		INSERT OR REPLACE INTO system_config (key, value, description, updated_at)
		VALUES ('last_catalog_refresh', ?, 'Last catalog refresh timestamp', ?)
	`, t.Format(time.RFC3339), time.Now())

	return err
}
