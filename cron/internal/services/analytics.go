package services

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/jmagar/nugs/cron/internal/models"
)

type AnalyticsService struct {
	DB         *sql.DB
	JobManager *models.JobManager
	startTime  time.Time
}

func NewAnalyticsService(db *sql.DB, jobManager *models.JobManager) *AnalyticsService {
	return &AnalyticsService{
		DB:         db,
		JobManager: jobManager,
		startTime:  time.Now(),
	}
}

func (s *AnalyticsService) GenerateReport(query *models.AnalyticsQuery) (*models.AnalyticsReport, error) {
	report := &models.AnalyticsReport{
		ReportID:    fmt.Sprintf("report_%d", time.Now().Unix()),
		ReportType:  query.ReportType,
		Timeframe:   query.Timeframe,
		GeneratedAt: time.Now(),
		Parameters:  make(map[string]interface{}),
	}

	// Store query parameters
	if query.StartDate != nil {
		report.Parameters["start_date"] = *query.StartDate
	}
	if query.EndDate != nil {
		report.Parameters["end_date"] = *query.EndDate
	}
	if len(query.ArtistIDs) > 0 {
		report.Parameters["artist_ids"] = query.ArtistIDs
	}

	switch query.ReportType {
	case "collection":
		stats, err := s.GetCollectionStats(query)
		if err != nil {
			return nil, err
		}
		report.CollectionStats = stats
		report.Summary = s.generateCollectionSummary(stats)

	case "artists":
		analytics, err := s.GetArtistAnalytics(query)
		if err != nil {
			return nil, err
		}
		report.ArtistAnalytics = analytics
		report.Summary = s.generateArtistSummary(analytics)

	case "downloads":
		analytics, err := s.GetDownloadAnalytics(query)
		if err != nil {
			return nil, err
		}
		report.DownloadAnalytics = analytics
		report.Summary = s.generateDownloadSummary(analytics)

	case "system":
		metrics, err := s.GetSystemMetrics()
		if err != nil {
			return nil, err
		}
		report.SystemMetrics = metrics
		report.Summary = s.generateSystemSummary(metrics)

	case "performance":
		performance, err := s.GetPerformanceMetrics()
		if err != nil {
			return nil, err
		}
		report.Performance = performance
		report.Summary = s.generatePerformanceSummary(performance)

	case "summary":
		// Generate a comprehensive summary report
		collectionStats, _ := s.GetCollectionStats(query)
		systemMetrics, _ := s.GetSystemMetrics()
		downloadAnalytics, _ := s.GetDownloadAnalytics(query)
		
		report.CollectionStats = collectionStats
		report.SystemMetrics = systemMetrics
		report.DownloadAnalytics = downloadAnalytics
		report.Summary = "Comprehensive summary report generated successfully"

	default:
		return nil, fmt.Errorf("unsupported report type: %s", query.ReportType)
	}

	if query.IncludeTimeSeries {
		timeSeries, err := s.generateTimeSeries(query)
		if err == nil {
			report.TimeSeries = timeSeries
		}
	}

	return report, nil
}

func (s *AnalyticsService) GetCollectionStats(query *models.AnalyticsQuery) (*models.CollectionStats, error) {
	stats := &models.CollectionStats{}

	// Basic counts
	err := s.DB.QueryRow(`
		SELECT 
			(SELECT COUNT(*) FROM artists) as total_artists,
			(SELECT COUNT(*) FROM shows) as total_shows,
			(SELECT COUNT(*) FROM downloads) as total_downloads,
			(SELECT COALESCE(SUM(size_mb), 0) / 1024.0 FROM downloads WHERE status = 'completed') as total_size_gb
	`).Scan(&stats.TotalArtists, &stats.TotalShows, &stats.TotalDownloads, &stats.TotalSizeGB)

	if err != nil {
		return nil, err
	}

	// Average shows per artist
	if stats.TotalArtists > 0 {
		stats.AverageShowsPerArtist = float64(stats.TotalShows) / float64(stats.TotalArtists)
	}

	// Recent activity
	s.DB.QueryRow(`
		SELECT COUNT(*) FROM shows WHERE date(created_at) = date('now')
	`).Scan(&stats.RecentActivity.NewShowsToday)

	s.DB.QueryRow(`
		SELECT COUNT(*) FROM shows 
		WHERE created_at >= datetime('now', '-7 days')
	`).Scan(&stats.RecentActivity.NewShowsThisWeek)

	s.DB.QueryRow(`
		SELECT COUNT(*) FROM shows 
		WHERE created_at >= datetime('now', 'start of month')
	`).Scan(&stats.RecentActivity.NewShowsThisMonth)

	s.DB.QueryRow(`
		SELECT COUNT(*) FROM downloads WHERE date(created_at) = date('now')
	`).Scan(&stats.RecentActivity.DownloadsToday)

	s.DB.QueryRow(`
		SELECT COUNT(*) FROM downloads 
		WHERE created_at >= datetime('now', '-7 days')
	`).Scan(&stats.RecentActivity.DownloadsThisWeek)

	s.DB.QueryRow(`
		SELECT COUNT(*) FROM downloads 
		WHERE created_at >= datetime('now', 'start of month')
	`).Scan(&stats.RecentActivity.DownloadsThisMonth)

	return stats, nil
}

