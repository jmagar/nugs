package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

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

type Artist struct {
	ID   int
	Name string
}

type ImportStats struct {
	TotalShows      int64
	ProcessedShows  int64
	ImportedShows   int64
	SkippedShows    int64
	ErrorShows      int64
	TotalArtists    int64
	ImportedArtists int64
}

func main() {
	log.Println("Starting Nugs.net catalog import...")

	// Open database
	db, err := sql.Open("sqlite3", "data/nugs_api.db?_foreign_keys=on")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// Optimize SQLite for bulk inserts
	optimizeDatabase(db)

	stats := &ImportStats{}

	// Phase 1: Import artists
	log.Println("Phase 1: Extracting and importing artists...")
	if err := importArtists(db, stats); err != nil {
		log.Fatal("Failed to import artists:", err)
	}

	// Phase 2: Import shows
	log.Println("Phase 2: Importing shows...")
	if err := importShows(db, stats); err != nil {
		log.Fatal("Failed to import shows:", err)
	}

	// Print final statistics
	printFinalStats(stats)
}

func generateSlug(name string) string {
	// Convert to lowercase
	slug := strings.ToLower(name)
	// Replace spaces and special characters with hyphens
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	slug = reg.ReplaceAllString(slug, "-")
	// Remove leading/trailing hyphens
	slug = strings.Trim(slug, "-")
	return slug
}

func optimizeDatabase(db *sql.DB) {
	optimizations := []string{
		"PRAGMA synchronous = OFF",
		"PRAGMA journal_mode = WAL",
		"PRAGMA cache_size = 10000",
		"PRAGMA temp_store = MEMORY",
		"PRAGMA mmap_size = 268435456", // 256MB mmap
	}

	for _, pragma := range optimizations {
		if _, err := db.Exec(pragma); err != nil {
			log.Printf("Warning: Failed to set %s: %v", pragma, err)
		}
	}
}

func importArtists(db *sql.DB, stats *ImportStats) error {
	// Open and parse JSON file
	file, err := os.Open("data/all_shows_catalog.json")
	if err != nil {
		return fmt.Errorf("failed to open catalog file: %v", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)

	// Find the Response object and containers array
	var response CatalogResponse
	if err := decoder.Decode(&response); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}

	stats.TotalShows = int64(len(response.Response.Containers))
	log.Printf("Total shows to process: %d", stats.TotalShows)

	// Extract unique artists
	artistMap := make(map[int]string)

	log.Println("Extracting unique artists...")
	for i, containerRaw := range response.Response.Containers {
		var show Show
		if err := json.Unmarshal(containerRaw, &show); err != nil {
			log.Printf("Warning: Failed to parse show %d: %v", i, err)
			continue
		}

		if show.ArtistID > 0 && show.ArtistName != "" {
			artistMap[show.ArtistID] = show.ArtistName
		}

		// Progress update every 1000 shows
		if i > 0 && i%1000 == 0 {
			fmt.Printf("\rExtracting artists... %d/%d (%.1f%%)", i, len(response.Response.Containers), float64(i)/float64(len(response.Response.Containers))*100)
		}
	}

	fmt.Printf("\rExtracting artists... %d/%d (100.0%%)\n", len(response.Response.Containers), len(response.Response.Containers))
	stats.TotalArtists = int64(len(artistMap))
	log.Printf("Found %d unique artists", len(artistMap))

	// Batch insert artists
	return batchInsertArtists(db, artistMap, stats)
}

