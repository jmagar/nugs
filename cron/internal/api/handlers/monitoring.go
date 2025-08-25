package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmagar/nugs/cron/internal/models"
	"github.com/jmagar/nugs/cron/internal/services"
)

type MonitoringHandler struct {
	MonitoringService *services.MonitoringService
	DB                *sql.DB
}

func NewMonitoringHandler(db *sql.DB, jobManager *models.JobManager) *MonitoringHandler {
	monitoringService := services.NewMonitoringService(db, jobManager)

	return &MonitoringHandler{
		MonitoringService: monitoringService,
		DB:                db,
	}
}

// POST /api/v1/monitoring/monitors
func (h *MonitoringHandler) CreateMonitor(c *gin.Context) {
	var req models.MonitorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format: " + err.Error(),
		})
		return
	}

	response, err := h.MonitoringService.CreateMonitor(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create monitor"})
		return
	}

	if !response.Success {
		c.JSON(http.StatusBadRequest, response)
		return
	}

	c.JSON(http.StatusCreated, response)
}

// POST /api/v1/monitoring/monitors/bulk
func (h *MonitoringHandler) CreateBulkMonitors(c *gin.Context) {
	var req models.BulkMonitorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format: " + err.Error(),
		})
		return
	}

	response, err := h.MonitoringService.CreateBulkMonitors(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create bulk monitors"})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GET /api/v1/monitoring/monitors
func (h *MonitoringHandler) GetMonitors(c *gin.Context) {
	// Parse pagination and filters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize > 100 {
		pageSize = 100
	}

	status := c.Query("status")
	artistName := c.Query("artist_name")

	// Build WHERE clause
	whereClause := "WHERE 1=1"
	args := []interface{}{}

	if status != "" {
		whereClause += " AND m.status = ?"
		args = append(args, status)
	}

	if artistName != "" {
		whereClause += " AND a.name LIKE ?"
		args = append(args, "%"+artistName+"%")
	}

	// Count total
	countQuery := `
		SELECT COUNT(*) 
		FROM monitors m 
		JOIN artists a ON m.artist_id = a.id 
	` + whereClause
	var total int64
	err := h.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count monitors"})
		return
	}

	// Get monitors
	offset := (page - 1) * pageSize
	query := `
		SELECT m.id, m.artist_id, a.name as artist_name, m.status, m.settings,
		       m.last_check, m.shows_found, m.alerts_sent, m.created_at, m.updated_at
		FROM monitors m
		JOIN artists a ON m.artist_id = a.id ` + whereClause + `
		ORDER BY m.created_at DESC
		LIMIT ? OFFSET ?
	`

	args = append(args, pageSize, offset)

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query monitors"})
		return
	}
	defer rows.Close()

	var monitors []gin.H
	for rows.Next() {
		var id, artistID, showsFound, alertsSent int
		var artistName, status, settings, createdAt, updatedAt string
		var lastCheck sql.NullString

		err := rows.Scan(
			&id, &artistID, &artistName, &status, &settings,
			&lastCheck, &showsFound, &alertsSent, &createdAt, &updatedAt,
		)

		if err != nil {
			continue
		}

		monitor := gin.H{
			"id":          id,
			"artist_id":   artistID,
			"artist_name": artistName,
			"status":      status,
			"settings":    settings,
			"shows_found": showsFound,
			"alerts_sent": alertsSent,
			"created_at":  createdAt,
			"updated_at":  updatedAt,
		}

		if lastCheck.Valid {
			monitor["last_check"] = lastCheck.String
		}

		monitors = append(monitors, monitor)
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	response := gin.H{
		"monitors": monitors,
		"pagination": gin.H{
			"page":        page,
			"page_size":   pageSize,
			"total":       total,
			"total_pages": totalPages,
			"has_next":    page < totalPages,
			"has_prev":    page > 1,
		},
	}

	c.JSON(http.StatusOK, response)
}

// GET /api/v1/monitoring/monitors/:id
func (h *MonitoringHandler) GetMonitor(c *gin.Context) {
	monitorID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid monitor ID"})
		return
	}

	query := `
		SELECT id, artist_id, artist_name, status, check_interval, last_checked,
		       last_new_show, total_shows, new_shows_found, notify_new_shows,
		       notify_show_updates, created_at, updated_at
		FROM artist_monitors 
		WHERE id = ?
	`

	var monitor models.ArtistMonitor
	var lastChecked, lastNewShow sql.NullString

	err = h.DB.QueryRow(query, monitorID).Scan(
		&monitor.ID, &monitor.ArtistID, &monitor.ArtistName, &monitor.Status,
		&monitor.CheckInterval, &lastChecked, &lastNewShow, &monitor.TotalShows,
		&monitor.NewShowsFound, &monitor.NotifyNewShows, &monitor.NotifyShowUpdates,
		&monitor.CreatedAt, &monitor.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Monitor not found"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get monitor"})
		return
	}

	if lastChecked.Valid {
		if t, err := time.Parse("2006-01-02 15:04:05", lastChecked.String); err == nil {
			monitor.LastChecked = &t
		}
	}

	if lastNewShow.Valid {
		if t, err := time.Parse("2006-01-02 15:04:05", lastNewShow.String); err == nil {
			monitor.LastNewShow = &t
		}
	}

	c.JSON(http.StatusOK, monitor)
}

