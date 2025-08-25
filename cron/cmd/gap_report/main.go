package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/jmagar/nugs/cron/internal/catalog"
	"github.com/jmagar/nugs/cron/internal/models"
)

type MissingShow struct {
	ContainerID int    `json:"container_id"`
	Date        string `json:"date"`
	Venue       string `json:"venue"`
	City        string `json:"city"`
	State       string `json:"state"`
}

type GapReport struct {
	Artist          string        `json:"artist"`
	ArtistID        int           `json:"artist_id"`
	TotalAvailable  int           `json:"total_available"`
	TotalDownloaded int           `json:"total_downloaded"`
	CompletionPct   float64       `json:"completion_pct"`
	MissingShows    []MissingShow `json:"missing_shows"`
	MissingCount    int           `json:"missing_count"`
}

type ReportSummary struct {
	TotalArtists      int     `json:"total_artists"`
	TotalShowsHave    int     `json:"total_shows_have"`
	TotalShowsAvail   int     `json:"total_shows_available"`
	OverallCompletion float64 `json:"overall_completion"`
	TotalMissing      int     `json:"total_missing"`
}

func main() {
	// Command line flags
	var (
		format     = flag.String("format", "terminal", "Output format: terminal, html, csv, json")
		sortBy     = flag.String("sort", "artist", "Sort by: artist, completion, missing, total")
		artistName = flag.String("artist", "", "Generate report for specific artist only")
		minMissing = flag.Int("min-missing", 0, "Only show artists with at least N missing shows")
		outputFile = flag.String("output", "", "Output file (default: stdout)")
	)
	flag.Parse()

	// Load shows data
	log.Println("Loading shows data from data/shows.json...")
	showsData, err := loadShowsData()
	if err != nil {
		log.Fatal("Error loading shows data:", err)
	}
	log.Printf("Loaded shows data for %d artists", len(showsData.Artists))

	// Load monitor config to get monitored artists
	log.Println("Loading monitor config...")
	monitorConfig, err := loadMonitorConfig("configs/monitor_config.json")
	if err != nil {
		log.Fatal("Error loading monitor config:", err)
	}
	log.Printf("Monitor config loaded with %d artists", len(monitorConfig.Artists))

	// Create catalog manager and pre-load catalog
	log.Println("Initializing catalog manager...")
	catalogManager := catalog.NewCatalogManager()

	log.Println("Pre-loading catalog for fast lookups...")
	catalogData, err := catalogManager.GetCatalog()
	if err != nil {
		log.Fatal("Error loading catalog:", err)
	}
	log.Printf("Catalog loaded: %d total shows", len(catalogData.AllShows))

	// Create fast lookup map
	showMap := make(map[int]*catalog.ShowContainer)
	for i := range catalogData.AllShows {
		show := &catalogData.AllShows[i]
		showMap[show.ContainerID] = show
	}
	log.Printf("Created fast lookup map for %d shows", len(showMap))

	// Generate reports
	log.Println("Starting report generation...")
	var reports []GapReport
	var summary ReportSummary

	processedCount := 0
	for _, artistConfig := range monitorConfig.Artists {
		if !artistConfig.Monitor {
			continue
		}

		// Filter by specific artist if requested
		if *artistName != "" && !strings.Contains(strings.ToLower(artistConfig.Artist), strings.ToLower(*artistName)) {
			continue
		}

		processedCount++
		log.Printf("Processing artist %d: %s", processedCount, artistConfig.Artist)

		// Get artist data from shows.json
		artistData, exists := showsData.Artists[artistConfig.Artist]
		if !exists {
			log.Printf("Warning: No show data found for monitored artist: %s", artistConfig.Artist)
			continue
		}

		log.Printf("  Found %d available shows, %d downloaded, %d missing",
			len(artistData.Available), len(artistData.Downloaded), len(artistData.Missing))

		// Get missing shows with venue details using fast lookup
		var missingShows []MissingShow
		for _, showID := range artistData.Missing {
			// Fast lookup from pre-loaded map
			show, exists := showMap[showID]
			if !exists {
				log.Printf("Warning: Could not find show %d in catalog", showID)
				continue
			}

			missingShows = append(missingShows, MissingShow{
				ContainerID: showID,
				Date:        show.PerformanceDateShort,
				Venue:       show.VenueName,
				City:        show.VenueCity,
				State:       show.VenueState,
			})
		}

		completionPct := 0.0
		if len(artistData.Available) > 0 {
			completionPct = float64(len(artistData.Downloaded)) / float64(len(artistData.Available)) * 100
		}

		report := GapReport{
			Artist:          artistConfig.Artist,
			ArtistID:        artistConfig.ID,
			TotalAvailable:  len(artistData.Available),
			TotalDownloaded: len(artistData.Downloaded),
			CompletionPct:   completionPct,
			MissingShows:    missingShows,
			MissingCount:    len(missingShows),
		}

		// Apply minimum missing filter
		if report.MissingCount >= *minMissing {
			reports = append(reports, report)
		}

		// Update summary
		summary.TotalShowsHave += len(artistData.Downloaded)
		summary.TotalShowsAvail += len(artistData.Available)
		summary.TotalMissing += len(artistData.Missing)
	}

	summary.TotalArtists = len(reports)
	if summary.TotalShowsAvail > 0 {
		summary.OverallCompletion = float64(summary.TotalShowsHave) / float64(summary.TotalShowsAvail) * 100
	}

	log.Printf("Generated reports for %d artists", len(reports))
	log.Printf("Summary: %d shows have, %d shows available, %.1f%% completion",
		summary.TotalShowsHave, summary.TotalShowsAvail, summary.OverallCompletion)

	// Sort reports
	log.Printf("Sorting reports by %s...", *sortBy)
	sortReports(reports, *sortBy)

	// Generate output
	log.Printf("Generating %s output...", *format)
	switch *format {
	case "html":
		log.Println("Generating HTML content...")
		html := generateHTMLContent(reports, summary)
		log.Printf("Generated HTML content: %d bytes", len(html))
		if *outputFile != "" {
			log.Printf("Writing HTML to file: %s", *outputFile)
			err := os.WriteFile(*outputFile, []byte(html), 0644)
			if err != nil {
				log.Fatal("Error writing HTML file:", err)
			}
			fmt.Printf("Modern HTML dashboard written to: %s\n", *outputFile)
		} else {
			fmt.Print(html)
		}
	case "json":
		output := map[string]interface{}{
			"summary": summary,
			"reports": reports,
		}
		jsonData, _ := json.MarshalIndent(output, "", "  ")
		if *outputFile != "" {
			err := os.WriteFile(*outputFile, jsonData, 0644)
			if err != nil {
				log.Fatal("Error writing JSON file:", err)
			}
			fmt.Printf("JSON report written to: %s\n", *outputFile)
		} else {
			fmt.Print(string(jsonData))
		}
	case "csv":
		generateCSVOutput(reports, summary, *outputFile)
	default:
		printTerminalOutput(reports, summary)
	}
}

