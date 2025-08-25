package models

import (
	"time"
)

type WebhookStatus string

const (
	WebhookStatusActive   WebhookStatus = "active"
	WebhookStatusDisabled WebhookStatus = "disabled"
	WebhookStatusFailed   WebhookStatus = "failed"
)

type WebhookEvent string

const (
	WebhookEventNewShow          WebhookEvent = "new_show"
	WebhookEventDownloadComplete WebhookEvent = "download_complete"
	WebhookEventDownloadFailed   WebhookEvent = "download_failed"
	WebhookEventCatalogRefresh   WebhookEvent = "catalog_refresh"
	WebhookEventMonitorAlert     WebhookEvent = "monitor_alert"
	WebhookEventSystemAlert      WebhookEvent = "system_alert"
)

type Webhook struct {
	ID           int            `json:"id" db:"id"`
	Name         string         `json:"name" db:"name"`
	URL          string         `json:"url" db:"url"`
	Events       []WebhookEvent `json:"events" db:"events"` // Stored as JSON string
	Status       WebhookStatus  `json:"status" db:"status"`
	Secret       string         `json:"secret,omitempty" db:"secret"`
	Headers      string         `json:"headers,omitempty" db:"headers"` // JSON string
	Timeout      int            `json:"timeout" db:"timeout"`           // seconds
	Retries      int            `json:"retries" db:"retries"`
	LastFired    *time.Time     `json:"last_fired,omitempty" db:"last_fired"`
	LastStatus   int            `json:"last_status" db:"last_status"`
	FailureCount int            `json:"failure_count" db:"failure_count"`
	CreatedAt    time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at" db:"updated_at"`

	// Statistics
	TotalFired   int64   `json:"total_fired"`
	SuccessCount int64   `json:"success_count"`
	FailureRate  float64 `json:"failure_rate"`
}

type WebhookDelivery struct {
	ID         int          `json:"id" db:"id"`
	WebhookID  int          `json:"webhook_id" db:"webhook_id"`
	Event      WebhookEvent `json:"event" db:"event"`
	URL        string       `json:"url" db:"url"`
	Payload    string       `json:"payload" db:"payload"`           // JSON string
	Headers    string       `json:"headers,omitempty" db:"headers"` // JSON string
	StatusCode int          `json:"status_code" db:"status_code"`
	Response   string       `json:"response,omitempty" db:"response"`
	Error      string       `json:"error,omitempty" db:"error"`
	Duration   int          `json:"duration_ms" db:"duration_ms"`
	Attempt    int          `json:"attempt" db:"attempt"`
	Success    bool         `json:"success" db:"success"`
	CreatedAt  time.Time    `json:"created_at" db:"created_at"`

	// Related data
	WebhookName string `json:"webhook_name,omitempty"`
}

type WebhookRequest struct {
	Name    string            `json:"name" binding:"required"`
	URL     string            `json:"url" binding:"required,url"`
	Events  []WebhookEvent    `json:"events" binding:"required"`
	Secret  string            `json:"secret,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Timeout int               `json:"timeout"` // seconds, default 10
	Retries int               `json:"retries"` // default 3
}

type WebhookUpdateRequest struct {
	Name    *string            `json:"name,omitempty"`
	URL     *string            `json:"url,omitempty"`
	Events  *[]WebhookEvent    `json:"events,omitempty"`
	Status  *WebhookStatus     `json:"status,omitempty"`
	Secret  *string            `json:"secret,omitempty"`
	Headers *map[string]string `json:"headers,omitempty"`
	Timeout *int               `json:"timeout,omitempty"`
	Retries *int               `json:"retries,omitempty"`
}

type WebhookResponse struct {
	Success   bool   `json:"success"`
	WebhookID int    `json:"webhook_id,omitempty"`
	Message   string `json:"message"`
	Error     string `json:"error,omitempty"`
}

type WebhookPayload struct {
	Event     WebhookEvent `json:"event"`
	Timestamp time.Time    `json:"timestamp"`
	Source    string       `json:"source"` // API name/version
	Data      interface{}  `json:"data"`
	Signature string       `json:"signature,omitempty"` // HMAC signature if secret provided
}

// Event-specific payload data structures
type NewShowPayload struct {
	Artist struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"artist"`
	Show struct {
		ID              int    `json:"id"`
		ContainerID     int    `json:"container_id"`
		Title           string `json:"title"`
		VenueName       string `json:"venue_name"`
		VenueCity       string `json:"venue_city"`
		VenueState      string `json:"venue_state"`
		PerformanceDate string `json:"performance_date"`
		PageURL         string `json:"page_url,omitempty"`
	} `json:"show"`
	MonitorID int `json:"monitor_id,omitempty"`
}

