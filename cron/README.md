# Nugs.net Monitoring System

Comprehensive automated system for monitoring, downloading, and tracking shows from Nugs.net. Uses efficient catalog caching to minimize API calls and includes comprehensive safety features.

## Architecture Overview

The system consists of five main components:
- **Catalog Manager** - Caches the entire Nugs.net catalog locally (30,101 shows)
- **Artist Monitor** - Downloads new shows for specified artists
- **Missing Shows Detector** - Identifies gaps in downloaded collections
- **Gap Report Generator** - Creates comprehensive HTML/terminal reports of collection gaps
- **API Safety Layer** - Rate limiting, circuit breaker, and logging

## Core Files

### Main Programs
- **`cmd/catalog/main.go`** - Manages cached catalog of all 30,101 shows
- **`cmd/monitor/main.go`** - Monitors artists and downloads new shows (no date restrictions)
- **`cmd/detector/main.go`** - Detects missing shows by matching folder names to catalog
- **`cmd/gap_report/main.go`** - Generates comprehensive gap analysis reports
- **`cmd/apimon/main.go`** - API monitoring and emergency controls
- **`internal/api/client.go`** - Safe API client with rate limiting and error handling

### Configuration Files
- **`monitor_config.json`** - Artists to monitor with folders and settings
- **`config.json`** - Nugs.net credentials and download settings  
- **`api_config.json`** - API safety limits (auto-generated with defaults)

### Data Files
- **`catalog_cache.json`** - Complete cached catalog (171MB, refreshed daily)
- **`shows.json`** - Enhanced tracking with metadata and per-artist status
- **`api_stats.json`** - API usage statistics and rate limiting data

### Executables
- **`bin/nugs-dl`** - Nugs.net downloader binary
- **`bin/catalog_manager`** - Catalog management CLI
- **`bin/monitor_artists`** - Artist monitoring and downloading
- **`bin/missing_shows_detector`** - Gap analysis (updates shows.json)
- **`bin/gap_report`** - Gap report generator (HTML/terminal output)
- **`bin/api_monitor`** - API monitoring and controls

### Scripts
- **`monitor_artists.sh`** - Shell wrapper for cron execution with logging
- **`setup_cron.sh`** - Script to configure daily cron job

## Setup

1. **Build executables:**
   ```bash
   make all
   # OR build individually:
   make catalog     # builds bin/catalog_manager
   make monitor     # builds bin/monitor_artists  
   make detector    # builds bin/missing_shows_detector
   make gap_report  # builds bin/gap_report
   make apimon      # builds bin/api_monitor
   ```

2. **Configure artists to monitor:**
   ```bash
   nano configs/monitor_config.json
   ```

3. **Set up daily monitoring:**
   ```bash
   ./scripts/setup_cron.sh
   ```

4. **Test the system:**
   ```bash
   ./bin/missing_shows_detector  # Analysis only
   ./bin/monitor_artists          # Download new shows
   ```

## Configuration

### Enhanced monitor_config.json
```json
{
  "artists": [
    {
      "id": 1125,
      "artist": "Billy Strings",
      "monitor": true,
      "artist_folder": "/mnt/user/data/media/music/Billy Strings"
    },
    {
      "id": 62,
      "artist": "Phish", 
      "monitor": false,
      "artist_folder": "/mnt/user/data/media/music/Phish"
    }
  ]
}
```

### Enhanced shows.json Structure (in data/)
```json
{
  "last_catalog_update": "2025-08-22T22:56:36-04:00",
  "catalog_total_shows": 30101,
  "catalog_total_artists": 532,
  "last_analysis_time": "2025-08-22T23:31:46-04:00",
  "artists": {
    "Billy Strings": {
      "artist_id": 1125,
      "downloaded": [19574, 19990],
      "available": [19574, 19990, 20100],
      "missing": [20100]
    }
  }
}
```

## How It Works (New Architecture)

