package models

import (
	"database/sql"
	"time"
)

type MonitorStatus string

const (
	MonitorStatusActive   MonitorStatus = "active"
	MonitorStatusPaused   MonitorStatus = "paused"
	MonitorStatusDisabled MonitorStatus = "disabled"
)

type AlertType string

const (
	AlertTypeNewShow     AlertType = "new_show"
	AlertTypeShowUpdate  AlertType = "show_update"
	AlertTypeMissingShow AlertType = "missing_show"
)

type ArtistMonitor struct {
	ID                int           `json:"id" db:"id"`
	ArtistID          int           `json:"artist_id" db:"artist_id"`
	ArtistName        string        `json:"artist_name" db:"artist_name"`
	Status            MonitorStatus `json:"status" db:"status"`
	CheckInterval     int           `json:"check_interval" db:"check_interval"` // minutes
	LastChecked       *time.Time    `json:"last_checked,omitempty" db:"last_checked"`
	LastNewShow       *time.Time    `json:"last_new_show,omitempty" db:"last_new_show"`
	TotalShows        int           `json:"total_shows" db:"total_shows"`
	NewShowsFound     int           `json:"new_shows_found" db:"new_shows_found"`
	NotifyNewShows    bool          `json:"notify_new_shows" db:"notify_new_shows"`
	NotifyShowUpdates bool          `json:"notify_show_updates" db:"notify_show_updates"`
	CreatedAt         time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at" db:"updated_at"`
}

type MonitorAlert struct {
	ID           int           `json:"id" db:"id"`
	MonitorID    int           `json:"monitor_id" db:"monitor_id"`
	ArtistID     int           `json:"artist_id" db:"artist_id"`
	ShowID       sql.NullInt64 `json:"show_id,omitempty" db:"show_id"`
	AlertType    AlertType     `json:"alert_type" db:"alert_type"`
	Message      string        `json:"message" db:"message"`
	Details      string        `json:"details,omitempty" db:"details"`
	Acknowledged bool          `json:"acknowledged" db:"acknowledged"`
	CreatedAt    time.Time     `json:"created_at" db:"created_at"`

	// Related data (populated via JOIN)
	ArtistName string `json:"artist_name,omitempty"`
	ShowTitle  string `json:"show_title,omitempty"`
}

type MonitorStats struct {
	TotalMonitors        int64      `json:"total_monitors"`
	ActiveMonitors       int64      `json:"active_monitors"`
	PausedMonitors       int64      `json:"paused_monitors"`
	TotalAlertsToday     int64      `json:"total_alerts_today"`
	UnacknowledgedAlerts int64      `json:"unacknowledged_alerts"`
	AverageCheckTime     float64    `json:"average_check_time_seconds"`
	LastCheckTime        *time.Time `json:"last_check_time,omitempty"`
}

type MonitorRequest struct {
	ArtistID          int  `json:"artist_id" binding:"required"`
	CheckInterval     int  `json:"check_interval"` // minutes, default 60
	NotifyNewShows    bool `json:"notify_new_shows"`
	NotifyShowUpdates bool `json:"notify_show_updates"`
}

type MonitorUpdateRequest struct {
	Status            *MonitorStatus `json:"status,omitempty"`
	CheckInterval     *int           `json:"check_interval,omitempty"`
	NotifyNewShows    *bool          `json:"notify_new_shows,omitempty"`
	NotifyShowUpdates *bool          `json:"notify_show_updates,omitempty"`
}

type MonitorResponse struct {
	Success   bool   `json:"success"`
	MonitorID int    `json:"monitor_id,omitempty"`
	Message   string `json:"message"`
	Error     string `json:"error,omitempty"`
}

type CheckResult struct {
	ArtistID      int    `json:"artist_id"`
	ArtistName    string `json:"artist_name"`
	PreviousCount int    `json:"previous_count"`
	CurrentCount  int    `json:"current_count"`
	NewShows      int    `json:"new_shows"`
	CheckDuration string `json:"check_duration"`
	Success       bool   `json:"success"`
	Error         string `json:"error,omitempty"`
}

type BulkMonitorRequest struct {
	ArtistIDs         []int `json:"artist_ids" binding:"required"`
	CheckInterval     int   `json:"check_interval"` // minutes, default 60
	NotifyNewShows    bool  `json:"notify_new_shows"`
	NotifyShowUpdates bool  `json:"notify_show_updates"`
}

type BulkMonitorResponse struct {
	Success        bool     `json:"success"`
	ProcessedCount int      `json:"processed_count"`
	SuccessCount   int      `json:"success_count"`
	FailedCount    int      `json:"failed_count"`
	Errors         []string `json:"errors,omitempty"`
	Message        string   `json:"message"`
}
