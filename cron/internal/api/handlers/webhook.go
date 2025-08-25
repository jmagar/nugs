package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmagar/nugs/cron/internal/models"
	"github.com/jmagar/nugs/cron/internal/services"
)

type WebhookHandler struct {
	WebhookService *services.WebhookService
	DB             *sql.DB
}

func NewWebhookHandler(db *sql.DB, jobManager *models.JobManager) *WebhookHandler {
	webhookService := services.NewWebhookService(db, jobManager)

	return &WebhookHandler{
		WebhookService: webhookService,
		DB:             db,
	}
}

// POST /api/v1/webhooks
func (h *WebhookHandler) CreateWebhook(c *gin.Context) {
	var req models.WebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format: " + err.Error(),
		})
		return
	}

	response, err := h.WebhookService.CreateWebhook(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create webhook"})
		return
	}

	if !response.Success {
		c.JSON(http.StatusBadRequest, response)
		return
	}

	c.JSON(http.StatusCreated, response)
}

// GET /api/v1/webhooks
func (h *WebhookHandler) GetWebhooks(c *gin.Context) {
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
	event := c.Query("event")

	// Build WHERE clause
	whereClause := "WHERE 1=1"
	args := []interface{}{}

	if status != "" {
		whereClause += " AND status = ?"
		args = append(args, status)
	}

	if event != "" {
		whereClause += " AND events LIKE ?"
		args = append(args, "%\""+event+"\"%")
	}

	// Count total
	countQuery := "SELECT COUNT(*) FROM webhooks " + whereClause
	var total int64
	err := h.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count webhooks"})
		return
	}

	// Get webhooks
	offset := (page - 1) * pageSize
	query := `
		SELECT w.id, w.description, w.url, w.events, w.status, w.secret, 
		       w.timeout, w.retry_count, w.last_delivery, w.total_deliveries, w.failed_deliveries,
		       w.created_at, w.updated_at,
		       COUNT(wd.id) as total_fired,
		       COUNT(CASE WHEN wd.success = 1 THEN 1 END) as success_count
		FROM webhooks w
		LEFT JOIN webhook_deliveries wd ON w.id = wd.webhook_id ` + whereClause + `
		GROUP BY w.id, w.description, w.url, w.events, w.status, w.secret, w.timeout, w.retry_count, w.last_delivery, w.total_deliveries, w.failed_deliveries, w.created_at, w.updated_at
		ORDER BY w.created_at DESC
		LIMIT ? OFFSET ?
	`

	args = append(args, pageSize, offset)

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query webhooks"})
		return
	}
	defer rows.Close()

	var webhooks []models.Webhook
	for rows.Next() {
		var webhook models.Webhook
		var eventsJSON string
		var lastFired sql.NullString
		var secret sql.NullString

		err := rows.Scan(
			&webhook.ID, &webhook.Description, &webhook.URL, &eventsJSON, &webhook.Status,
			&secret, &webhook.Timeout, &webhook.RetryCount,
			&lastFired, &webhook.TotalDeliveries, &webhook.FailedDeliveries,
			&webhook.CreatedAt, &webhook.UpdatedAt, &webhook.TotalFired, &webhook.SuccessCount,
		)

		if err != nil {
			continue
		}

		// Parse events
		var events []models.WebhookEvent
		if json.Unmarshal([]byte(eventsJSON), &events) == nil {
			webhook.Events = events
		}

		// Handle secret (don't expose actual value)
		if secret.Valid && secret.String != "" {
			webhook.Secret = "***" // Mask the secret
		}

		// Parse last fired time
		if lastFired.Valid {
			if t, err := time.Parse("2006-01-02 15:04:05", lastFired.String); err == nil {
				webhook.LastFired = &t
			}
		}

		// Calculate failure rate
		if webhook.TotalFired > 0 {
			webhook.FailureRate = float64(webhook.TotalFired-webhook.SuccessCount) / float64(webhook.TotalFired) * 100
		}

		webhooks = append(webhooks, webhook)
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	response := gin.H{
		"data":        webhooks,
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": totalPages,
		"has_next":    page < totalPages,
		"has_prev":    page > 1,
	}

	c.JSON(http.StatusOK, response)
}

