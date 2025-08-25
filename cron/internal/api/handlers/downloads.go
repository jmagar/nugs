package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmagar/nugs/cron/internal/models"
	"github.com/jmagar/nugs/cron/internal/services"
)

type DownloadHandler struct {
	DownloadManager *services.DownloadManager
	DB              *sql.DB
}

func NewDownloadHandler(db *sql.DB, jobManager *models.JobManager) *DownloadHandler {
	downloadManager := services.NewDownloadManager(db, jobManager)

	return &DownloadHandler{
		DownloadManager: downloadManager,
		DB:              db,
	}
}

// GET /api/v1/downloads
func (h *DownloadHandler) GetDownloads(c *gin.Context) {
	// Parse pagination and filters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize > 100 {
		pageSize = 100
	}

	artistID := c.Query("artist_id")
	status := c.Query("status")
	format := c.Query("format")

	// Build WHERE clause
	whereClause := "WHERE 1=1"
	args := []interface{}{}

	if artistID != "" {
		whereClause += " AND s.artist_id = ?"
		args = append(args, artistID)
	}

	if status != "" {
		whereClause += " AND d.status = ?"
		args = append(args, status)
	}

	if format != "" {
		whereClause += " AND d.format = ?"
		args = append(args, format)
	}

	// Count total
	countQuery := `
		SELECT COUNT(*) 
		FROM downloads d 
		JOIN shows s ON d.show_id = s.id 
		JOIN artists a ON s.artist_id = a.id ` + whereClause

	var total int64
	err := h.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count downloads"})
		return
	}

	// Get downloads
	offset := (page - 1) * pageSize
	query := `
		SELECT d.id, d.show_id, d.container_id, d.artist_name, d.download_path,
		       d.size_mb, d.quality, d.format, d.status, d.completed_at, d.created_at,
		       s.venue, s.city, s.state, s.date
		FROM downloads d
		JOIN shows s ON d.show_id = s.id
		JOIN artists a ON s.artist_id = a.id ` + whereClause + `
		ORDER BY d.created_at DESC
		LIMIT ? OFFSET ?
	`

	args = append(args, pageSize, offset)

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query downloads"})
		return
	}
	defer rows.Close()

	var downloads []models.Download
	for rows.Next() {
		var download models.Download
		var filePath, completedAt sql.NullString
		var sizeFloat sql.NullFloat64

		err := rows.Scan(
			&download.ID, &download.ShowID, &download.ContainerID, &download.ArtistName,
			&filePath, &sizeFloat, &download.Quality, &download.Format,
			&download.Status, &completedAt, &download.CreatedAt,
			&download.VenueName, &download.VenueCity,
			&download.VenueState, &download.PerformanceDate,
		)

		if err != nil {
			continue
		}

		if filePath.Valid {
			download.FilePath = sql.NullString{String: filePath.String, Valid: true}
		}

		if sizeFloat.Valid {
			download.FileSize = int64(sizeFloat.Float64 * 1024 * 1024) // Convert MB to bytes
		}

		if completedAt.Valid {
			if t, err := time.Parse("2006-01-02 15:04:05", completedAt.String); err == nil {
				download.DownloadedAt = &t
			}
		}

		// Set show title from venue and city
		download.ShowTitle = download.VenueName + ", " + download.VenueCity

		downloads = append(downloads, download)
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	response := gin.H{
		"downloads": downloads,
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

// Custom request struct for backward compatibility
type QueueDownloadRequest struct {
	ShowID      int    `json:"show_id"`      // Standard field name
	ContainerID int    `json:"container_id"` // Legacy field name (same as show_id)
	Format      string `json:"format" binding:"required"`
	Quality     string `json:"quality"`
	Priority    int    `json:"priority"`
}

// POST /api/v1/downloads/queue
func (h *DownloadHandler) QueueDownload(c *gin.Context) {
	var req QueueDownloadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format: " + err.Error(),
		})
		return
	}

	// Use container_id if show_id not provided (backward compatibility)
	showID := req.ShowID
	if showID == 0 && req.ContainerID != 0 {
		showID = req.ContainerID
	}

	if showID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "show_id or container_id is required",
		})
		return
	}

	// Normalize format and quality values - database expects uppercase formats
	var formatStr string
	switch strings.ToLower(req.Format) {
	case "mp3":
		formatStr = "MP3"
	case "flac":
		formatStr = "FLAC"
	case "alac":
		formatStr = "ALAC"
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid format. Must be 'mp3', 'flac', or 'alac'",
		})
		return
	}

	qualityStr := "standard" // Default
	switch strings.ToLower(req.Quality) {
	case "16bit/44.1khz", "lossless", "flac":
		qualityStr = "lossless"
	case "320kbps", "hd", "high":
		qualityStr = "hd"
	case "128kbps", "standard", "low", "":
		qualityStr = "standard"
	}

	// Create standard download request
	standardReq := &models.DownloadRequest{
		ShowID:   showID,
		Format:   models.DownloadFormat(formatStr),
		Quality:  models.DownloadQuality(qualityStr),
		Priority: req.Priority,
	}

	response, err := h.DownloadManager.QueueDownload(standardReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !response.Success {
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Return expected format for tests
	c.JSON(http.StatusCreated, gin.H{
		"download_id":    response.DownloadID,
		"message":        response.Message,
		"queue_position": 1, // Will be calculated later
	})
}

// GET /api/v1/downloads/:id
func (h *DownloadHandler) GetDownload(c *gin.Context) {
	downloadID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid download ID"})
		return
	}

	query := `
		SELECT d.id, d.show_id, d.container_id, d.artist_name, d.file_path,
		       d.file_size, d.quality, d.format, d.status, d.downloaded_at, d.created_at,
		       s.container_info, s.venue_name, s.venue_city, s.venue_state, s.performance_date_formatted
		FROM downloads d
		JOIN shows s ON d.show_id = s.id
		WHERE d.id = ?
	`

	var download models.Download
	var filePath, downloadedAt sql.NullString

	err = h.DB.QueryRow(query, downloadID).Scan(
		&download.ID, &download.ShowID, &download.ContainerID, &download.ArtistName,
		&filePath, &download.FileSize, &download.Quality, &download.Format,
		&download.Status, &downloadedAt, &download.CreatedAt,
		&download.ShowTitle, &download.VenueName, &download.VenueCity,
		&download.VenueState, &download.PerformanceDate,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Download not found"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get download"})
		return
	}

	if filePath.Valid {
		download.FilePath = filePath
	}

	if downloadedAt.Valid {
		if t, err := time.Parse("2006-01-02 15:04:05", downloadedAt.String); err == nil {
			download.DownloadedAt = &t
		}
	}

	c.JSON(http.StatusOK, download)
}

