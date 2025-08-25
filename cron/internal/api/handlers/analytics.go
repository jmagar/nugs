package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmagar/nugs/cron/internal/models"
	"github.com/jmagar/nugs/cron/internal/services"
)

type AnalyticsHandler struct {
	AnalyticsService *services.AnalyticsService
	DB               *sql.DB
}

func NewAnalyticsHandler(db *sql.DB, jobManager *models.JobManager) *AnalyticsHandler {
	analyticsService := services.NewAnalyticsService(db, jobManager)

	return &AnalyticsHandler{
		AnalyticsService: analyticsService,
		DB:               db,
	}
}

// POST /api/v1/analytics/reports
func (h *AnalyticsHandler) GenerateReport(c *gin.Context) {
	var query models.AnalyticsQuery
	if err := c.ShouldBindJSON(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format: " + err.Error(),
		})
		return
	}

	// Set defaults
	if query.Timeframe == "" {
		query.Timeframe = models.TimeframeMonth
	}

	report, err := h.AnalyticsService.GenerateReport(&query)
	if err != nil {
		// Check if it's a validation error
		if strings.Contains(err.Error(), "unsupported report type") {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate report: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, report)
}

// GET /api/v1/analytics/collection
func (h *AnalyticsHandler) GetCollectionStats(c *gin.Context) {
	timeframe := models.AnalyticsTimeframe(c.DefaultQuery("timeframe", "month"))

	query := &models.AnalyticsQuery{
		ReportType: "collection",
		Timeframe:  timeframe,
	}

	stats, err := h.AnalyticsService.GetCollectionStats(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get collection statistics",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GET /api/v1/analytics/artists
func (h *AnalyticsHandler) GetArtistAnalytics(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	timeframe := models.AnalyticsTimeframe(c.DefaultQuery("timeframe", "month"))

	query := &models.AnalyticsQuery{
		ReportType: "artists",
		Timeframe:  timeframe,
		Limit:      limit,
	}

	// Note: artist_ids filter not implemented yet - could parse comma-separated IDs

	analytics, err := h.AnalyticsService.GetArtistAnalytics(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get artist analytics",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      analytics,
		"total":     len(analytics),
		"timeframe": timeframe,
	})
}

// GET /api/v1/analytics/downloads
func (h *AnalyticsHandler) GetDownloadAnalytics(c *gin.Context) {
	timeframe := models.AnalyticsTimeframe(c.DefaultQuery("timeframe", "month"))

	query := &models.AnalyticsQuery{
		ReportType:        "downloads",
		Timeframe:         timeframe,
		IncludeTimeSeries: c.Query("include_time_series") == "true",
	}

	analytics, err := h.AnalyticsService.GetDownloadAnalytics(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get download analytics",
		})
		return
	}

	response := gin.H{
		"data":      analytics,
		"timeframe": timeframe,
	}

	if query.IncludeTimeSeries {
		timeSeries, err := h.AnalyticsService.GenerateReport(query)
		if err == nil && timeSeries.TimeSeries != nil {
			response["time_series"] = timeSeries.TimeSeries
		}
	}

	c.JSON(http.StatusOK, response)
}

// GET /api/v1/analytics/system
func (h *AnalyticsHandler) GetSystemMetrics(c *gin.Context) {
	metrics, err := h.AnalyticsService.GetSystemMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get system metrics",
		})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// GET /api/v1/analytics/performance
func (h *AnalyticsHandler) GetPerformanceMetrics(c *gin.Context) {
	metrics, err := h.AnalyticsService.GetPerformanceMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get performance metrics",
		})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// GET /api/v1/analytics/top/artists
