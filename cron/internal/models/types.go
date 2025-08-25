package models

// Config holds configuration for Nugs.net credentials and download settings
type Config struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Format   int    `json:"format"`
	OutPath  string `json:"outPath"`
}

// MonitorConfig holds configuration for which artists to monitor
type MonitorConfig struct {
	Artists []Artist `json:"artists"`
}

// Artist represents an artist configuration for monitoring
type Artist struct {
	ID           int    `json:"id"`
	Artist       string `json:"artist"`
	Monitor      bool   `json:"monitor"`
	ArtistFolder string `json:"artist_folder"`
}

// ShowsData represents the complete tracking data structure
type ShowsData struct {
	LastCatalogUpdate   string                    `json:"last_catalog_update"`
	CatalogTotalShows   int                       `json:"catalog_total_shows"`
	CatalogTotalArtists int                       `json:"catalog_total_artists"`
	LastAnalysisTime    string                    `json:"last_analysis_time"`
	Artists             map[string]ArtistShowData `json:"artists"`
}

// ArtistShowData tracks shows for a specific artist
type ArtistShowData struct {
	ArtistID   int   `json:"artist_id"`
	Downloaded []int `json:"downloaded"`
	Available  []int `json:"available"`
	Missing    []int `json:"missing"`
}