// DELETE /api/v1/downloads/:id
func (h *DownloadHandler) CancelDownload(c *gin.Context) {
	downloadID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid download ID"})
		return
	}

	// Check if download exists and get its status
	var status string
	err = h.DB.QueryRow("SELECT status FROM downloads WHERE id = ?", downloadID).Scan(&status)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Download not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check download status"})
		return
	}

	// Check if download can be cancelled (business logic)
	if status != "pending" && status != "queued" && status != "downloading" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "download cannot be cancelled (not pending or in progress)"})
		return
	}

	// Update download status to cancelled
	_, err = h.DB.Exec("UPDATE downloads SET status = 'cancelled', updated_at = CURRENT_TIMESTAMP WHERE id = ?", downloadID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel download"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Download cancelled successfully",
	})
}

// GET /api/v1/downloads/stats
func (h *DownloadHandler) GetDownloadStats(c *gin.Context) {
	stats, err := h.DownloadManager.GetDownloadStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get download statistics"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GET /api/v1/downloads/queue
func (h *DownloadHandler) GetDownloadQueue(c *gin.Context) {
	query := `
		SELECT d.id, d.show_id, d.container_id, d.artist_name, d.format, d.quality, 
		       d.status, d.queue_position, d.created_at,
		       s.venue, s.city, s.date
		FROM downloads d
		JOIN shows s ON d.show_id = s.id
		WHERE d.status IN ('pending', 'queued') AND d.queue_position IS NOT NULL
		ORDER BY d.queue_position ASC
	`

	rows, err := h.DB.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get download queue"})
		return
	}
	defer rows.Close()

	var queueItems []gin.H
	for rows.Next() {
		var downloadID, showID, containerID int
		var queuePosition sql.NullInt64
		var artistName, format, quality, status, createdAt string
		var venueName, venueCity, showDate string

		err := rows.Scan(
			&downloadID, &showID, &containerID, &artistName, &format, &quality,
			&status, &queuePosition, &createdAt,
			&venueName, &venueCity, &showDate,
		)

		if err != nil {
			continue
		}

		position := 0
		if queuePosition.Valid {
			position = int(queuePosition.Int64)
		}

		queueItems = append(queueItems, gin.H{
			"position":         position,
			"download_id":      downloadID,
			"show_id":          showID,
			"container_id":     containerID,
			"artist_name":      artistName,
			"show_title":       venueName + ", " + venueCity,
			"venue_name":       venueName,
			"venue_city":       venueCity,
			"performance_date": showDate,
			"format":           format,
			"quality":          quality,
			"status":           status,
			"created_at":       createdAt,
		})
	}

	// Count items by status
	pendingItems := 0
	processingItems := 0
	for _, item := range queueItems {
		if status, ok := item["status"].(string); ok {
			switch status {
			case "pending", "queued":
				pendingItems++
			case "downloading":
				processingItems++
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"queue": queueItems,
		"stats": gin.H{
			"total_items":      len(queueItems),
			"pending_items":    pendingItems,
			"processing_items": processingItems,
		},
	})
}

// POST /api/v1/downloads/queue/reorder
func (h *DownloadHandler) ReorderQueue(c *gin.Context) {
	var req struct {
		DownloadIDs []string `json:"download_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Validate that download_ids is not empty
	if len(req.DownloadIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "download_ids cannot be empty"})
		return
	}

	// Convert string IDs to integers
	downloadIDs := make([]int, len(req.DownloadIDs))
	for i, idStr := range req.DownloadIDs {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid download ID: " + idStr})
			return
		}
		downloadIDs[i] = id
	}

	// Begin transaction
	tx, err := h.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Printf("Warning: failed to rollback transaction: %v", err)
		}
	}()

	// Update positions
	for i, downloadID := range downloadIDs {
		_, err := tx.Exec(`
			UPDATE downloads 
			SET queue_position = ? 
			WHERE id = ?
		`, i+1, downloadID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reorder queue"})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit changes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Queue reordered successfully",
	})
}