func (s *AnalyticsService) GetArtistAnalytics(query *models.AnalyticsQuery) ([]models.ArtistAnalytics, error) {
	whereClause := "WHERE 1=1"
	args := []interface{}{}

	if len(query.ArtistIDs) > 0 {
		placeholders := make([]string, len(query.ArtistIDs))
		for i, id := range query.ArtistIDs {
			placeholders[i] = "?"
			args = append(args, id)
		}
		whereClause += " AND a.id IN (" + strings.Join(placeholders, ",") + ")"
	}

	querySQL := `
		SELECT 
			a.id, a.name,
			COUNT(DISTINCT s.id) as total_shows,
			COUNT(DISTINCT d.id) as total_downloads,
			COALESCE(SUM(CASE WHEN d.status = 'completed' THEN d.size_mb ELSE 0 END), 0) / 1024.0 as total_size_gb,
			MIN(s.date) as first_show_date,
			MAX(s.date) as last_show_date
		FROM artists a
		LEFT JOIN shows s ON a.id = s.artist_id
		LEFT JOIN downloads d ON s.id = d.show_id
		` + whereClause + `
		GROUP BY a.id, a.name
		ORDER BY total_downloads DESC
	`

	limit := query.Limit
	if limit > 0 && limit <= 1000 {
		querySQL += " LIMIT ?"
		args = append(args, limit)
	} else {
		querySQL += " LIMIT 100" // Default limit
	}

	rows, err := s.DB.Query(querySQL, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var analytics []models.ArtistAnalytics
	for rows.Next() {
		var artist models.ArtistAnalytics
		var firstShow, lastShow sql.NullString

		err := rows.Scan(
			&artist.ArtistID, &artist.ArtistName, &artist.TotalShows,
			&artist.TotalDownloads, &artist.TotalSizeGB, &firstShow, &lastShow,
		)
		if err != nil {
			continue
		}

		if firstShow.Valid {
			artist.FirstShowDate = &firstShow.String
		}
		if lastShow.Valid {
			artist.LastShowDate = &lastShow.String
		}

		// Calculate derived metrics
		if artist.TotalShows > 0 {
			artist.PopularityScore = float64(artist.TotalDownloads) / float64(artist.TotalShows)
			artist.AverageShowSizeGB = artist.TotalSizeGB / float64(artist.TotalShows)
		}

		// Get preferred format and quality
		var preferredFormat, preferredQuality sql.NullString
		s.DB.QueryRow(`
			SELECT format, quality
			FROM downloads d
			JOIN shows s ON d.show_id = s.id
			WHERE s.artist_id = ? AND d.status = 'completed'
			GROUP BY format, quality
			ORDER BY COUNT(*) DESC
			LIMIT 1
		`, artist.ArtistID).Scan(&preferredFormat, &preferredQuality)

		if preferredFormat.Valid {
			artist.PreferredFormat = preferredFormat.String
		}
		if preferredQuality.Valid {
			artist.PreferredQuality = preferredQuality.String
		}

		// Growth metrics
		s.DB.QueryRow(`
			SELECT COUNT(*) FROM shows s 
			WHERE s.artist_id = ? AND s.created_at >= datetime('now', '-30 days')
		`, artist.ArtistID).Scan(&artist.ShowGrowthLastMonth)

		s.DB.QueryRow(`
			SELECT COUNT(*) FROM downloads d
			JOIN shows s ON d.show_id = s.id
			WHERE s.artist_id = ? AND d.created_at >= datetime('now', '-30 days')
		`, artist.ArtistID).Scan(&artist.DownloadGrowthLastMonth)

		analytics = append(analytics, artist)
	}

	return analytics, nil
}

func (s *AnalyticsService) GetDownloadAnalytics(query *models.AnalyticsQuery) (*models.DownloadAnalytics, error) {
	analytics := &models.DownloadAnalytics{
		FormatBreakdown:  make(map[string]int64),
		QualityBreakdown: make(map[string]int64),
	}

	// Basic download stats
	err := s.DB.QueryRow(`
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed,
			COUNT(CASE WHEN status IN ('pending', 'in_progress') THEN 1 END) as pending,
			COALESCE(SUM(CASE WHEN status = 'completed' THEN size_mb ELSE 0 END), 0) / 1024.0 as total_size_gb
		FROM downloads
	`).Scan(&analytics.TotalDownloads, &analytics.CompletedDownloads,
		&analytics.FailedDownloads, &analytics.PendingDownloads, &analytics.TotalSizeGB)

	if err != nil {
		return nil, err
	}

	// Calculate success rate
	if analytics.TotalDownloads > 0 {
		analytics.SuccessRate = float64(analytics.CompletedDownloads) / float64(analytics.TotalDownloads) * 100
		analytics.AverageSizeGB = analytics.TotalSizeGB / float64(analytics.CompletedDownloads)
	}

	// Format breakdown
	rows, err := s.DB.Query(`
		SELECT format, COUNT(*), 
		       COALESCE(SUM(CASE WHEN status = 'completed' THEN size_mb ELSE 0 END), 0) / 1024.0 as size_gb
		FROM downloads 
		GROUP BY format
	`)
	if err == nil {
		defer rows.Close()
		var formatStats []models.FormatStats
		for rows.Next() {
			var format string
			var count int64
			var sizeGB float64
			if rows.Scan(&format, &count, &sizeGB) == nil {
				analytics.FormatBreakdown[format] = count

				percentage := float64(count) / float64(analytics.TotalDownloads) * 100
				avgSize := float64(0)
				if count > 0 {
					avgSize = sizeGB / float64(count)
				}

				formatStats = append(formatStats, models.FormatStats{
					Format:        format,
					Count:         count,
					Percentage:    percentage,
					TotalSizeGB:   sizeGB,
					AverageSizeGB: avgSize,
				})
			}
		}
		analytics.PopularFormats = formatStats
	}

	// Quality breakdown
	rows, err = s.DB.Query(`SELECT quality, COUNT(*) FROM downloads GROUP BY quality`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var quality string
			var count int64
			if rows.Scan(&quality, &count) == nil {
				analytics.QualityBreakdown[quality] = count
			}
		}
	}

	// Popular venues
	rows, err = s.DB.Query(`
		SELECT s.venue_name, s.venue_city, s.venue_state, 
		       COUNT(DISTINCT s.id) as show_count,
		       COUNT(d.id) as download_count
		FROM shows s
		JOIN downloads d ON s.id = d.show_id
		WHERE s.venue_name IS NOT NULL AND s.venue_name != ''
		GROUP BY s.venue_name, s.venue_city, s.venue_state
		ORDER BY download_count DESC
		LIMIT 10
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var venue models.VenueStats
			if rows.Scan(&venue.VenueName, &venue.VenueCity, &venue.VenueState,
				&venue.ShowCount, &venue.DownloadCount) == nil {
				analytics.PopularVenues = append(analytics.PopularVenues, venue)
			}
		}
	}

	// Download trends (last 30 days)
	rows, err = s.DB.Query(`
		SELECT date(created_at) as date, 
		       COUNT(*) as count,
		       COALESCE(SUM(CASE WHEN status = 'completed' THEN size_mb ELSE 0 END), 0) / 1024.0 as size_gb
		FROM downloads
		WHERE created_at >= datetime('now', '-30 days')
		GROUP BY date(created_at)
		ORDER BY date
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var trend models.TrendPoint
			if rows.Scan(&trend.Date, &trend.Count, &trend.SizeGB) == nil {
				analytics.DownloadTrends = append(analytics.DownloadTrends, trend)
			}
		}
	}

	// Peak download hours
	rows, err = s.DB.Query(`
		SELECT strftime('%H', created_at) as hour, COUNT(*) as count
		FROM downloads
		WHERE created_at >= datetime('now', '-7 days')
		GROUP BY strftime('%H', created_at)
		ORDER BY count DESC
		LIMIT 24
	`)
	if err == nil {
		defer rows.Close()
		hourMap := make(map[int]int64)
		for rows.Next() {
			var hourStr string
			var count int64
			if rows.Scan(&hourStr, &count) == nil {
				var hour int
				fmt.Sscanf(hourStr, "%d", &hour)
				hourMap[hour] = count
			}
		}

		// Convert to sorted slice
		for hour := 0; hour < 24; hour++ {
			analytics.PeakDownloadHours = append(analytics.PeakDownloadHours, models.HourStats{
				Hour:  hour,
				Count: hourMap[hour],
			})
		}
	}

	return analytics, nil
}