func batchInsertArtists(db *sql.DB, artistMap map[int]string, stats *ImportStats) error {
	// Prepare statement
	stmt, err := db.Prepare(`
		INSERT OR REPLACE INTO artists (id, nugs_id, name, slug, monitored, show_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, false, 0, datetime('now'), datetime('now'))
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare artist statement: %v", err)
	}
	defer stmt.Close()

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}

	batchSize := 500
	count := 0

	log.Println("Inserting artists...")
	for id, name := range artistMap {
		slug := generateSlug(name)
		_, err := tx.Stmt(stmt).Exec(id, id, name, slug) // id, nugs_id, name, slug
		if err != nil {
			log.Printf("Warning: Failed to insert artist %d (%s): %v", id, name, err)
			continue
		}

		atomic.AddInt64(&stats.ImportedArtists, 1)
		count++

		// Commit batch
		if count%batchSize == 0 {
			if err := tx.Commit(); err != nil {
				return fmt.Errorf("failed to commit artist batch: %v", err)
			}

			// Start new transaction
			tx, err = db.Begin()
			if err != nil {
				return fmt.Errorf("failed to begin new transaction: %v", err)
			}

			fmt.Printf("\rInserting artists... %d/%d (%.1f%%)", count, len(artistMap), float64(count)/float64(len(artistMap))*100)
		}
	}

	// Commit remaining
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit final artist batch: %v", err)
	}

	fmt.Printf("\rInserting artists... %d/%d (100.0%%)\n", len(artistMap), len(artistMap))
	log.Printf("Imported %d artists", stats.ImportedArtists)

	return nil
}

func importShows(db *sql.DB, stats *ImportStats) error {
	// Open and parse JSON file again (streaming)
	file, err := os.Open("data/all_shows_catalog.json")
	if err != nil {
		return fmt.Errorf("failed to open catalog file: %v", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)

	// Parse response structure
	var response CatalogResponse
	if err := decoder.Decode(&response); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}

	// Prepare show insert statement
	stmt, err := db.Prepare(`
		INSERT OR REPLACE INTO shows (
			id, artist_id, container_id, venue_name, venue_city,
			venue_state, performance_date, performance_date_formatted,
			performance_date_short, container_info, availability_type,
			availability_type_str, active_state, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare show statement: %v", err)
	}
	defer stmt.Close()

	return batchInsertShows(db, stmt, response.Response.Containers, stats)
}

func batchInsertShows(db *sql.DB, stmt *sql.Stmt, containers []json.RawMessage, stats *ImportStats) error {
	batchSize := 500
	processed := int64(0)

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}

	startTime := time.Now()

	log.Println("Importing shows...")
	for i, containerRaw := range containers {
		var show Show
		if err := json.Unmarshal(containerRaw, &show); err != nil {
			log.Printf("Warning: Failed to parse show %d: %v", i, err)
			atomic.AddInt64(&stats.ErrorShows, 1)
			continue
		}

		// Skip shows without required data
		if show.ContainerID == 0 || show.ArtistID == 0 {
			atomic.AddInt64(&stats.SkippedShows, 1)
			continue
		}

		// Insert show
		_, err := tx.Stmt(stmt).Exec(
			show.ContainerID,              // id
			show.ArtistID,                 // artist_id
			show.ContainerID,              // container_id
			show.VenueName,                // venue_name
			show.VenueCity,                // venue_city
			show.VenueState,               // venue_state
			show.PerformanceDate,          // performance_date
			show.PerformanceDateFormatted, // performance_date_formatted
			show.PerformanceDateShort,     // performance_date_short
			show.ContainerInfo,            // container_info
			show.AvailabilityType,         // availability_type
			show.AvailabilityTypeStr,      // availability_type_str
			show.ActiveState,              // active_state
		)

		if err != nil {
			log.Printf("Warning: Failed to insert show %d: %v", show.ContainerID, err)
			atomic.AddInt64(&stats.ErrorShows, 1)
			continue
		}

		atomic.AddInt64(&stats.ImportedShows, 1)
		processed++

		// Progress and batch commit
		if processed%int64(batchSize) == 0 {
			if err := tx.Commit(); err != nil {
				return fmt.Errorf("failed to commit show batch: %v", err)
			}

			// Start new transaction
			tx, err = db.Begin()
			if err != nil {
				return fmt.Errorf("failed to begin new transaction: %v", err)
			}

			// Progress update
			elapsed := time.Since(startTime)
			rate := float64(processed) / elapsed.Seconds()
			remaining := stats.TotalShows - processed
			eta := time.Duration(float64(remaining)/rate) * time.Second

			fmt.Printf("\rImporting shows... %d/%d (%.1f%%) | %.0f shows/sec | ETA: %v",
				processed, stats.TotalShows,
				float64(processed)/float64(stats.TotalShows)*100,
				rate, eta.Round(time.Second))
		}

		atomic.AddInt64(&stats.ProcessedShows, 1)
	}

	// Commit remaining
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit final show batch: %v", err)
	}

	fmt.Printf("\rImporting shows... %d/%d (100.0%%)\n", processed, stats.TotalShows)

	return nil
}

func printFinalStats(stats *ImportStats) {
	log.Println("\n" + strings.Repeat("=", 50))
	log.Println("IMPORT COMPLETED")
	log.Println(strings.Repeat("=", 50))
	log.Printf("Total shows processed: %d", stats.ProcessedShows)
	log.Printf("Shows imported: %d", stats.ImportedShows)
	log.Printf("Shows skipped: %d", stats.SkippedShows)
	log.Printf("Shows with errors: %d", stats.ErrorShows)
	log.Printf("Artists imported: %d", stats.ImportedArtists)
	log.Printf("Success rate: %.1f%%", float64(stats.ImportedShows)/float64(stats.ProcessedShows)*100)
	log.Println(strings.Repeat("=", 50))
}