type DownloadCompletePayload struct {
	Download struct {
		ID          int     `json:"id"`
		ShowID      int     `json:"show_id"`
		ContainerID int     `json:"container_id"`
		ArtistName  string  `json:"artist_name"`
		ShowTitle   string  `json:"show_title"`
		Format      string  `json:"format"`
		Quality     string  `json:"quality"`
		FileSizeGB  float64 `json:"file_size_gb"`
		Duration    string  `json:"duration"`
	} `json:"download"`
}

type DownloadFailedPayload struct {
	Download struct {
		ID          int    `json:"id"`
		ShowID      int    `json:"show_id"`
		ContainerID int    `json:"container_id"`
		ArtistName  string `json:"artist_name"`
		ShowTitle   string `json:"show_title"`
		Format      string `json:"format"`
		Quality     string `json:"quality"`
	} `json:"download"`
	Error   string `json:"error"`
	Attempt int    `json:"attempt"`
}

type CatalogRefreshPayload struct {
	JobID           string `json:"job_id"`
	Status          string `json:"status"` // completed, failed
	Duration        string `json:"duration"`
	ShowsImported   int64  `json:"shows_imported"`
	ArtistsImported int64  `json:"artists_imported"`
	Error           string `json:"error,omitempty"`
}

type MonitorAlertPayload struct {
	Alert struct {
		ID      int    `json:"id"`
		Type    string `json:"type"`
		Message string `json:"message"`
		Details string `json:"details,omitempty"`
	} `json:"alert"`
	Artist struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"artist"`
	Monitor struct {
		ID int `json:"id"`
	} `json:"monitor"`
}

type SystemAlertPayload struct {
	Alert struct {
		Type      string `json:"type"`
		Severity  string `json:"severity"` // info, warning, error, critical
		Message   string `json:"message"`
		Details   string `json:"details,omitempty"`
		Component string `json:"component,omitempty"`
	} `json:"alert"`
	System struct {
		HealthScore int    `json:"health_score"`
		Status      string `json:"status"`
		Version     string `json:"version,omitempty"`
	} `json:"system"`
}

type WebhookStats struct {
	TotalWebhooks        int64            `json:"total_webhooks"`
	ActiveWebhooks       int64            `json:"active_webhooks"`
	DisabledWebhooks     int64            `json:"disabled_webhooks"`
	FailedWebhooks       int64            `json:"failed_webhooks"`
	TotalDeliveries      int64            `json:"total_deliveries"`
	SuccessfulDeliveries int64            `json:"successful_deliveries"`
	FailedDeliveries     int64            `json:"failed_deliveries"`
	AverageResponseTime  float64          `json:"average_response_time_ms"`
	DeliverySuccessRate  float64          `json:"delivery_success_rate"`
	EventBreakdown       map[string]int64 `json:"event_breakdown"`
}

type WebhookTestRequest struct {
	Event      WebhookEvent `json:"event" binding:"required"`
	SampleData bool         `json:"sample_data"` // Use sample data instead of real data
}

type WebhookTestResponse struct {
	Success    bool   `json:"success"`
	StatusCode int    `json:"status_code"`
	Response   string `json:"response,omitempty"`
	Error      string `json:"error,omitempty"`
	Duration   int    `json:"duration_ms"`
	DeliveryID int    `json:"delivery_id,omitempty"`
}