func (s *AnalyticsService) GetSystemMetrics() (*models.SystemMetrics, error) {
	metrics := &models.SystemMetrics{}

	// Database size
	var dbSizeMB float64
	if stat, err := os.Stat("./data/nugs_api.db"); err == nil {
		dbSizeMB = float64(stat.Size()) / (1024 * 1024)
	}
	metrics.DatabaseSize = dbSizeMB

	// File and storage info
	s.DB.QueryRow(`
		SELECT COUNT(*) FROM downloads WHERE file_path IS NOT NULL AND file_path != ''
	`).Scan(&metrics.TotalFiles)

	// System storage (simplified)
	stat := &syscall.Statfs_t{}
	if syscall.Statfs("/home/jmagar/code/nugs/downloads", stat) == nil {
		metrics.TotalStorage = float64(stat.Blocks*uint64(stat.Bsize)) / (1024 * 1024 * 1024)
		metrics.AvailableStorage = float64(stat.Bavail*uint64(stat.Bsize)) / (1024 * 1024 * 1024)
	}

	// Active monitors
	s.DB.QueryRow(`SELECT COUNT(*) FROM artist_monitors WHERE status = 'active'`).Scan(&metrics.ActiveMonitors)

	// System uptime
	metrics.SystemUptime = time.Since(s.startTime).String()

	// Last catalog refresh
	var lastRefreshStr sql.NullString
	s.DB.QueryRow(`
		SELECT value FROM system_config WHERE key = 'last_catalog_refresh'
	`).Scan(&lastRefreshStr)
	if lastRefreshStr.Valid {
		metrics.LastCatalogRefresh = &lastRefreshStr.String
	}

	// API request stats (simplified - would need request tracking)
	metrics.APIRequests.Today = 0
	metrics.APIRequests.ThisWeek = 0
	metrics.APIRequests.ThisMonth = 0

	// Job stats from job manager
	jobs := s.JobManager.ListJobs()
	metrics.JobStats.TotalJobs = int64(len(jobs))
	for _, job := range jobs {
		switch job.Status {
		case models.JobStatusCompleted:
			metrics.JobStats.CompletedJobs++
		case models.JobStatusFailed:
			metrics.JobStats.FailedJobs++
		case models.JobStatusRunning:
			metrics.JobStats.RunningJobs++
		}
	}

	return metrics, nil
}

