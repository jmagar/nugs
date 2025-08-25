package services

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jmagar/nugs/cron/internal/models"
)

type WebhookService struct {
	DB         *sql.DB
	JobManager *models.JobManager
	httpClient *http.Client
}

func NewWebhookService(db *sql.DB, jobManager *models.JobManager) *WebhookService {
	return &WebhookService{
		DB:         db,
		JobManager: jobManager,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *WebhookService) CreateWebhook(req *models.WebhookRequest) (*models.WebhookResponse, error) {
	// Set defaults
	if req.Timeout == 0 {
		req.Timeout = 10
	}
	if req.Retries == 0 {
		req.Retries = 3
	}

	// Validate events
	for _, event := range req.Events {
		if !s.isValidEvent(event) {
			return &models.WebhookResponse{
				Success: false,
				Error:   fmt.Sprintf("Invalid event type: %s", event),
			}, nil
		}
	}

	// Serialize events
	eventsJSON, err := json.Marshal(req.Events)
	if err != nil {
		return &models.WebhookResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to marshal events: %v", err),
		}, fmt.Errorf("failed to marshal events: %w", err)
	}

	// Insert webhook
	result, err := s.DB.Exec(`
		INSERT INTO webhooks (description, url, events, status, secret, timeout, retry_count, 
		                     total_deliveries, successful_deliveries, failed_deliveries, created_at, updated_at)
		VALUES (?, ?, ?, 'active', ?, ?, ?, 0, 0, 0, datetime('now'), datetime('now'))
	`, req.Name, req.URL, string(eventsJSON), req.Secret, req.Timeout, req.Retries)

	if err != nil {
		return &models.WebhookResponse{
			Success: false,
			Error:   "Failed to create webhook",
		}, err
	}

	webhookID, _ := result.LastInsertId()

	return &models.WebhookResponse{
		Success:   true,
		WebhookID: int(webhookID),
		Message:   "Webhook created successfully",
	}, nil
}

func (s *WebhookService) UpdateWebhook(webhookID int, req *models.WebhookUpdateRequest) error {
	updates := []string{}
	args := []interface{}{}

	// Prefer Description over Name for backward compatibility
	if req.Description != nil {
		updates = append(updates, "description = ?")
		args = append(args, *req.Description)
	} else if req.Name != nil {
		updates = append(updates, "description = ?")
		args = append(args, *req.Name)
	}

	if req.URL != nil {
		updates = append(updates, "url = ?")
		args = append(args, *req.URL)
	}

	if req.Events != nil {
		// Validate events
		for _, event := range *req.Events {
			if !s.isValidEvent(event) {
				return fmt.Errorf("invalid event type: %s", event)
			}
		}
		eventsJSON, err := json.Marshal(*req.Events)
		if err != nil {
			return fmt.Errorf("failed to marshal events: %w", err)
		}
		updates = append(updates, "events = ?")
		args = append(args, string(eventsJSON))
	}

	if req.Status != nil {
		updates = append(updates, "status = ?")
		args = append(args, *req.Status)
	}

	if req.Secret != nil {
		updates = append(updates, "secret = ?")
		args = append(args, *req.Secret)
	}

	// Error if headers update is attempted, as this column doesn't exist in the current schema
	if req.Headers != nil {
		return fmt.Errorf("updating headers is not supported")
	}

	if req.Timeout != nil {
		updates = append(updates, "timeout = ?")
		args = append(args, *req.Timeout)
	}

	if req.Retries != nil {
		updates = append(updates, "retry_count = ?")
		args = append(args, *req.Retries)
	}

	if len(updates) == 0 {
		return fmt.Errorf("no fields to update")
	}

	updates = append(updates, "updated_at = datetime('now')")
	args = append(args, webhookID)

	query := fmt.Sprintf("UPDATE webhooks SET %s WHERE id = ?", strings.Join(updates, ", "))

	result, err := s.DB.Exec(query, args...)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("webhook not found")
	}

	return nil
}

func (s *WebhookService) DeleteWebhook(webhookID int) error {
	result, err := s.DB.Exec("DELETE FROM webhooks WHERE id = ?", webhookID)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("webhook not found")
	}

	// Delete related deliveries
	if _, err := s.DB.Exec("DELETE FROM webhook_deliveries WHERE webhook_id = ?", webhookID); err != nil {
		log.Printf("Warning: failed to delete webhook deliveries: %v", err)
	}

	return nil
}

