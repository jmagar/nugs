package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"sort"
	"strings"
	"time"
	
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
	TotalArtists       int     `json:"total_artists"`
	TotalShowsHave     int     `json:"total_shows_have"`
	TotalShowsAvail    int     `json:"total_shows_available"`
	OverallCompletion  float64 `json:"overall_completion"`
	TotalMissing       int     `json:"total_missing"`
}

func main() {
	// Command line flags
	var (
		format      = flag.String("format", "terminal", "Output format: terminal, html, csv, json")
		sortBy      = flag.String("sort", "artist", "Sort by: artist, completion, missing, total")
		artistName  = flag.String("artist", "", "Generate report for specific artist only")
		minMissing  = flag.Int("min-missing", 0, "Only show artists with at least N missing shows")
		outputFile  = flag.String("output", "", "Output file (default: stdout)")
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
			err := ioutil.WriteFile(*outputFile, []byte(html), 0644)
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
			err := ioutil.WriteFile(*outputFile, jsonData, 0644)
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
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>🎵 Nugs Collection Gap Analysis</title>
    <link rel="icon" href="data:image/svg+xml,<svg xmlns=%22http://www.w3.org/2000/svg%22 viewBox=%220 0 100 100%22><text y=%22.9em%22 font-size=%2290%22>🎵</text></svg>">
    <style>
        @import url('https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700;800&display=swap');
        
        * { box-sizing: border-box; }
        body { 
            font-family: 'Inter', system-ui, -apple-system, sans-serif;
            margin: 0;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            color: #2d3748;
            line-height: 1.6;
        }
        
        .container {
            max-width: 1400px;
            margin: 0 auto;
            padding: 40px 20px;
        }
        
        .header {
            text-align: center;
            margin-bottom: 48px;
            color: white;
        }
        
        .main-title {
            font-size: 4rem;
            font-weight: 800;
            margin-bottom: 16px;
            text-shadow: 0 4px 20px rgba(255, 255, 255, 0.3);
        }
        
        .subtitle {
            font-size: 1.25rem;
            opacity: 0.8;
            max-width: 600px;
            margin: 0 auto;
        }
        
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
            gap: 24px;
            margin-bottom: 48px;
        }
        
        .stat-card {
            background: rgba(255, 255, 255, 0.95);
            border-radius: 16px;
            padding: 32px 24px;
            text-align: center;
            box-shadow: 0 10px 30px rgba(0, 0, 0, 0.1);
            backdrop-filter: blur(10px);
            border: 1px solid rgba(255, 255, 255, 0.2);
            transition: all 0.3s ease;
        }
        
        .stat-card:hover {
            transform: translateY(-2px);
            box-shadow: 0 15px 35px rgba(0, 0, 0, 0.15);
        }
        
        .stat-icon {
            font-size: 2.5rem;
            margin-bottom: 16px;
        }
        
        .stat-value {
            font-size: 2.8rem;
            font-weight: 800;
            margin-bottom: 8px;
        }
        
        .stat-title {
            font-size: 16px;
            font-weight: 600;
            margin-bottom: 4px;
            color: #4a5568;
        }
        
        .stat-subtitle {
            font-size: 14px;
            color: #718096;
        }
        
        .controls {
            background: rgba(255, 255, 255, 0.15);
            border: 1px solid rgba(255, 255, 255, 0.2);
            border-radius: 16px;
            padding: 24px;
            margin-bottom: 32px;
            backdrop-filter: blur(10px);
        }
        
        .controls-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
        }
        
        .control-group label {
            color: white;
            font-weight: 600;
            font-size: 14px;
            margin-bottom: 8px;
            display: flex;
            align-items: center;
        }
        
        .control-group i {
            margin-right: 8px;
        }
        
        .control-input {
            background: rgba(255, 255, 255, 0.9);
            border: none;
            border-radius: 8px;
            padding: 12px 16px;
            width: 100%;
            font-size: 15px;
            font-family: inherit;
            transition: all 0.2s ease;
        }
        
        .control-input:focus {
            outline: none;
            background: white;
            box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.3);
        }
        
        .artists-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(400px, 1fr));
            gap: 24px;
            margin-bottom: 60px;
        }
        
        .artist-card {
            background: rgba(255, 255, 255, 0.98);
            border-radius: 20px;
            padding: 28px;
            box-shadow: 0 8px 25px rgba(0, 0, 0, 0.1);
            border: 1px solid rgba(255, 255, 255, 0.3);
            backdrop-filter: blur(10px);
            transition: all 0.3s ease;
            cursor: pointer;
            position: relative;
            overflow: hidden;
        }
        
        .artist-card:hover {
            transform: translateY(-5px);
            box-shadow: 0 20px 40px rgba(0, 0, 0, 0.15);
        }
        
        .artist-card::before {
            content: '';
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            height: 4px;
            background: linear-gradient(90deg, var(--completion-color, #ef4444), var(--completion-color, #ef4444));
        }
        
        .artist-header {
            display: flex;
            align-items: flex-start;
            gap: 20px;
            margin-bottom: 24px;
        }
        
        .album-art {
            width: 80px;
            height: 80px;
            border-radius: 12px;
            object-fit: cover;
            border: 2px solid rgba(255, 255, 255, 0.8);
            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
            transition: all 0.3s ease;
        }
        
        .artist-card:hover .album-art {
            transform: scale(1.05);
            box-shadow: 0 6px 16px rgba(0, 0, 0, 0.15);
        }
        
        .artist-info {
            flex: 1;
        }
        
        .artist-name {
            font-size: 1.3rem;
            font-weight: 700;
            color: #1a202c;
            margin-bottom: 8px;
            line-height: 1.2;
        }
        
        .artist-stats-line {
            font-size: 15px;
            color: #4a5568;
            margin-bottom: 16px;
        }
        
        .completion-bar {
            width: 100%;
            height: 8px;
            background: #e2e8f0;
            border-radius: 4px;
            overflow: hidden;
            margin-bottom: 12px;
        }
        
        .completion-fill {
            height: 100%;
            border-radius: 4px;
            transition: all 0.8s ease;
            background: var(--completion-color, #ef4444);
        }
        
        .completion-percentage {
            font-size: 18px;
            font-weight: 700;
            color: var(--completion-color, #ef4444);
        }
        
        .artist-numbers {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-top: 20px;
        }
        
        .stat-group {
            display: flex;
            gap: 24px;
        }
        
        .mini-stat {
            text-align: center;
        }
        
        .mini-stat-value {
            font-size: 1.4rem;
            font-weight: 700;
        }
        
        .mini-stat-label {
            font-size: 12px;
            color: #718096;
            font-weight: 500;
        }
        
        .view-button {
            background: linear-gradient(135deg, #667eea, #764ba2);
            color: white;
            border: none;
            border-radius: 10px;
            padding: 8px 16px;
            font-size: 13px;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.3s ease;
            display: flex;
            align-items: center;
            gap: 6px;
        }
        
        .view-button:hover {
            transform: translateY(-1px);
            box-shadow: 0 6px 16px rgba(102, 126, 234, 0.4);
        }
        
        .modal-overlay {
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: rgba(0, 0, 0, 0.75);
            display: flex;
            align-items: center;
            justify-content: center;
            z-index: 1000;
            padding: 20px;
            backdrop-filter: blur(8px);
        }
        
        .modal {
            background: rgba(255, 255, 255, 0.98);
            border-radius: 20px;
            padding: 32px;
            max-width: 900px;
            max-height: 80vh;
            overflow-y: auto;
            border: 1px solid rgba(255, 255, 255, 0.3);
            backdrop-filter: blur(15px);
            box-shadow: 0 25px 50px rgba(0, 0, 0, 0.25);
        }
        
        .modal-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 28px;
            padding-bottom: 20px;
            border-bottom: 2px solid #e2e8f0;
        }
        
        .modal-title {
            font-size: 1.8rem;
            font-weight: 800;
            color: #1a202c;
        }
        
        .close-button {
            background: #ef4444;
            color: white;
            border: none;
            border-radius: 8px;
            padding: 8px 12px;
            cursor: pointer;
            font-size: 16px;
            transition: all 0.2s ease;
        }
        
        .close-button:hover {
            background: #dc2626;
            transform: scale(1.05);
        }
        
        .modal-stats {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
            gap: 16px;
            margin-bottom: 32px;
        }
        
        .modal-stat-card {
            background: linear-gradient(135deg, #f8fafc, #e2e8f0);
            padding: 20px 16px;
            border-radius: 12px;
            text-align: center;
            border: 1px solid #e2e8f0;
        }
        
        .missing-shows-section h3 {
            color: #1a202c;
            margin-bottom: 20px;
            font-size: 1.2rem;
            font-weight: 700;
            display: flex;
            align-items: center;
            gap: 8px;
        }
        
        .missing-shows-list {
            max-height: 400px;
            overflow-y: auto;
            border: 1px solid #e2e8f0;
            border-radius: 12px;
            background: #f8fafc;
        }
        
        .missing-show-item {
            display: flex;
            align-items: center;
            gap: 16px;
            padding: 16px;
            border-bottom: 1px solid #e2e8f0;
            transition: all 0.2s ease;
        }
        
        .missing-show-item:hover {
            background: #e2e8f0;
        }
        
        .missing-show-item:last-child {
            border-bottom: none;
        }
        
        .missing-show-art {
            width: 50px;
            height: 50px;
            border-radius: 8px;
            object-fit: cover;
            border: 1px solid #cbd5e0;
        }
        
        .missing-show-info {
            flex: 1;
        }
        
        .missing-show-date {
            font-weight: 700;
            color: #2d3748;
            margin-bottom: 4px;
        }
        
        .missing-show-date a {
            color: #667eea;
            text-decoration: none;
            transition: color 0.2s ease;
        }
        
        .missing-show-date a:hover {
            color: #5a67d8;
            text-decoration: underline;
        }
        
        .missing-show-venue {
            font-size: 14px;
            color: #4a5568;
            margin-bottom: 2px;
        }
        
        .missing-show-id {
            font-size: 12px;
            color: #718096;
        }
        
        .empty-state {
            text-align: center;
            padding: 80px 20px;
            color: rgba(255, 255, 255, 0.8);
        }
        
        .empty-state i {
            font-size: 4rem;
            margin-bottom: 20px;
            opacity: 0.5;
        }
        
        .empty-state-text {
            font-size: 1.25rem;
            font-weight: 500;
        }
        
        .footer {
            text-align: center;
            padding: 40px 20px;
            color: rgba(255, 255, 255, 0.7);
            border-top: 1px solid rgba(255, 255, 255, 0.1);
            margin-top: 60px;
        }
        
        .footer-date {
            margin-bottom: 12px;
            font-size: 16px;
        }
        
        .footer-credit {
            font-size: 14px;
            opacity: 0.8;
        }
        
        .hidden { display: none !important; }
        
        /* Color variables for completion */
        .completion-high { --completion-color: #10b981; }
        .completion-medium { --completion-color: #f59e0b; }
        .completion-low { --completion-color: #ef4444; }
        
        /* Responsive design */
        @media (max-width: 768px) {
            .main-title { font-size: 2.5rem; }
            .artists-grid { grid-template-columns: 1fr; }
            .controls-grid { grid-template-columns: 1fr; }
            .stats-grid { grid-template-columns: repeat(2, 1fr); }
            .modal { margin: 10px; padding: 24px; }
        }
        
        @media (max-width: 480px) {
            .stats-grid { grid-template-columns: 1fr; }
            .artist-header { flex-direction: column; text-align: center; }
            .artist-numbers { flex-direction: column; gap: 16px; }
        }
    </style>
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css">
</head>
<body>
    <div class="container">
        <!-- Header -->
        <div class="header">
            <h1 class="main-title">🎵 Nugs Collection Analysis</h1>
            <p class="subtitle">Comprehensive gap analysis of your music collection with detailed insights</p>
        </div>
        
        <!-- Summary Stats -->
        <div class="stats-grid">`)

	// Summary stats
	html.WriteString(fmt.Sprintf(`
            <div class="stat-card">
                <div class="stat-icon" style="color: #667eea;"><i class="fas fa-users"></i></div>
                <div class="stat-value" style="color: #667eea;">%d</div>
                <div class="stat-title">Artists Monitored</div>
                <div class="stat-subtitle">active collections</div>
            </div>
            <div class="stat-card">
                <div class="stat-icon" style="color: #10b981;"><i class="fas fa-download"></i></div>
                <div class="stat-value" style="color: #10b981;">%s</div>
                <div class="stat-title">Shows Downloaded</div>
                <div class="stat-subtitle">in collection</div>
            </div>
            <div class="stat-card">
                <div class="stat-icon" style="color: #f59e0b;"><i class="fas fa-music"></i></div>
                <div class="stat-value" style="color: #f59e0b;">%s</div>
                <div class="stat-title">Shows Available</div>
                <div class="stat-subtitle">on platform</div>
            </div>`,
		summary.TotalArtists,
		formatNumber(summary.TotalShowsHave),
		formatNumber(summary.TotalShowsAvail)))
	
	completionColor := "#ef4444" // red
	if summary.OverallCompletion >= 90 {
		completionColor = "#10b981" // green
	} else if summary.OverallCompletion >= 70 {
		completionColor = "#f59e0b" // yellow
	}
	
	html.WriteString(fmt.Sprintf(`
            <div class="stat-card">
                <div class="stat-icon" style="color: %s;"><i class="fas fa-chart-pie"></i></div>
                <div class="stat-value" style="color: %s;">%.1f%%</div>
                <div class="stat-title">Overall Completion</div>
                <div class="stat-subtitle">collection progress</div>
            </div>
            <div class="stat-card">
                <div class="stat-icon" style="color: #ef4444;"><i class="fas fa-exclamation-triangle"></i></div>
                <div class="stat-value" style="color: #ef4444;">%s</div>
                <div class="stat-title">Missing Shows</div>
                <div class="stat-subtitle">to download</div>
            </div>
        </div>`,
		completionColor, completionColor, summary.OverallCompletion,
		formatNumber(summary.TotalMissing)))

	// Controls
	html.WriteString(`
        <!-- Controls -->
        <div class="controls">
            <div class="controls-grid">
                <div class="control-group">
                    <label for="searchInput"><i class="fas fa-search"></i>Search Artists</label>
                    <input type="text" id="searchInput" class="control-input" placeholder="Search by artist name..." oninput="filterArtists()">
                </div>
                <div class="control-group">
                    <label for="sortSelect"><i class="fas fa-sort"></i>Sort By</label>
                    <select id="sortSelect" class="control-input" onchange="sortArtists()">
                        <option value="completion">Completion %</option>
                        <option value="artist">Artist Name</option>
                        <option value="missing">Missing Shows</option>
                        <option value="total">Total Shows</option>
                    </select>
                </div>
                <div class="control-group">
                    <label for="minMissingInput"><i class="fas fa-filter"></i>Min Missing Shows</label>
                    <input type="number" id="minMissingInput" class="control-input" placeholder="0" min="0" oninput="filterArtists()">
                </div>
            </div>
        </div>
        
        <!-- Artists Grid -->
        <div class="artists-grid" id="artistsGrid">`)

	// Artist cards
	for _, report := range reports {
		completionClass := "completion-low"
		if report.CompletionPct >= 90 {
			completionClass = "completion-high"
		} else if report.CompletionPct >= 70 {
			completionClass = "completion-medium"
		}
		
		latestShowID := report.ArtistID
		if len(report.MissingShows) > 0 {
			latestShowID = report.MissingShows[0].ContainerID
		}
		
		html.WriteString(fmt.Sprintf(`
            <div class="artist-card %s" data-artist="%s" data-completion="%.1f" data-missing="%d" data-total="%d" onclick="openModal('%s')">
                <div class="artist-header">
                    <img class="album-art" src="https://nugs.net/images/releases/%d/%d_300.jpg" 
                         alt="%s album art" 
                         onerror="this.src='data:image/svg+xml,<svg xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 100 100\"><rect width=\"100\" height=\"100\" fill=\"%%236366f1\"/><text x=\"50\" y=\"55\" text-anchor=\"middle\" fill=\"white\" font-size=\"40\">🎵</text></svg>'">
                    <div class="artist-info">
                        <h2 class="artist-name">%s</h2>
                        <div class="artist-stats-line">%d/%d shows • %.1f%% complete</div>
                        <div class="completion-bar">
                            <div class="completion-fill" style="width: %.1f%%;"></div>
                        </div>
                        <div class="completion-percentage">%.1f%% Complete</div>
                    </div>
                </div>
                
                <div class="artist-numbers">
                    <div class="stat-group">
                        <div class="mini-stat">
                            <div class="mini-stat-value" style="color: #10b981;">%d</div>
                            <div class="mini-stat-label">Downloaded</div>
                        </div>
                        <div class="mini-stat">
                            <div class="mini-stat-value" style="color: #ef4444;">%d</div>
                            <div class="mini-stat-label">Missing</div>
                        </div>
                    </div>
                    <button class="view-button">
                        <i class="fas fa-eye"></i>
                        View Details
                    </button>
                </div>
            </div>`,
			completionClass,
			report.Artist,
			report.CompletionPct,
			report.MissingCount,
			report.TotalAvailable,
			strings.ReplaceAll(report.Artist, "'", "\\'"),
			latestShowID, latestShowID,
			report.Artist,
			report.Artist,
			report.TotalDownloaded, report.TotalAvailable, report.CompletionPct,
			report.CompletionPct,
			report.CompletionPct,
			report.TotalDownloaded,
			report.MissingCount))
	}

	html.WriteString(`
        </div>
        
        <!-- Empty State -->
        <div class="empty-state hidden" id="emptyState">
            <i class="fas fa-search"></i>
            <div class="empty-state-text">No artists match your current filters</div>
        </div>
        
        <!-- Footer -->
        <div class="footer">
            <div class="footer-date">Generated on ` + 
            fmt.Sprintf("%s", time.Now().Format("January 2, 2006 at 3:04 PM")) + `</div>
            <div class="footer-credit">🎵 Nugs Collection Gap Report</div>
        </div>
    </div>

    <!-- Modal -->
    <div class="modal-overlay hidden" id="modalOverlay" onclick="closeModal()">
        <div class="modal" onclick="event.stopPropagation()">
            <div class="modal-header">
                <h2 class="modal-title" id="modalTitle">Artist Details</h2>
                <button class="close-button" onclick="closeModal()">
                    <i class="fas fa-times"></i>
                </button>
            </div>
            <div class="modal-stats" id="modalStats">
                <!-- Stats will be populated by JavaScript -->
            </div>
            <div class="missing-shows-section" id="missingShowsSection">
                <!-- Missing shows will be populated by JavaScript -->
            </div>
        </div>
    </div>

    <script>`)

	// Embed artist data as JavaScript
	html.WriteString(`
        const artistsData = {`)
	
	for i, report := range reports {
		if i > 0 {
			html.WriteString(`,`)
		}
		missingShowsJSON, _ := json.Marshal(report.MissingShows)
		html.WriteString(fmt.Sprintf(`
            "%s": {
                "artist": "%s",
                "total_available": %d,
                "total_downloaded": %d,
                "completion_pct": %.1f,
                "missing_shows": %s
            }`,
			strings.ReplaceAll(report.Artist, `"`, `\"`),
			strings.ReplaceAll(report.Artist, `"`, `\"`),
			report.TotalAvailable,
			report.TotalDownloaded,
			report.CompletionPct,
			string(missingShowsJSON)))
	}
	
	html.WriteString(`
        };
        
        function filterArtists() {
            const searchTerm = document.getElementById('searchInput').value.toLowerCase();
            const minMissing = parseInt(document.getElementById('minMissingInput').value) || 0;
            const cards = document.querySelectorAll('.artist-card');
            let visibleCount = 0;
            
            cards.forEach(card => {
                const artistName = card.dataset.artist.toLowerCase();
                const missing = parseInt(card.dataset.missing);
                
                if (artistName.includes(searchTerm) && missing >= minMissing) {
                    card.classList.remove('hidden');
                    visibleCount++;
                } else {
                    card.classList.add('hidden');
                }
            });
            
            document.getElementById('emptyState').classList.toggle('hidden', visibleCount > 0);
        }
        
        function sortArtists() {
            const sortBy = document.getElementById('sortSelect').value;
            const container = document.getElementById('artistsGrid');
            const cards = Array.from(container.querySelectorAll('.artist-card'));
            
            cards.sort((a, b) => {
                switch(sortBy) {
                    case 'artist':
                        return a.dataset.artist.localeCompare(b.dataset.artist);
                    case 'completion':
                        return parseFloat(b.dataset.completion) - parseFloat(a.dataset.completion);
                    case 'missing':
                        return parseInt(b.dataset.missing) - parseInt(a.dataset.missing);
                    case 'total':
                        return parseInt(b.dataset.total) - parseInt(a.dataset.total);
                    default:
                        return 0;
                }
            });
            
            cards.forEach(card => container.appendChild(card));
        }
        
        function openModal(artistName) {
            const data = artistsData[artistName];
            if (!data) return;
            
            document.getElementById('modalTitle').textContent = artistName;
            
            // Update modal stats
            const statsHtml = ` + "`" + `
                <div class="modal-stat-card">
                    <div class="stat-icon" style="color: #10b981;"><i class="fas fa-check-circle"></i></div>
                    <div class="stat-value" style="color: #10b981;">${data.total_downloaded}</div>
                    <div class="stat-title">Downloaded</div>
                </div>
                <div class="modal-stat-card">
                    <div class="stat-icon" style="color: #ef4444;"><i class="fas fa-exclamation-circle"></i></div>
                    <div class="stat-value" style="color: #ef4444;">${data.missing_shows.length}</div>
                    <div class="stat-title">Missing</div>
                </div>
                <div class="modal-stat-card">
                    <div class="stat-icon" style="color: #667eea;"><i class="fas fa-music"></i></div>
                    <div class="stat-value" style="color: #667eea;">${data.total_available}</div>
                    <div class="stat-title">Total Available</div>
                </div>
                <div class="modal-stat-card">
                    <div class="stat-icon" style="color: ${data.completion_pct >= 90 ? '#10b981' : data.completion_pct >= 70 ? '#f59e0b' : '#ef4444'};"><i class="fas fa-chart-pie"></i></div>
                    <div class="stat-value" style="color: ${data.completion_pct >= 90 ? '#10b981' : data.completion_pct >= 70 ? '#f59e0b' : '#ef4444'};">${data.completion_pct.toFixed(1)}%</div>
                    <div class="stat-title">Completion</div>
                </div>
            ` + "`" + `;
            document.getElementById('modalStats').innerHTML = statsHtml;
            
            // Update missing shows
            if (data.missing_shows.length > 0) {
                let showsHtml = ` + "`" + `<h3><i class="fas fa-list-ul"></i>Missing Shows (${data.missing_shows.length})</h3><div class="missing-shows-list">` + "`" + `;
                
                data.missing_shows.forEach(show => {
                    showsHtml += ` + "`" + `
                        <div class="missing-show-item">
                            <img class="missing-show-art" src="https://nugs.net/images/releases/${show.container_id}/${show.container_id}_300.jpg" 
                                 alt="Show artwork"
                                 onerror="this.src='data:image/svg+xml,<svg xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 100 100\"><rect width=\"100\" height=\"100\" fill=\"%236366f1\"/><text x=\"50\" y=\"55\" text-anchor=\"middle\" fill=\"white\" font-size=\"20\">🎵</text></svg>'">
                            <div class="missing-show-info">
                                <div class="missing-show-date">
                                    <a href="https://nugs.net/${show.container_id}" target="_blank">${show.date}</a>
                                </div>
                                <div class="missing-show-venue">${show.venue}, ${show.city} ${show.state}</div>
                                <div class="missing-show-id">ID: ${show.container_id}</div>
                            </div>
                        </div>
                    ` + "`" + `;
                });
                
                showsHtml += ` + "`" + `</div>` + "`" + `;
                document.getElementById('missingShowsSection').innerHTML = showsHtml;
            } else {
                document.getElementById('missingShowsSection').innerHTML = ` + "`" + `<h3><i class="fas fa-check-circle" style="color: #10b981;"></i>All Shows Downloaded!</h3><p style="color: #10b981; font-weight: 600;">This collection is complete! 🎉</p>` + "`" + `;
            }
            
            document.getElementById('modalOverlay').classList.remove('hidden');
            document.body.style.overflow = 'hidden';
        }
        
        function closeModal() {
            document.getElementById('modalOverlay').classList.add('hidden');
            document.body.style.overflow = 'auto';
        }
        
        // Initialize default sorting
        sortArtists();
    </script>
</body>
</html>`)

	return html.String()
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
		err := ioutil.WriteFile(outputFile, []byte(output.String()), 0644)
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
	data, err := ioutil.ReadFile("data/shows.json")
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
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	
	var config models.MonitorConfig
	err = json.Unmarshal(data, &config)
	return &config, err
}