func (h *AnalyticsHandler) GetTopArtists(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	sortBy := c.DefaultQuery("sort_by", "downloads") // downloads, shows, size

	var orderClause string
	switch sortBy {
	case "shows":
		orderClause = "total_shows DESC"
	case "size":
		orderClause = "total_size_gb DESC"
	default:
		orderClause = "total_downloads DESC"
	}

	query := `
		SELECT 
			a.id, a.name,
			COUNT(DISTINCT s.id) as total_shows,
			COUNT(DISTINCT d.id) as total_downloads,
			COALESCE(SUM(CASE WHEN d.status = 'completed' THEN d.file_size ELSE 0 END), 0) / 1073741824.0 as total_size_gb
		FROM artists a
		LEFT JOIN shows s ON a.id = s.artist_id
		LEFT JOIN downloads d ON s.id = d.show_id
		GROUP BY a.id, a.name
		HAVING COUNT(DISTINCT d.id) > 0
		ORDER BY ` + orderClause + `
		LIMIT ?
	`

	rows, err := h.DB.Query(query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get top artists",
		})
		return
	}
	defer rows.Close()

	var topItems []models.TopListItem
	for rows.Next() {
		var id int
		var name string
		var shows, downloads int64
		var sizeGB float64

		if rows.Scan(&id, &name, &shows, &downloads, &sizeGB) == nil {
			var value, secondaryValue float64
			var unit string

			switch sortBy {
			case "shows":
				value = float64(shows)
				secondaryValue = float64(downloads)
				unit = "shows"
			case "size":
				value = sizeGB
				secondaryValue = float64(downloads)
				unit = "GB"
			default:
				value = float64(downloads)
				secondaryValue = sizeGB
				unit = "downloads"
			}

			topItems = append(topItems, models.TopListItem{
				ID:             id,
				Name:           name,
				Value:          value,
				SecondaryValue: secondaryValue,
				Unit:           unit,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    topItems,
		"sort_by": sortBy,
		"limit":   limit,
		"total":   len(topItems),
	})
}

// GET /api/v1/analytics/top/venues
func (h *AnalyticsHandler) GetTopVenues(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	query := `
		SELECT s.venue, s.city, s.state,
		       COUNT(DISTINCT s.id) as show_count,
		       COUNT(d.id) as download_count
		FROM shows s
		JOIN downloads d ON s.id = d.show_id
		WHERE s.venue IS NOT NULL AND s.venue != ''
		GROUP BY s.venue, s.city, s.state
		ORDER BY download_count DESC
		LIMIT ?
	`

	rows, err := h.DB.Query(query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get top venues",
		})
		return
	}
	defer rows.Close()

	var topVenues []models.VenueStats
	for rows.Next() {
		var venue models.VenueStats
		if rows.Scan(&venue.VenueName, &venue.VenueCity, &venue.VenueState,
			&venue.ShowCount, &venue.DownloadCount) == nil {
			topVenues = append(topVenues, venue)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  topVenues,
		"limit": limit,
		"total": len(topVenues),
	})
}

// GET /api/v1/analytics/trends/downloads
func (h *AnalyticsHandler) GetDownloadTrends(c *gin.Context) {
	timeframe := models.AnalyticsTimeframe(c.DefaultQuery("timeframe", "month"))
	groupBy := c.DefaultQuery("group_by", "day") // day, week, month

	var dateFormat, duration string
	switch groupBy {
	case "week":
		dateFormat = "strftime('%Y-W%W', created_at)"
		duration = "84 days" // 12 weeks
	case "month":
		dateFormat = "strftime('%Y-%m', created_at)"
		duration = "365 days" // 12 months
	default:
		dateFormat = "date(created_at)"
		duration = "30 days"
	}

	query := `
		SELECT ` + dateFormat + ` as period,
		       COUNT(*) as downloads,
		       COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed,
		       COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed,
		       COALESCE(SUM(CASE WHEN status = 'completed' THEN file_size ELSE 0 END), 0) / 1073741824.0 as size_gb
		FROM downloads
		WHERE created_at >= datetime('now', '-` + duration + `')
		GROUP BY ` + dateFormat + `
		ORDER BY period DESC
		LIMIT 50
	`

	rows, err := h.DB.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get download trends",
		})
		return
	}
	defer rows.Close()

	var trends []gin.H
	for rows.Next() {
		var period string
		var downloads, completed, failed int64
		var sizeGB float64

		if rows.Scan(&period, &downloads, &completed, &failed, &sizeGB) == nil {
			successRate := float64(0)
			if downloads > 0 {
				successRate = float64(completed) / float64(downloads) * 100
			}

			trends = append(trends, gin.H{
				"period":       period,
				"downloads":    downloads,
				"completed":    completed,
				"failed":       failed,
				"success_rate": successRate,
				"size_gb":      sizeGB,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      trends,
		"timeframe": timeframe,
		"group_by":  groupBy,
		"total":     len(trends),
	})
}

// GET /api/v1/analytics/summary
func (h *AnalyticsHandler) GetDashboardSummary(c *gin.Context) {
	// Get key metrics for dashboard display
	var summary gin.H = gin.H{}

	// Collection overview
	var totalArtists, totalShows, totalDownloads int64
	var totalSizeGB float64
	row := h.DB.QueryRow(`
		SELECT 
			(SELECT COUNT(*) FROM artists) as total_artists,
			(SELECT COUNT(*) FROM shows) as total_shows,
			(SELECT COUNT(*) FROM downloads) as total_downloads,
			(SELECT COALESCE(SUM(file_size), 0) / 1073741824.0 FROM downloads WHERE status = 'completed') as total_size_gb
	`)
	if err := row.Scan(&totalArtists, &totalShows, &totalDownloads, &totalSizeGB); err != nil {
		log.Printf("Error scanning dashboard stats: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get dashboard stats"})
		return
	}

	summary["collection"] = gin.H{
		"total_artists":   totalArtists,
		"total_shows":     totalShows,
		"total_downloads": totalDownloads,
		"total_size_gb":   totalSizeGB,
	}

	// Recent activity (last 24 hours)
	var recentShows, recentDownloads int64
	if err := h.DB.QueryRow(`SELECT COUNT(*) FROM shows WHERE created_at >= datetime('now', '-1 day')`).Scan(&recentShows); err != nil {
		log.Printf("Error scanning recent shows: %v", err)
		recentShows = 0
	}
	if err := h.DB.QueryRow(`SELECT COUNT(*) FROM downloads WHERE created_at >= datetime('now', '-1 day')`).Scan(&recentDownloads); err != nil {
		log.Printf("Error scanning recent downloads: %v", err)
		recentDownloads = 0
	}

	summary["recent_activity"] = gin.H{
		"new_shows_24h":     recentShows,
		"new_downloads_24h": recentDownloads,
	}

	// System status
	var activeMonitors, runningJobs, failedJobs int64
	if err := h.DB.QueryRow(`SELECT COUNT(*) FROM artist_monitors WHERE status = 'active'`).Scan(&activeMonitors); err != nil {
		log.Printf("Warning: failed to get active monitors: %v", err)
		activeMonitors = 0
	}

	jobs := h.AnalyticsService.JobManager.ListJobs()
	for _, job := range jobs {
		switch job.Status {
		case models.JobStatusRunning:
			runningJobs++
		case models.JobStatusFailed:
			failedJobs++
		}
	}

	summary["system_status"] = gin.H{
		"active_monitors": activeMonitors,
		"running_jobs":    runningJobs,
		"failed_jobs":     failedJobs,
	}

	// Top format breakdown
	formatRows, err := h.DB.Query(`
		SELECT format, COUNT(*) as count
		FROM downloads
		GROUP BY format
		ORDER BY count DESC
		LIMIT 3
	`)
	if err == nil {
		defer formatRows.Close()
		var formats []gin.H
		for formatRows.Next() {
			var format string
			var count int64
			if formatRows.Scan(&format, &count) == nil {
				percentage := float64(count) / float64(totalDownloads) * 100
				formats = append(formats, gin.H{
					"format":     format,
					"count":      count,
					"percentage": percentage,
				})
			}
		}
		summary["popular_formats"] = formats
	}

	c.JSON(http.StatusOK, summary)
}