1. **Catalog Refresh** - Daily fetch of entire catalog (1 API call, no auth needed)
2. **Local Queries** - All artist/show lookups use cached catalog (no API calls)
3. **Folder Matching** - Advanced parser matches downloaded folder names to catalog show IDs
4. **Smart Downloads** - Downloads all missing shows (no date restrictions)
5. **Gap Analysis** - Comprehensive HTML/terminal reports with interactive features
6. **API Safety** - Rate limits: 30/min, 500/hr, 5000/day with circuit breaker
7. **Remote Sync** - Rsync to tootie server with cleanup of local files

## Key Improvements

- **99.9% fewer API calls** - From 600+ calls to 1 daily catalog refresh
- **No authentication needed** - Catalog endpoint is public
- **Accurate gap detection** - Folder name parsing matches 99%+ of downloaded shows
- **No date restrictions** - Downloads all missing shows, not just recent ones
- **Interactive HTML reports** - Visual gap analysis with search/filtering
- **Comprehensive monitoring** - Enable monitoring for any artists in your collection
- **API safety** - Rate limiting, circuit breaker, emergency stop
- **Efficient caching** - 171MB JSON catalog cached locally

## CLI Tools

### Catalog Manager
```bash
./bin/catalog_manager stats                    # Show catalog statistics
./bin/catalog_manager refresh                  # Force catalog refresh
./bin/catalog_manager artist "Billy Strings"   # Show all shows for artist
```

### API Monitor
```bash
./bin/api_monitor status            # Show API client status
./bin/api_monitor stats             # Show detailed statistics
./bin/api_monitor logs             # Show recent API logs
./bin/api_monitor stop             # Emergency stop (creates STOP_API file)
./bin/api_monitor start            # Remove emergency stop
```

### Missing Shows Analysis
```bash
./bin/missing_shows_detector        # Full analysis of all monitored artists (updates shows.json)
```

### Gap Report Generator
```bash
# Terminal output (default)
./bin/gap_report                              # All monitored artists
./bin/gap_report --sort completion            # Sort by completion percentage
./bin/gap_report --min-missing 10             # Only show artists with 10+ missing shows
./bin/gap_report --artist "Billy Strings"     # Single artist report

# HTML output (interactive)
./bin/gap_report --format html                # Creates gap_report.html
./bin/gap_report --format html --output collection_gaps.html

# Other formats (terminal, html, csv, json)
./bin/gap_report --format json --output gaps.json
```

**Gap Report Features:**
- **Interactive HTML** with search, filtering, and sorting
- **Detailed missing shows** with ID, date, venue information
- **Completion percentages** and visual progress bars
- **Summary statistics** across entire collection
- **Top complete collections** highlighting
- **Command-line filtering** by artist or missing count

## Logs

### Application Logs
```bash
tail -f monitor.log                 # Monitoring activity
ls -la logs/api_logs/              # Daily API request logs
```

### API Logs Structure
```bash
logs/api_logs/
├── api_requests_2025-08-22.log  # Daily request logs (JSON format)
├── api_requests_2025-08-23.log
└── ...
```

## Finding Artist IDs

### Method 1: Use cached catalog
```bash
./bin/catalog_manager stats | grep -i "artist name"
```

### Method 2: Direct API call
```bash
curl -s "https://streamapi.nugs.net/api.aspx?method=catalog.artists" | jq '.Response[] | select(.artistName | contains("Artist Name")) | {artistID, artistName}'
```

## API Safety Features

- **Rate Limiting**: 30 requests/minute, 500/hour, 5000/day
- **Circuit Breaker**: Stops after 5 consecutive errors
- **Emergency Stop**: Create `STOP_API` file to halt all requests
- **Request Logging**: All API calls logged with timing and response codes
- **Retry Logic**: Exponential backoff with configurable attempts

## Dependencies

- **Go 1.19+** - For building executables
- **SSH access** - To tootie server on port 29229
- **rsync** - For file transfers
- **jq** - For JSON processing (optional)
- All Go dependencies are vendored (standalone)