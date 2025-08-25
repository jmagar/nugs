package models

import "time"

type AnalyticsTimeframe string

const (
	TimeframeDay   AnalyticsTimeframe = "day"
	TimeframeWeek  AnalyticsTimeframe = "week"
	TimeframeMonth AnalyticsTimeframe = "month"
	TimeframeYear  AnalyticsTimeframe = "year"
	TimeframeAll   AnalyticsTimeframe = "all"
)

type CollectionStats struct {
	TotalArtists          int64   `json:"total_artists"`
	TotalShows            int64   `json:"total_shows"`
	TotalDownloads        int64   `json:"total_downloads"`
	TotalSizeGB           float64 `json:"total_size_gb"`
	AverageShowsPerArtist float64 `json:"average_shows_per_artist"`
	RecentActivity        struct {
		NewShowsToday      int64 `json:"new_shows_today"`
		NewShowsThisWeek   int64 `json:"new_shows_this_week"`
		NewShowsThisMonth  int64 `json:"new_shows_this_month"`
		DownloadsToday     int64 `json:"downloads_today"`
		DownloadsThisWeek  int64 `json:"downloads_this_week"`
		DownloadsThisMonth int64 `json:"downloads_this_month"`
	} `json:"recent_activity"`
}

type ArtistAnalytics struct {
	ArtistID                int     `json:"artist_id"`
	ArtistName              string  `json:"artist_name"`
	TotalShows              int64   `json:"total_shows"`
	TotalDownloads          int64   `json:"total_downloads"`
	TotalSizeGB             float64 `json:"total_size_gb"`
	AverageShowSizeGB       float64 `json:"average_show_size_gb"`
	PopularityScore         float64 `json:"popularity_score"` // Based on downloads/shows ratio
	FirstShowDate           *string `json:"first_show_date,omitempty"`
	LastShowDate            *string `json:"last_show_date,omitempty"`
	MostDownloadedShow      *string `json:"most_downloaded_show,omitempty"`
	PreferredFormat         string  `json:"preferred_format,omitempty"`
	PreferredQuality        string  `json:"preferred_quality,omitempty"`
	ShowGrowthLastMonth     int64   `json:"show_growth_last_month"`
	DownloadGrowthLastMonth int64   `json:"download_growth_last_month"`
}

type DownloadAnalytics struct {
	TotalDownloads      int64            `json:"total_downloads"`
	CompletedDownloads  int64            `json:"completed_downloads"`
	FailedDownloads     int64            `json:"failed_downloads"`
	PendingDownloads    int64            `json:"pending_downloads"`
	SuccessRate         float64          `json:"success_rate"`
	TotalSizeGB         float64          `json:"total_size_gb"`
	AverageSizeGB       float64          `json:"average_size_gb"`
	FormatBreakdown     map[string]int64 `json:"format_breakdown"`
	QualityBreakdown    map[string]int64 `json:"quality_breakdown"`
	PopularVenues       []VenueStats     `json:"popular_venues"`
	PopularFormats      []FormatStats    `json:"popular_formats"`
	DownloadTrends      []TrendPoint     `json:"download_trends"`
	AverageDownloadTime float64          `json:"average_download_time_minutes"`
	PeakDownloadHours   []HourStats      `json:"peak_download_hours"`
}

type VenueStats struct {
	VenueName     string `json:"venue_name"`
	VenueCity     string `json:"venue_city"`
	VenueState    string `json:"venue_state"`
	ShowCount     int64  `json:"show_count"`
	DownloadCount int64  `json:"download_count"`
}

type FormatStats struct {
	Format        string  `json:"format"`
	Count         int64   `json:"count"`
	Percentage    float64 `json:"percentage"`
	TotalSizeGB   float64 `json:"total_size_gb"`
	AverageSizeGB float64 `json:"average_size_gb"`
}

type TrendPoint struct {
	Date   string  `json:"date"`
	Count  int64   `json:"count"`
	SizeGB float64 `json:"size_gb,omitempty"`
}

type HourStats struct {
	Hour  int   `json:"hour"`
	Count int64 `json:"count"`
}