func (s *AnalyticsService) GetPerformanceMetrics() (*models.PerformanceMetrics, error) {
	// Simplified performance metrics
	// In production, these would be collected from monitoring systems
	metrics := &models.PerformanceMetrics{
		AverageResponseTime:  50.0,  // ms
		DatabaseResponseTime: 10.0,  // ms
		CacheHitRate:         0.0,   // %
		ErrorRate:            2.0,   // %
		ThroughputPerSecond:  100.0, // requests/sec
		MemoryUsageMB:        128.0, // MB
		CPUUsagePercent:      15.0,  // %
	}

	return metrics, nil
}

func (s *AnalyticsService) generateTimeSeries(query *models.AnalyticsQuery) ([]models.TimeSeriesData, error) {
	var timeSeries []models.TimeSeriesData

	// Generate time series based on report type and timeframe
	switch query.ReportType {
	case "downloads":
		// Downloads over time
		downloads, err := s.generateDownloadTimeSeries(query.Timeframe)
		if err == nil {
			timeSeries = append(timeSeries, downloads)
		}

	case "collection":
		// Shows added over time
		shows, err := s.generateShowsTimeSeries(query.Timeframe)
		if err == nil {
			timeSeries = append(timeSeries, shows)
		}
	}

	return timeSeries, nil
}

func (s *AnalyticsService) generateDownloadTimeSeries(timeframe models.AnalyticsTimeframe) (models.TimeSeriesData, error) {
	var groupBy string
	switch timeframe {
	case models.TimeframeDay:
		groupBy = "strftime('%Y-%m-%d %H:00', created_at)"
	case models.TimeframeWeek, models.TimeframeMonth:
		groupBy = "date(created_at)"
	default:
		groupBy = "strftime('%Y-%m', created_at)"
	}

	query := fmt.Sprintf(`
		SELECT %s as period, COUNT(*) as count
		FROM downloads
		WHERE created_at >= datetime('now', '-%s')
		GROUP BY %s
		ORDER BY period
	`, groupBy, s.getTimeframeDuration(timeframe), groupBy)

	rows, err := s.DB.Query(query)
	if err != nil {
		return models.TimeSeriesData{}, err
	}
	defer rows.Close()

	var timestamps []string
	var values []float64

	for rows.Next() {
		var timestamp string
		var count float64
		if rows.Scan(&timestamp, &count) == nil {
			timestamps = append(timestamps, timestamp)
			values = append(values, count)
		}
	}

	return models.TimeSeriesData{
		Timestamps: timestamps,
		Values:     values,
		Label:      "Downloads",
		Unit:       "count",
	}, nil
}