func (s *WebhookService) TriggerEvent(event models.WebhookEvent, data interface{}) error {
	// Get all webhooks that listen for this event
	rows, err := s.DB.Query(`
		SELECT id, name, url, events, secret, timeout, retries, headers
		FROM webhooks
		WHERE status = 'active' AND events LIKE ?
	`, "%\""+string(event)+"\"%")

	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var webhook models.Webhook
		var eventsJSON string

		err := rows.Scan(&webhook.ID, &webhook.Name, &webhook.URL, &eventsJSON,
			&webhook.Secret, &webhook.Timeout, &webhook.Retries, &webhook.Headers)
		if err != nil {
			continue
		}

		// Parse events to check if this webhook handles this event
		var events []models.WebhookEvent
		if json.Unmarshal([]byte(eventsJSON), &events) == nil {
			shouldTrigger := false
			for _, e := range events {
				if e == event {
					shouldTrigger = true
					break
				}
			}

			if shouldTrigger {
				// Trigger webhook asynchronously
				go s.deliverWebhook(&webhook, event, data, 1)
			}
		}
	}

	return nil
}

func (s *WebhookService) deliverWebhook(webhook *models.Webhook, event models.WebhookEvent, data interface{}, attempt int) {
	startTime := time.Now()

	// Create payload
	payload := models.WebhookPayload{
		Event:     event,
		Timestamp: time.Now(),
		Source:    "nugs-api/v1.0.0",
		Data:      data,
	}

	// Add signature if secret is provided
	payloadBytes, _ := json.Marshal(payload)
	if webhook.Secret != "" {
		signature := s.generateSignature(webhook.Secret, payloadBytes)
		payload.Signature = signature
		payloadBytes, _ = json.Marshal(payload) // Re-marshal with signature
	}

	// Prepare request
	req, err := http.NewRequest("POST", webhook.URL, bytes.NewReader(payloadBytes))
	if err != nil {
		s.recordDelivery(webhook.ID, event, webhook.URL, string(payloadBytes), "", 0, "", err.Error(), int(time.Since(startTime).Milliseconds()), attempt, false)
		return
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "nugs-api-webhook/1.0")
	req.Header.Set("X-Webhook-Event", string(event))
	req.Header.Set("X-Webhook-Delivery", fmt.Sprintf("%d", time.Now().Unix()))

	// Add custom headers
	if webhook.Headers != "" {
		var customHeaders map[string]string
		if json.Unmarshal([]byte(webhook.Headers), &customHeaders) == nil {
			for key, value := range customHeaders {
				req.Header.Set(key, value)
			}
		}
	}

	// Set signature header
	if webhook.Secret != "" && payload.Signature != "" {
		req.Header.Set("X-Hub-Signature-256", "sha256="+payload.Signature)
	}

	// Set timeout
	client := &http.Client{
		Timeout: time.Duration(webhook.Timeout) * time.Second,
	}

	// Make request
	resp, err := client.Do(req)
	duration := int(time.Since(startTime).Milliseconds())

	if err != nil {
		// Delivery failed - record and retry if needed
		s.recordDelivery(webhook.ID, event, webhook.URL, string(payloadBytes), "", 0, "", err.Error(), duration, attempt, false)

		if attempt < webhook.Retries {
			// Retry with exponential backoff
			backoff := time.Duration(attempt*attempt) * time.Second
			time.Sleep(backoff)
			go s.deliverWebhook(webhook, event, data, attempt+1)
		} else {
			// Mark webhook as failed after max retries
			if _, err := s.DB.Exec("UPDATE webhooks SET status = 'failed', failure_count = failure_count + 1 WHERE id = ?", webhook.ID); err != nil {
				log.Printf("Warning: failed to update webhook status: %v", err)
			}
		}
		return
	}
	defer resp.Body.Close()

	// Read response
	responseBody, _ := io.ReadAll(resp.Body)
	responseStr := string(responseBody)
	if len(responseStr) > 1000 {
		responseStr = responseStr[:1000] + "..."
	}

	success := resp.StatusCode >= 200 && resp.StatusCode < 300

	// Record delivery
	headersJSON, _ := json.Marshal(req.Header)
	s.recordDelivery(webhook.ID, event, webhook.URL, string(payloadBytes),
		string(headersJSON), resp.StatusCode, responseStr, "", duration, attempt, success)

	if success {
		// Update webhook success stats
		if _, err := s.DB.Exec(`
			UPDATE webhooks 
			SET last_fired = datetime('now'), last_status = ?, failure_count = 0
			WHERE id = ?
		`, resp.StatusCode, webhook.ID); err != nil {
			log.Printf("Warning: failed to update webhook stats: %v", err)
		}
	} else {
		// Retry if needed
		if attempt < webhook.Retries {
			backoff := time.Duration(attempt*attempt) * time.Second
			time.Sleep(backoff)
			go s.deliverWebhook(webhook, event, data, attempt+1)
		} else {
			// Mark as failed
			if _, err := s.DB.Exec(`
				UPDATE webhooks 
				SET last_status = ?, failure_count = failure_count + 1
				WHERE id = ?
			`, resp.StatusCode, webhook.ID); err != nil {
				log.Printf("Warning: failed to update webhook failure status: %v", err)
			}
		}
	}
}

