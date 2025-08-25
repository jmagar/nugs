package models

import (
	"database/sql"
	"time"
)

type DownloadStatus string

const (
	DownloadStatusPending    DownloadStatus = "pending"
	DownloadStatusInProgress DownloadStatus = "in_progress"
	DownloadStatusCompleted  DownloadStatus = "completed"
	DownloadStatusFailed     DownloadStatus = "failed"
	DownloadStatusCancelled  DownloadStatus = "cancelled"
)

type DownloadFormat string

const (
	DownloadFormatMP3  DownloadFormat = "mp3"
	DownloadFormatFLAC DownloadFormat = "flac"
	DownloadFormatALAC DownloadFormat = "alac"
)

type DownloadQuality string

const (
	DownloadQualityStandard DownloadQuality = "standard"
	DownloadQualityHD       DownloadQuality = "hd"
	DownloadQualityLossless DownloadQuality = "lossless"
)

type Download struct {
	ID           int             `json:"id" db:"id"`
	ShowID       int             `json:"show_id" db:"show_id"`
	ContainerID  int             `json:"container_id" db:"container_id"`
	ArtistName   string          `json:"artist_name" db:"artist_name"`
	FilePath     sql.NullString  `json:"file_path,omitempty" db:"file_path"`
	FileSize     int64           `json:"file_size" db:"file_size"`
	Quality      DownloadQuality `json:"quality" db:"quality"`
	Format       DownloadFormat  `json:"format" db:"format"`
	Status       DownloadStatus  `json:"status" db:"status"`
	Progress     int             `json:"progress"` // 0-100, not stored in DB
	ErrorMessage string          `json:"error_message,omitempty"`
	DownloadedAt *time.Time      `json:"downloaded_at,omitempty" db:"downloaded_at"`
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at,omitempty"`

	// Show details (populated via JOIN)
	ShowTitle       string `json:"show_title,omitempty"`
	VenueName       string `json:"venue_name,omitempty"`
	VenueCity       string `json:"venue_city,omitempty"`
	VenueState      string `json:"venue_state,omitempty"`
	PerformanceDate string `json:"performance_date,omitempty"`
}

type DownloadQueue struct {
	ID         int       `json:"id" db:"id"`
	DownloadID int       `json:"download_id" db:"download_id"`
	Position   int       `json:"position" db:"position"`
	Priority   int       `json:"priority" db:"priority"` // Higher number = higher priority
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

type DownloadRequest struct {
	ShowID   int             `json:"show_id" binding:"required"`
	Format   DownloadFormat  `json:"format" binding:"required"`
	Quality  DownloadQuality `json:"quality"`
	Priority int             `json:"priority"` // 1-10, default 5
}

type DownloadResponse struct {
	Success    bool   `json:"success"`
	DownloadID int    `json:"download_id,omitempty"`
	JobID      string `json:"job_id,omitempty"`
	Status     string `json:"status"`
	Message    string `json:"message"`
	Error      string `json:"error,omitempty"`
}

type DownloadStats struct {
	TotalDownloads      int64            `json:"total_downloads"`
	CompletedDownloads  int64            `json:"completed_downloads"`
	FailedDownloads     int64            `json:"failed_downloads"`
	PendingDownloads    int64            `json:"pending_downloads"`
	InProgressDownloads int64            `json:"in_progress_downloads"`
	TotalSizeGB         float64          `json:"total_size_gb"`
	FormatBreakdown     map[string]int64 `json:"format_breakdown"`
	QualityBreakdown    map[string]int64 `json:"quality_breakdown"`
	QueueLength         int64            `json:"queue_length"`
	ActiveDownloads     int64            `json:"active_downloads"`
	AverageSpeedMbps    float64          `json:"average_speed_mbps"`
}