func (s *AnalyticsService) generateShowsTimeSeries(timeframe models.AnalyticsTimeframe) (models.TimeSeriesData, error) {
	var groupBy string
	switch timeframe {
	case models.TimeframeDay:
		groupBy = "strftime('%Y-%m-%d %H:00', created_at)"
	case models.TimeframeWeek, models.TimeframeMonth:
		groupBy = "date(created_at)"
	default:
		groupBy = "strftime('%Y-%m', created_at)"
	}

	query := fmt.Sprintf(`
		SELECT %s as period, COUNT(*) as count
		FROM shows
		WHERE created_at >= datetime('now', '-%s')
		GROUP BY %s
		ORDER BY period
	`, groupBy, s.getTimeframeDuration(timeframe), groupBy)

	rows, err := s.DB.Query(query)
	if err != nil {
		return models.TimeSeriesData{}, err
	}
	defer rows.Close()

	var timestamps []string
	var values []float64

	for rows.Next() {
		var timestamp string
		var count float64
		if rows.Scan(&timestamp, &count) == nil {
			timestamps = append(timestamps, timestamp)
			values = append(values, count)
		}
	}

	return models.TimeSeriesData{
		Timestamps: timestamps,
		Values:     values,
		Label:      "Shows Added",
		Unit:       "count",
	}, nil
}

func (s *AnalyticsService) getTimeframeDuration(timeframe models.AnalyticsTimeframe) string {
	switch timeframe {
	case models.TimeframeDay:
		return "1 day"
	case models.TimeframeWeek:
		return "7 days"
	case models.TimeframeMonth:
		return "30 days"
	case models.TimeframeYear:
		return "365 days"
	default:
		return "30 days"
	}
}

// Summary generators
func (s *AnalyticsService) generateCollectionSummary(stats *models.CollectionStats) string {
	return fmt.Sprintf("Collection contains %d artists with %d total shows (%.1f shows per artist on average). Total download size: %.1f GB. Recent activity: %d new shows and %d downloads this month.",
		stats.TotalArtists, stats.TotalShows, stats.AverageShowsPerArtist,
		stats.TotalSizeGB, stats.RecentActivity.NewShowsThisMonth, stats.RecentActivity.DownloadsThisMonth)
}

func (s *AnalyticsService) generateArtistSummary(analytics []models.ArtistAnalytics) string {
	if len(analytics) == 0 {
		return "No artist data available."
	}

	topArtist := analytics[0]
	return fmt.Sprintf("Top artist: %s with %d shows and %d downloads (popularity score: %.1f). Analyzed %d artists total.",
		topArtist.ArtistName, topArtist.TotalShows, topArtist.TotalDownloads,
		topArtist.PopularityScore, len(analytics))
}

func (s *AnalyticsService) generateDownloadSummary(analytics *models.DownloadAnalytics) string {
	return fmt.Sprintf("Download statistics: %d total downloads with %.1f%% success rate. Total size: %.1f GB (avg %.2f GB per download). Peak activity in recent trends.",
		analytics.TotalDownloads, analytics.SuccessRate, analytics.TotalSizeGB, analytics.AverageSizeGB)
}

func (s *AnalyticsService) generateSystemSummary(metrics *models.SystemMetrics) string {
	return fmt.Sprintf("System status: Database %.1f MB, %d total files, %.1f GB storage used (%.1f GB available). %d active monitors, %d jobs completed.",
		metrics.DatabaseSize, metrics.TotalFiles, metrics.TotalStorage-metrics.AvailableStorage,
		metrics.AvailableStorage, metrics.ActiveMonitors, metrics.JobStats.CompletedJobs)
}

func (s *AnalyticsService) generatePerformanceSummary(perf *models.PerformanceMetrics) string {
	return fmt.Sprintf("Performance: %.1fms avg response time, %.1f%% error rate, %.1f req/sec throughput. CPU: %.1f%%, Memory: %.1f MB.",
		perf.AverageResponseTime, perf.ErrorRate, perf.ThroughputPerSecond,
		perf.CPUUsagePercent, perf.MemoryUsageMB)
}