func (s *WebhookService) recordDelivery(webhookID int, event models.WebhookEvent, url, payload, headers string, statusCode int, response, errorMsg string, duration, attempt int, success bool) {
	if _, err := s.DB.Exec(`
		INSERT INTO webhook_deliveries (webhook_id, event, url, payload, headers, status_code, 
		                               response, error, duration_ms, attempt, success, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
	`, webhookID, event, url, payload, headers, statusCode, response, errorMsg, duration, attempt, success); err != nil {
		log.Printf("Warning: failed to record webhook delivery: %v", err)
	}
}

func (s *WebhookService) generateSignature(secret string, payload []byte) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

func (s *WebhookService) isValidEvent(event models.WebhookEvent) bool {
	validEvents := []models.WebhookEvent{
		models.WebhookEventNewShow,
		models.WebhookEventDownloadComplete,
		models.WebhookEventDownloadFailed,
		models.WebhookEventCatalogRefresh,
		models.WebhookEventMonitorAlert,
		models.WebhookEventSystemAlert,
	}

	for _, validEvent := range validEvents {
		if event == validEvent {
			return true
		}
	}
	return false
}

func (s *WebhookService) TestWebhook(webhookID int, req *models.WebhookTestRequest) (*models.WebhookTestResponse, error) {
	// Get webhook details
	var webhook models.Webhook
	var eventsJSON, headersJSON string
	err := s.DB.QueryRow(`
		SELECT id, name, url, events, secret, headers, timeout, retries
		FROM webhooks WHERE id = ?
	`, webhookID).Scan(&webhook.ID, &webhook.Name, &webhook.URL, &eventsJSON,
		&webhook.Secret, &headersJSON, &webhook.Timeout, &webhook.Retries)

	if err == sql.ErrNoRows {
		return &models.WebhookTestResponse{
			Success: false,
			Error:   "Webhook not found",
		}, nil
	}
	if err != nil {
		return nil, err
	}

	webhook.Headers = headersJSON

	// Generate test data based on event type
	var testData interface{}
	if req.SampleData {
		testData = s.generateSampleData(req.Event)
	} else {
		testData = map[string]interface{}{
			"test":    true,
			"message": "This is a test webhook delivery",
		}
	}

	// Deliver webhook synchronously for testing
	startTime := time.Now()

	// Create payload
	payload := models.WebhookPayload{
		Event:     req.Event,
		Timestamp: time.Now(),
		Source:    "nugs-api/v1.0.0-test",
		Data:      testData,
	}

	payloadBytes, _ := json.Marshal(payload)
	if webhook.Secret != "" {
		signature := s.generateSignature(webhook.Secret, payloadBytes)
		payload.Signature = signature
		payloadBytes, _ = json.Marshal(payload)
	}

	// Prepare request
	httpReq, err := http.NewRequest("POST", webhook.URL, bytes.NewReader(payloadBytes))
	if err != nil {
		return &models.WebhookTestResponse{
			Success: false,
			Error:   "Failed to create request: " + err.Error(),
		}, nil
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "nugs-api-webhook/1.0-test")
	httpReq.Header.Set("X-Webhook-Event", string(req.Event))
	httpReq.Header.Set("X-Webhook-Test", "true")

	if webhook.Secret != "" && payload.Signature != "" {
		httpReq.Header.Set("X-Hub-Signature-256", "sha256="+payload.Signature)
	}

	// Custom headers
	if webhook.Headers != "" {
		var customHeaders map[string]string
		if json.Unmarshal([]byte(webhook.Headers), &customHeaders) == nil {
			for key, value := range customHeaders {
				httpReq.Header.Set(key, value)
			}
		}
	}

	// Make request
	client := &http.Client{
		Timeout: time.Duration(webhook.Timeout) * time.Second,
	}

	resp, err := client.Do(httpReq)
	duration := int(time.Since(startTime).Milliseconds())

	if err != nil {
		// Record failed test delivery
		headersJSON, _ := json.Marshal(httpReq.Header)
		s.recordDelivery(webhook.ID, req.Event, webhook.URL, string(payloadBytes),
			string(headersJSON), 0, "", "Test delivery failed: "+err.Error(), duration, 1, false)

		return &models.WebhookTestResponse{
			Success:  false,
			Error:    err.Error(),
			Duration: duration,
		}, nil
	}
	defer resp.Body.Close()

	// Read response
	responseBody, _ := io.ReadAll(resp.Body)
	responseStr := string(responseBody)
	success := resp.StatusCode >= 200 && resp.StatusCode < 300

	// Record test delivery
	headersBytes, _ := json.Marshal(httpReq.Header)
	result, _ := s.DB.Exec(`
		INSERT INTO webhook_deliveries (webhook_id, event, url, payload, headers, status_code, 
		                               response, error, duration_ms, attempt, success, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'Test delivery', ?, 1, ?, datetime('now'))
	`, webhook.ID, req.Event, webhook.URL, string(payloadBytes), string(headersBytes),
		resp.StatusCode, responseStr, duration, success)

	deliveryID, _ := result.LastInsertId()

	return &models.WebhookTestResponse{
		Success:    success,
		StatusCode: resp.StatusCode,
		Response:   responseStr,
		Duration:   duration,
		DeliveryID: int(deliveryID),
	}, nil
}