type SystemMetrics struct {
	DatabaseSize       float64 `json:"database_size_mb"`
	TotalFiles         int64   `json:"total_files"`
	TotalStorage       float64 `json:"total_storage_gb"`
	AvailableStorage   float64 `json:"available_storage_gb"`
	ActiveDownloads    int     `json:"active_downloads"`
	ActiveMonitors     int64   `json:"active_monitors"`
	SystemUptime       string  `json:"system_uptime"`
	LastCatalogRefresh *string `json:"last_catalog_refresh,omitempty"`
	APIRequests        struct {
		Today     int64 `json:"today"`
		ThisWeek  int64 `json:"this_week"`
		ThisMonth int64 `json:"this_month"`
	} `json:"api_requests"`
	JobStats struct {
		TotalJobs     int64 `json:"total_jobs"`
		CompletedJobs int64 `json:"completed_jobs"`
		FailedJobs    int64 `json:"failed_jobs"`
		RunningJobs   int64 `json:"running_jobs"`
	} `json:"job_stats"`
}

type PerformanceMetrics struct {
	AverageResponseTime  float64 `json:"average_response_time_ms"`
	DatabaseResponseTime float64 `json:"database_response_time_ms"`
	CacheHitRate         float64 `json:"cache_hit_rate"`
	ErrorRate            float64 `json:"error_rate"`
	ThroughputPerSecond  float64 `json:"throughput_per_second"`
	MemoryUsageMB        float64 `json:"memory_usage_mb"`
	CPUUsagePercent      float64 `json:"cpu_usage_percent"`
}

type TimeSeriesData struct {
	Timestamps []string  `json:"timestamps"`
	Values     []float64 `json:"values"`
	Label      string    `json:"label"`
	Unit       string    `json:"unit"`
}

type AnalyticsReport struct {
	ReportID    string                 `json:"report_id"`
	ReportType  string                 `json:"report_type"`
	Timeframe   AnalyticsTimeframe     `json:"timeframe"`
	GeneratedAt time.Time              `json:"generated_at"`
	GeneratedBy string                 `json:"generated_by,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`

	// Report data based on type
	CollectionStats   *CollectionStats    `json:"collection_stats,omitempty"`
	ArtistAnalytics   []ArtistAnalytics   `json:"artist_analytics,omitempty"`
	DownloadAnalytics *DownloadAnalytics  `json:"download_analytics,omitempty"`
	SystemMetrics     *SystemMetrics      `json:"system_metrics,omitempty"`
	Performance       *PerformanceMetrics `json:"performance,omitempty"`
	TimeSeries        []TimeSeriesData    `json:"time_series,omitempty"`

	Summary         string   `json:"summary"`
	Recommendations []string `json:"recommendations,omitempty"`
}

type AnalyticsQuery struct {
	ReportType        string                 `json:"report_type" binding:"required"` // collection, artists, downloads, system, performance
	Timeframe         AnalyticsTimeframe     `json:"timeframe"`
	StartDate         *string                `json:"start_date,omitempty"`
	EndDate           *string                `json:"end_date,omitempty"`
	ArtistIDs         []int                  `json:"artist_ids,omitempty"`
	Filters           map[string]interface{} `json:"filters,omitempty"`
	GroupBy           string                 `json:"group_by,omitempty"` // day, week, month, artist, format, venue
	Limit             int                    `json:"limit,omitempty"`
	IncludeTimeSeries bool                   `json:"include_time_series,omitempty"`
}

type TopListItem struct {
	ID             int     `json:"id,omitempty"`
	Name           string  `json:"name"`
	Value          float64 `json:"value"`
	SecondaryValue float64 `json:"secondary_value,omitempty"`
	Unit           string  `json:"unit,omitempty"`
	Percentage     float64 `json:"percentage,omitempty"`
}

type ComparisonData struct {
	Period        string  `json:"period"`
	Current       float64 `json:"current"`
	Previous      float64 `json:"previous"`
	Change        float64 `json:"change"`
	ChangePercent float64 `json:"change_percent"`
	Trend         string  `json:"trend"` // up, down, stable
}

type HealthScore struct {
	Overall         int            `json:"overall"` // 0-100
	Categories      map[string]int `json:"categories"`
	Issues          []string       `json:"issues,omitempty"`
	Recommendations []string       `json:"recommendations,omitempty"`
	LastUpdated     time.Time      `json:"last_updated"`
}