// GET /api/v1/analytics/health
func (h *AnalyticsHandler) GetHealthScore(c *gin.Context) {
	// Calculate overall system health score
	score := models.HealthScore{
		Categories:      make(map[string]int),
		Issues:          []string{},
		Recommendations: []string{},
		LastUpdated:     time.Now(),
	}

	// Database health (check for recent activity)
	var recentActivity int64
	if err := h.DB.QueryRow(`SELECT COUNT(*) FROM shows WHERE created_at >= datetime('now', '-7 days')`).Scan(&recentActivity); err != nil {
		log.Printf("Warning: failed to get recent activity: %v", err)
		recentActivity = 0
	}
	if recentActivity > 0 {
		score.Categories["database"] = 90
	} else {
		score.Categories["database"] = 60
		score.Issues = append(score.Issues, "No new shows added in the last week")
		score.Recommendations = append(score.Recommendations, "Consider running a catalog refresh")
	}

	// Download health (check success rate)
	var totalDownloads, completedDownloads int64
	if err := h.DB.QueryRow(`
		SELECT COUNT(*), COUNT(CASE WHEN status = 'completed' THEN 1 END)
		FROM downloads WHERE created_at >= datetime('now', '-7 days')
	`).Scan(&totalDownloads, &completedDownloads); err != nil {
		log.Printf("Warning: failed to get download health stats: %v", err)
		totalDownloads = 0
		completedDownloads = 0
	}

	if totalDownloads > 0 {
		successRate := float64(completedDownloads) / float64(totalDownloads) * 100
		if successRate > 90 {
			score.Categories["downloads"] = 95
		} else if successRate > 70 {
			score.Categories["downloads"] = 75
		} else {
			score.Categories["downloads"] = 50
			score.Issues = append(score.Issues, fmt.Sprintf("Download success rate is %.1f%%", successRate))
			score.Recommendations = append(score.Recommendations, "Check download system configuration")
		}
	} else {
		score.Categories["downloads"] = 80 // Neutral if no downloads
	}

	// Monitoring health
	var activeMonitors int64
	if err := h.DB.QueryRow(`SELECT COUNT(*) FROM artist_monitors WHERE status = 'active'`).Scan(&activeMonitors); err != nil {
		log.Printf("Warning: failed to get monitoring health stats: %v", err)
		activeMonitors = 0
	}
	if activeMonitors > 0 {
		score.Categories["monitoring"] = 85
	} else {
		score.Categories["monitoring"] = 40
		score.Issues = append(score.Issues, "No active artist monitors")
		score.Recommendations = append(score.Recommendations, "Set up monitoring for favorite artists")
	}

	// Storage health (simplified)
	metrics, err := h.AnalyticsService.GetSystemMetrics()
	if err == nil && metrics.AvailableStorage > 1.0 { // > 1GB free
		score.Categories["storage"] = 90
	} else {
		score.Categories["storage"] = 70
		if err == nil && metrics.AvailableStorage < 1.0 {
			score.Issues = append(score.Issues, "Low disk space")
			score.Recommendations = append(score.Recommendations, "Clean up old downloads or expand storage")
		}
	}

	// Calculate overall score
	total := 0
	count := 0
	for _, categoryScore := range score.Categories {
		total += categoryScore
		count++
	}
	if count > 0 {
		score.Overall = total / count
	}

	c.JSON(http.StatusOK, score)
}