func (s *WebhookService) generateSampleData(event models.WebhookEvent) interface{} {
	switch event {
	case models.WebhookEventNewShow:
		return models.NewShowPayload{
			Artist: struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			}{ID: 1, Name: "Sample Artist"},
			Show: struct {
				ID              int    `json:"id"`
				ContainerID     int    `json:"container_id"`
				Title           string `json:"title"`
				VenueName       string `json:"venue_name"`
				VenueCity       string `json:"venue_city"`
				VenueState      string `json:"venue_state"`
				PerformanceDate string `json:"performance_date"`
				PageURL         string `json:"page_url,omitempty"`
			}{
				ID:              12345,
				ContainerID:     67890,
				Title:           "Sample Concert - Sample Venue - 2024-01-15",
				VenueName:       "Sample Venue",
				VenueCity:       "Sample City",
				VenueState:      "CA",
				PerformanceDate: "2024-01-15",
			},
			MonitorID: 1,
		}

	case models.WebhookEventDownloadComplete:
		return models.DownloadCompletePayload{
			Download: struct {
				ID          int     `json:"id"`
				ShowID      int     `json:"show_id"`
				ContainerID int     `json:"container_id"`
				ArtistName  string  `json:"artist_name"`
				ShowTitle   string  `json:"show_title"`
				Format      string  `json:"format"`
				Quality     string  `json:"quality"`
				FileSizeGB  float64 `json:"file_size_gb"`
				Duration    string  `json:"duration"`
			}{
				ID:          123,
				ShowID:      12345,
				ContainerID: 67890,
				ArtistName:  "Sample Artist",
				ShowTitle:   "Sample Concert",
				Format:      "flac",
				Quality:     "lossless",
				FileSizeGB:  2.5,
				Duration:    "15m32s",
			},
		}

	default:
		return map[string]interface{}{
			"sample": true,
			"event":  event,
			"data":   "Sample webhook data",
		}
	}
}

func (s *WebhookService) GetWebhookStats() (*models.WebhookStats, error) {
	stats := &models.WebhookStats{
		EventBreakdown: make(map[string]int64),
	}

	// Get webhook counts
	err := s.DB.QueryRow(`
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'active' THEN 1 END) as active,
			COUNT(CASE WHEN status = 'disabled' THEN 1 END) as disabled,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed
		FROM webhooks
	`).Scan(&stats.TotalWebhooks, &stats.ActiveWebhooks, &stats.DisabledWebhooks, &stats.FailedWebhooks)

	if err != nil {
		return nil, err
	}

	// Get delivery stats
	if err := s.DB.QueryRow(`
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN success = 1 THEN 1 END) as successful,
			COUNT(CASE WHEN success = 0 THEN 1 END) as failed,
			COALESCE(AVG(duration_ms), 0) as avg_duration
		FROM webhook_deliveries
	`).Scan(&stats.TotalDeliveries, &stats.SuccessfulDeliveries, &stats.FailedDeliveries, &stats.AverageResponseTime); err != nil {
		log.Printf("Warning: failed to get webhook delivery stats: %v", err)
		stats.TotalDeliveries = 0
		stats.SuccessfulDeliveries = 0
		stats.FailedDeliveries = 0
		stats.AverageResponseTime = 0
	}

	// Calculate success rate
	if stats.TotalDeliveries > 0 {
		stats.DeliverySuccessRate = float64(stats.SuccessfulDeliveries) / float64(stats.TotalDeliveries) * 100
	}

	// Get event breakdown
	rows, err := s.DB.Query(`
		SELECT event, COUNT(*) 
		FROM webhook_deliveries 
		GROUP BY event
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var event string
			var count int64
			if rows.Scan(&event, &count) == nil {
				stats.EventBreakdown[event] = count
			}
		}
	}

	return stats, nil
}