func printTerminalOutput(reports []GapReport, summary ReportSummary) {
	fmt.Println("🎵 Nugs Collection Gap Report")
	fmt.Println("=" + strings.Repeat("=", 50))
	fmt.Printf("📊 Summary: %d artists monitored\n", summary.TotalArtists)
	fmt.Printf("✅ Shows downloaded: %d\n", summary.TotalShowsHave)
	fmt.Printf("📀 Shows available: %d\n", summary.TotalShowsAvail)
	fmt.Printf("📈 Overall completion: %.1f%%\n", summary.OverallCompletion)
	fmt.Printf("❌ Missing shows: %d\n", summary.TotalMissing)
	fmt.Println()

	for _, report := range reports {
		fmt.Printf("🎤 %s\n", report.Artist)
		fmt.Printf("   Downloaded: %d/%d (%.1f%% complete)\n",
			report.TotalDownloaded, report.TotalAvailable, report.CompletionPct)
		fmt.Printf("   Missing: %d shows\n", len(report.MissingShows))

		if len(report.MissingShows) > 0 && len(report.MissingShows) <= 20 {
			for _, missing := range report.MissingShows {
				fmt.Printf("     • %s - %s, %s %s (#%d)\n",
					missing.Date, missing.Venue, missing.City, missing.State, missing.ContainerID)
			}
		} else if len(report.MissingShows) > 20 {
			fmt.Printf("     ... %d missing shows (use --format html for full list)\n", len(report.MissingShows))
		}
		fmt.Println()
	}
}

func generateCSVOutput(reports []GapReport, summary ReportSummary, outputFile string) {
	var output strings.Builder

	// CSV Header
	output.WriteString("Artist,Total Available,Total Downloaded,Completion %,Missing Count,Missing Show IDs\n")

	// Data rows
	for _, report := range reports {
		var missingIDs []string
		for _, missing := range report.MissingShows {
			missingIDs = append(missingIDs, fmt.Sprintf("%d", missing.ContainerID))
		}

		output.WriteString(fmt.Sprintf("%s,%d,%d,%.1f,%d,\"%s\"\n",
			report.Artist,
			report.TotalAvailable,
			report.TotalDownloaded,
			report.CompletionPct,
			len(report.MissingShows),
			strings.Join(missingIDs, ",")))
	}

	if outputFile != "" {
		err := os.WriteFile(outputFile, []byte(output.String()), 0644)
		if err != nil {
			log.Fatal("Error writing CSV file:", err)
		}
		fmt.Printf("CSV report written to: %s\n", outputFile)
	} else {
		fmt.Print(output.String())
	}
}

func sortReports(reports []GapReport, sortBy string) {
	sort.Slice(reports, func(i, j int) bool {
		switch sortBy {
		case "completion":
			return reports[j].CompletionPct < reports[i].CompletionPct // Lowest completion first
		case "missing":
			return reports[i].MissingCount > reports[j].MissingCount // Most missing first
		case "total":
			return reports[i].TotalAvailable > reports[j].TotalAvailable // Most shows first
		default: // "artist"
			return reports[i].Artist < reports[j].Artist
		}
	})
}

// Helper functions
func loadShowsData() (*models.ShowsData, error) {
	data, err := os.ReadFile("data/shows.json")
	if err != nil {
		return nil, err
	}

	var shows models.ShowsData
	err = json.Unmarshal(data, &shows)
	if err != nil {
		return nil, err
	}

	if shows.Artists == nil {
		shows.Artists = make(map[string]models.ArtistShowData)
	}

	return &shows, nil
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