// GET /api/v1/webhooks/:id
func (h *WebhookHandler) GetWebhook(c *gin.Context) {
	webhookID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook ID"})
		return
	}

	query := `
		SELECT w.id, w.description, w.url, w.events, w.status, w.secret, 
		       w.timeout, w.retry_count, w.last_delivery, w.total_deliveries, w.failed_deliveries,
		       w.created_at, w.updated_at,
		       COUNT(wd.id) as total_fired,
		       COUNT(CASE WHEN wd.success = 1 THEN 1 END) as success_count
		FROM webhooks w
		LEFT JOIN webhook_deliveries wd ON w.id = wd.webhook_id
		WHERE w.id = ?
		GROUP BY w.id, w.description, w.url, w.events, w.status, w.secret, w.timeout, w.retry_count, w.last_delivery, w.total_deliveries, w.failed_deliveries, w.created_at, w.updated_at
	`

	var webhook models.Webhook
	var eventsJSON string
	var lastFired sql.NullString
	var secret sql.NullString

	err = h.DB.QueryRow(query, webhookID).Scan(
		&webhook.ID, &webhook.Description, &webhook.URL, &eventsJSON, &webhook.Status,
		&secret, &webhook.Timeout, &webhook.RetryCount,
		&lastFired, &webhook.TotalDeliveries, &webhook.FailedDeliveries,
		&webhook.CreatedAt, &webhook.UpdatedAt, &webhook.TotalFired, &webhook.SuccessCount,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Webhook not found"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get webhook"})
		return
	}

	// Parse events
	var events []models.WebhookEvent
	if json.Unmarshal([]byte(eventsJSON), &events) == nil {
		webhook.Events = events
	}

	// Handle secret (don't expose actual value)
	if secret.Valid && secret.String != "" {
		webhook.Secret = "***" // Mask the secret
	}

	// Parse last fired time
	if lastFired.Valid {
		if t, err := time.Parse("2006-01-02 15:04:05", lastFired.String); err == nil {
			webhook.LastFired = &t
		}
	}

	// Calculate failure rate
	if webhook.TotalFired > 0 {
		webhook.FailureRate = float64(webhook.TotalFired-webhook.SuccessCount) / float64(webhook.TotalFired) * 100
	}

	c.JSON(http.StatusOK, webhook)
}

// PUT /api/v1/webhooks/:id
func (h *WebhookHandler) UpdateWebhook(c *gin.Context) {
	webhookID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook ID"})
		return
	}

	var req models.WebhookUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format: " + err.Error(),
		})
		return
	}

	err = h.WebhookService.UpdateWebhook(webhookID, &req)
	if err != nil {
		if err.Error() == "webhook not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Webhook not found"})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Webhook updated successfully",
	})
}

// DELETE /api/v1/webhooks/:id
func (h *WebhookHandler) DeleteWebhook(c *gin.Context) {
	webhookID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook ID"})
		return
	}

	err = h.WebhookService.DeleteWebhook(webhookID)
	if err != nil {
		if err.Error() == "webhook not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Webhook not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete webhook"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Webhook deleted successfully",
	})
}

// POST /api/v1/webhooks/:id/test
func (h *WebhookHandler) TestWebhook(c *gin.Context) {
	webhookID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook ID"})
		return
	}

	var req models.WebhookTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format: " + err.Error(),
		})
		return
	}

	result, err := h.WebhookService.TestWebhook(webhookID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to test webhook: " + err.Error(),
		})
		return
	}

	if !result.Success && result.Error == "Webhook not found" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Webhook not found"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GET /api/v1/webhooks/:id/deliveries
func (h *WebhookHandler) GetWebhookDeliveries(c *gin.Context) {
	webhookID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook ID"})
		return
	}

	// Parse pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize > 100 {
		pageSize = 100
	}

	event := c.Query("event")
	success := c.Query("success")

	// Build WHERE clause
	whereClause := "WHERE wd.webhook_id = ?"
	args := []interface{}{webhookID}

	if event != "" {
		whereClause += " AND wd.event = ?"
		args = append(args, event)
	}

	if success != "" {
		whereClause += " AND wd.success = ?"
		args = append(args, success == "true")
	}

	// Count total
	countQuery := "SELECT COUNT(*) FROM webhook_deliveries wd " + whereClause
	var total int64
	err = h.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count deliveries"})
		return
	}

	// Get deliveries
	offset := (page - 1) * pageSize
	query := `
		SELECT wd.id, wd.webhook_id, wd.event, wd.url, wd.payload, wd.headers,
		       wd.status_code, wd.response, wd.error, wd.duration_ms, wd.attempt, 
		       wd.success, wd.created_at, w.description as webhook_name
		FROM webhook_deliveries wd
		JOIN webhooks w ON wd.webhook_id = w.id ` + whereClause + `
		ORDER BY wd.created_at DESC
		LIMIT ? OFFSET ?
	`

	args = append(args, pageSize, offset)

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query deliveries"})
		return
	}
	defer rows.Close()

	var deliveries []models.WebhookDelivery
	for rows.Next() {
		var delivery models.WebhookDelivery
		var payload, headers, response, errorMsg sql.NullString

		err := rows.Scan(
			&delivery.ID, &delivery.WebhookID, &delivery.Event, &delivery.URL,
			&payload, &headers, &delivery.StatusCode, &response, &errorMsg,
			&delivery.Duration, &delivery.Attempt, &delivery.Success,
			&delivery.CreatedAt, &delivery.WebhookName,
		)

		if err != nil {
			continue
		}

		if payload.Valid {
			delivery.Payload = payload.String
		}
		if headers.Valid {
			delivery.Headers = headers.String
		}
		if response.Valid {
			delivery.Response = response.String
		}
		if errorMsg.Valid {
			delivery.Error = errorMsg.String
		}

		deliveries = append(deliveries, delivery)
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	response := gin.H{
		"data":        deliveries,
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": totalPages,
		"has_next":    page < totalPages,
		"has_prev":    page > 1,
	}

	c.JSON(http.StatusOK, response)
}

