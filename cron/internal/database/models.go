package database

import (
	"time"
)

// User represents a system user
type User struct {
	ID        int       `json:"id" db:"id"`
	Username  string    `json:"username" db:"username"`
	Email     string    `json:"email" db:"email"`
	Password  string    `json:"-" db:"password_hash"` // Never serialize password
	Role      string    `json:"role" db:"role"`
	Active    bool      `json:"active" db:"active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Artist represents an artist in our system
type Artist struct {
	ID        int        `json:"id" db:"id"`
	NugsID    int        `json:"nugs_id" db:"nugs_id"`
	Name      string     `json:"name" db:"name"`
	Slug      string     `json:"slug" db:"slug"`
	Monitored bool       `json:"monitored" db:"monitored"`
	ShowCount int        `json:"show_count" db:"show_count"`
	LastScan  *time.Time `json:"last_scan,omitempty" db:"last_scan"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
}

// Show represents a show/concert
type Show struct {
	ID                       int       `json:"id" db:"id"`
	ContainerID              int       `json:"container_id" db:"container_id"`
	ArtistID                 int       `json:"artist_id" db:"artist_id"`
	ArtistName               string    `json:"artist_name" db:"artist_name"`
	VenueName                string    `json:"venue_name" db:"venue_name"`
	VenueCity                string    `json:"venue_city" db:"venue_city"`
	VenueState               string    `json:"venue_state" db:"venue_state"`
	PerformanceDate          string    `json:"performance_date" db:"performance_date"`
	PerformanceDateShort     string    `json:"performance_date_short" db:"performance_date_short"`
	PerformanceDateFormatted string    `json:"performance_date_formatted" db:"performance_date_formatted"`
	ContainerInfo            string    `json:"container_info" db:"container_info"`
	AvailabilityType         int       `json:"availability_type" db:"availability_type"`
	AvailabilityTypeStr      string    `json:"availability_type_str" db:"availability_type_str"`
	ActiveState              string    `json:"active_state" db:"active_state"`
	CreatedAt                time.Time `json:"created_at" db:"created_at"`
	UpdatedAt                time.Time `json:"updated_at" db:"updated_at"`
}

// Download represents a downloaded show
type Download struct {
	ID           int       `json:"id" db:"id"`
	ShowID       int       `json:"show_id" db:"show_id"`
	ContainerID  int       `json:"container_id" db:"container_id"`
	ArtistName   string    `json:"artist_name" db:"artist_name"`
	FilePath     string    `json:"file_path" db:"file_path"`
	FileSize     int64     `json:"file_size" db:"file_size"`
	Quality      string    `json:"quality" db:"quality"`
	Format       string    `json:"format" db:"format"`
	Status       string    `json:"status" db:"status"` // downloaded, failed, pending
	DownloadedAt time.Time `json:"downloaded_at" db:"downloaded_at"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// APILog represents API request logging
type APILog struct {
	ID           int       `json:"id" db:"id"`
	UserID       *int      `json:"user_id,omitempty" db:"user_id"`
	Method       string    `json:"method" db:"method"`
	Path         string    `json:"path" db:"path"`
	StatusCode   int       `json:"status_code" db:"status_code"`
	ResponseTime int64     `json:"response_time_ms" db:"response_time_ms"`
	IPAddress    string    `json:"ip_address" db:"ip_address"`
	UserAgent    string    `json:"user_agent" db:"user_agent"`
	RequestID    string    `json:"request_id" db:"request_id"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// PaginationParams represents pagination parameters
type PaginationParams struct {
	Page     int `json:"page" form:"page"`
	PageSize int `json:"page_size" form:"page_size"`
	Offset   int `json:"offset"`
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	Total      int64       `json:"total"`
	TotalPages int         `json:"total_pages"`
	HasNext    bool        `json:"has_next"`
	HasPrev    bool        `json:"has_prev"`
}

// ArtistStats represents statistics for an artist
type ArtistStats struct {
	Artist               Artist     `json:"artist"`
	TotalShows           int        `json:"total_shows"`
	DownloadedShows      int        `json:"downloaded_shows"`
	AvailableShows       int        `json:"available_shows"`
	MissingShows         int        `json:"missing_shows"`
	LastDownload         *time.Time `json:"last_download,omitempty"`
	StorageUsed          int64      `json:"storage_used_bytes"`
	StorageUsedFormatted string     `json:"storage_used_formatted"`
}

// SystemStats represents overall system statistics
type SystemStats struct {
	TotalArtists          int        `json:"total_artists"`
	MonitoredArtists      int        `json:"monitored_artists"`
	TotalShows            int        `json:"total_shows"`
	TotalDownloads        int        `json:"total_downloads"`
	TotalStorage          int64      `json:"total_storage_bytes"`
	TotalStorageFormatted string     `json:"total_storage_formatted"`
	LastCatalogUpdate     *time.Time `json:"last_catalog_update,omitempty"`
	ActiveUsers           int        `json:"active_users"`
	APIRequestsToday      int        `json:"api_requests_today"`
}