// PUT /api/v1/monitoring/monitors/:id
func (h *MonitoringHandler) UpdateMonitor(c *gin.Context) {
	monitorID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid monitor ID"})
		return
	}

	var req models.MonitorUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format: " + err.Error(),
		})
		return
	}

	err = h.MonitoringService.UpdateMonitor(monitorID, &req)
	if err != nil {
		if err.Error() == "monitor not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Monitor not found"})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Monitor updated successfully",
	})
}

// DELETE /api/v1/monitoring/monitors/:id
func (h *MonitoringHandler) DeleteMonitor(c *gin.Context) {
	monitorID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid monitor ID"})
		return
	}

	err = h.MonitoringService.DeleteMonitor(monitorID)
	if err != nil {
		if err.Error() == "monitor not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Monitor not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete monitor"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Monitor deleted successfully",
	})
}

// POST /api/v1/monitoring/check/all
func (h *MonitoringHandler) CheckAllMonitors(c *gin.Context) {
	job := h.MonitoringService.CheckAllMonitors()

	c.JSON(http.StatusAccepted, gin.H{
		"success": true,
		"job_id":  job.ID,
		"message": "Monitoring check started",
		"status":  job.Status,
	})
}

// POST /api/v1/monitoring/check/artist/:id
func (h *MonitoringHandler) CheckArtist(c *gin.Context) {
	artistID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid artist ID"})
		return
	}

	result, err := h.MonitoringService.CheckArtist(artistID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GET /api/v1/monitoring/alerts
func (h *MonitoringHandler) GetAlerts(c *gin.Context) {
	// Parse pagination and filters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize > 100 {
		pageSize = 100
	}

	alertType := c.Query("alert_type")
	acknowledged := c.Query("acknowledged")
	artistID := c.Query("artist_id")

	// Build WHERE clause
	whereClause := "WHERE 1=1"
	args := []interface{}{}

	if alertType != "" {
		whereClause += " AND ma.type = ?"
		args = append(args, alertType)
	}

	if acknowledged != "" {
		whereClause += " AND ma.acknowledged = ?"
		args = append(args, acknowledged == "true")
	}

	if artistID != "" {
		whereClause += " AND ma.artist_id = ?"
		args = append(args, artistID)
	}

	// Count total
	countQuery := `
		SELECT COUNT(*) 
		FROM monitor_alerts ma 
		JOIN monitors m ON ma.monitor_id = m.id ` + whereClause

	var total int64
	err := h.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count alerts"})
		return
	}

	// Get alerts
	offset := (page - 1) * pageSize
	query := `
		SELECT ma.id, ma.monitor_id, ma.artist_id, ma.type, ma.title,
		       ma.message, ma.data, ma.acknowledged, ma.created_at,
		       a.name as artist_name, COALESCE(s.venue, '') as show_title
		FROM monitor_alerts ma
		JOIN monitors m ON ma.monitor_id = m.id
		JOIN artists a ON ma.artist_id = a.id
		LEFT JOIN shows s ON ma.artist_id = s.artist_id ` + whereClause + `
		ORDER BY ma.created_at DESC
		LIMIT ? OFFSET ?
	`

	args = append(args, pageSize, offset)

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query alerts"})
		return
	}
	defer rows.Close()

	var alerts []gin.H
	for rows.Next() {
		var id, monitorID, artistID int
		var alertType, title, message, data, createdAt, artistName, showTitle string
		var acknowledged bool

		err := rows.Scan(
			&id, &monitorID, &artistID, &alertType, &title,
			&message, &data, &acknowledged, &createdAt,
			&artistName, &showTitle,
		)

		if err != nil {
			continue
		}

		alert := gin.H{
			"id":           id,
			"monitor_id":   monitorID,
			"artist_id":    artistID,
			"type":         alertType,
			"title":        title,
			"message":      message,
			"data":         data,
			"acknowledged": acknowledged,
			"created_at":   createdAt,
			"artist_name":  artistName,
			"show_title":   showTitle,
		}

		alerts = append(alerts, alert)
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	response := gin.H{
		"alerts": alerts,
		"pagination": gin.H{
			"page":        page,
			"page_size":   pageSize,
			"total":       total,
			"total_pages": totalPages,
			"has_next":    page < totalPages,
			"has_prev":    page > 1,
		},
	}

	c.JSON(http.StatusOK, response)
}

// PUT /api/v1/monitoring/alerts/:id/acknowledge
func (h *MonitoringHandler) AcknowledgeAlert(c *gin.Context) {
	alertID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid alert ID"})
		return
	}

	result, err := h.DB.Exec("UPDATE monitor_alerts SET acknowledged = 1 WHERE id = ?", alertID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to acknowledge alert"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Alert not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Alert acknowledged successfully",
	})
}

// GET /api/v1/monitoring/stats
func (h *MonitoringHandler) GetMonitoringStats(c *gin.Context) {
	stats, err := h.MonitoringService.GetMonitorStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get monitoring statistics"})
		return
	}

	c.JSON(http.StatusOK, stats)
}