// GET /api/v1/webhooks/deliveries
func (h *WebhookHandler) GetAllDeliveries(c *gin.Context) {
	// Parse pagination and filters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize > 100 {
		pageSize = 100
	}

	webhookID := c.Query("webhook_id")
	event := c.Query("event")
	success := c.Query("success")

	// Build WHERE clause
	whereClause := "WHERE 1=1"
	args := []interface{}{}

	if webhookID != "" {
		whereClause += " AND wd.webhook_id = ?"
		args = append(args, webhookID)
	}

	if event != "" {
		whereClause += " AND wd.event = ?"
		args = append(args, event)
	}

	if success != "" {
		whereClause += " AND wd.success = ?"
		args = append(args, success == "true")
	}

	// Count total
	countQuery := "SELECT COUNT(*) FROM webhook_deliveries wd " + whereClause
	var total int64
	err := h.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count deliveries"})
		return
	}

	// Get deliveries
	offset := (page - 1) * pageSize
	query := `
		SELECT wd.id, wd.webhook_id, wd.event, wd.url, wd.status_code, 
		       wd.duration_ms, wd.attempt, wd.success, wd.created_at, 
		       w.description as webhook_name
		FROM webhook_deliveries wd
		JOIN webhooks w ON wd.webhook_id = w.id ` + whereClause + `
		ORDER BY wd.created_at DESC
		LIMIT ? OFFSET ?
	`

	args = append(args, pageSize, offset)

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query deliveries"})
		return
	}
	defer rows.Close()

	var deliveries []gin.H
	for rows.Next() {
		var id, webhookID, statusCode, duration, attempt int
		var event, url, webhookName string
		var success bool
		var createdAt time.Time

		err := rows.Scan(&id, &webhookID, &event, &url, &statusCode, &duration,
			&attempt, &success, &createdAt, &webhookName)

		if err != nil {
			continue
		}

		deliveries = append(deliveries, gin.H{
			"id":           id,
			"webhook_id":   webhookID,
			"webhook_name": webhookName,
			"event":        event,
			"url":          url,
			"status_code":  statusCode,
			"duration_ms":  duration,
			"attempt":      attempt,
			"success":      success,
			"created_at":   createdAt,
		})
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	response := gin.H{
		"data":        deliveries,
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": totalPages,
		"has_next":    page < totalPages,
		"has_prev":    page > 1,
	}

	c.JSON(http.StatusOK, response)
}

// GET /api/v1/webhooks/stats
func (h *WebhookHandler) GetWebhookStats(c *gin.Context) {
	stats, err := h.WebhookService.GetWebhookStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get webhook statistics",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GET /api/v1/webhooks/events
func (h *WebhookHandler) GetAvailableEvents(c *gin.Context) {
	events := []gin.H{
		{
			"event":       models.WebhookEventNewShow,
			"description": "Triggered when new shows are found for monitored artists",
		},
		{
			"event":       models.WebhookEventDownloadComplete,
			"description": "Triggered when a download completes successfully",
		},
		{
			"event":       models.WebhookEventDownloadFailed,
			"description": "Triggered when a download fails",
		},
		{
			"event":       models.WebhookEventCatalogRefresh,
			"description": "Triggered when catalog refresh completes",
		},
		{
			"event":       models.WebhookEventMonitorAlert,
			"description": "Triggered when monitoring alerts are generated",
		},
		{
			"event":       models.WebhookEventSystemAlert,
			"description": "Triggered for system-level alerts and issues",
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"events": events,
		"total":  len(events),
	})
}
