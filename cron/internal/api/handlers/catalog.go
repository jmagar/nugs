package handlers

import (
	"database/sql"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// CatalogHandler handles catalog-related endpoints
type CatalogHandler struct {
	DB *sql.DB
}

// Artist represents an artist in the catalog
type Artist struct {
	ID           int        `json:"id" db:"id"`
	NugsArtistID *int       `json:"nugs_artist_id,omitempty" db:"nugs_artist_id"`
	Name         string     `json:"name" db:"name"`
	Slug         string     `json:"slug" db:"slug"`
	ShowCount    int        `json:"show_count" db:"show_count"`
	IsActive     bool       `json:"is_active" db:"is_active"`
	LastUpdated  *time.Time `json:"last_updated,omitempty" db:"last_updated"`
	Genres       *string    `json:"genres,omitempty" db:"genres"`
	Description  *string    `json:"description,omitempty" db:"description"`
	ImageURL     *string    `json:"image_url,omitempty" db:"image_url"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
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

// PaginationParams represents pagination parameters
type PaginationParams struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
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

func NewCatalogHandler(db *sql.DB) *CatalogHandler {
	return &CatalogHandler{DB: db}
}

// validatePagination ensures pagination parameters are valid
func validatePagination(params *PaginationParams) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 20 // Default page size
	}
	if params.PageSize > 100 {
		params.PageSize = 100 // Max page size
	}
	params.Offset = (params.Page - 1) * params.PageSize
}

// createPaginatedResponse creates a standardized paginated response
func createPaginatedResponse(data interface{}, params PaginationParams, total int64) *PaginatedResponse {
	totalPages := int(math.Ceil(float64(total) / float64(params.PageSize)))

	return &PaginatedResponse{
		Data:       data,
		Page:       params.Page,
		PageSize:   params.PageSize,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    params.Page < totalPages,
		HasPrev:    params.Page > 1,
	}
}

// GetArtists returns a paginated list of artists
func (h *CatalogHandler) GetArtists(c *gin.Context) {
	// Parse pagination parameters
	params := PaginationParams{}
	if page := c.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil {
			params.Page = p
		}
	}
	if pageSize := c.Query("page_size"); pageSize != "" {
		if ps, err := strconv.Atoi(pageSize); err == nil {
			params.PageSize = ps
		}
	}
	validatePagination(&params)

	// Parse filters
	search := c.Query("search")
	// Note: monitored filter functionality requires joins with monitors table - not implemented yet

	// Build WHERE clause
	whereClause := "WHERE 1=1"
	args := []interface{}{}

	if search != "" {
		whereClause += " AND (name LIKE ? OR slug LIKE ?)"
		searchPattern := "%" + search + "%"
		args = append(args, searchPattern, searchPattern)
	}

	// Count total records
	countQuery := "SELECT COUNT(*) FROM artists " + whereClause
	var total int64
	err := h.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count artists"})
		return
	}

	// Get paginated results
	query := `
		SELECT id, nugs_artist_id, name, slug, show_count, is_active,
		       last_updated, genres, description, image_url, created_at, updated_at
		FROM artists ` + whereClause + `
		ORDER BY name ASC
		LIMIT ? OFFSET ?
	`

	args = append(args, params.PageSize, params.Offset)

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query artists"})
		return
	}
	defer rows.Close()

	var artists []Artist
	for rows.Next() {
		var artist Artist
		err := rows.Scan(
			&artist.ID, &artist.NugsArtistID, &artist.Name, &artist.Slug,
			&artist.ShowCount, &artist.IsActive, &artist.LastUpdated,
			&artist.Genres, &artist.Description, &artist.ImageURL,
			&artist.CreatedAt, &artist.UpdatedAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan artist"})
			return
		}
		artists = append(artists, artist)
	}

	response := createPaginatedResponse(artists, params, total)
	c.JSON(http.StatusOK, response)
}

// GetArtist returns a specific artist by ID or slug
func (h *CatalogHandler) GetArtist(c *gin.Context) {
	identifier := c.Param("id")

	query := `
		SELECT id, nugs_artist_id, name, slug, show_count, is_active,
		       last_updated, genres, description, image_url, created_at, updated_at
		FROM artists WHERE id = ? OR slug = ?
	`

	var artist Artist
	err := h.DB.QueryRow(query, identifier, identifier).Scan(
		&artist.ID, &artist.NugsArtistID, &artist.Name, &artist.Slug,
		&artist.ShowCount, &artist.IsActive, &artist.LastUpdated,
		&artist.Genres, &artist.Description, &artist.ImageURL,
		&artist.CreatedAt, &artist.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Artist not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get artist"})
		return
	}

	c.JSON(http.StatusOK, artist)
}

// GetArtistShows returns paginated shows for a specific artist
func (h *CatalogHandler) GetArtistShows(c *gin.Context) {
	artistID := c.Param("id")

	// Parse pagination parameters
	params := PaginationParams{}
	if page := c.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil {
			params.Page = p
		}
	}
	if pageSize := c.Query("page_size"); pageSize != "" {
		if ps, err := strconv.Atoi(pageSize); err == nil {
			params.PageSize = ps
		}
	}
	validatePagination(&params)

	// Verify artist exists
	var artistExists bool
	err := h.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM artists WHERE id = ? OR slug = ?)", artistID, artistID).Scan(&artistExists)
	if err != nil || !artistExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Artist not found"})
		return
	}

	// Count total shows
	countQuery := "SELECT COUNT(*) FROM shows WHERE artist_id = ?"
	var total int64
	err = h.DB.QueryRow(countQuery, artistID).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count shows"})
		return
	}

	// Get paginated shows
	query := `
		SELECT s.id, s.container_id, s.artist_id, a.name as artist_name, s.venue,
		       s.city, s.state, s.date, '' as performance_date_short,
		       '' as performance_date_formatted, '' as container_info, 0 as availability_type,
		       '' as availability_type_str, '' as active_state, s.created_at, s.updated_at
		FROM shows s 
		JOIN artists a ON s.artist_id = a.id
		WHERE s.artist_id = ?
		ORDER BY s.date DESC
		LIMIT ? OFFSET ?
	`

	rows, err := h.DB.Query(query, artistID, params.PageSize, params.Offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query shows"})
		return
	}
	defer rows.Close()

	var shows []Show
	for rows.Next() {
		var show Show
		err := rows.Scan(
			&show.ID, &show.ContainerID, &show.ArtistID, &show.ArtistName,
			&show.VenueName, &show.VenueCity, &show.VenueState,
			&show.PerformanceDate, &show.PerformanceDateShort,
			&show.PerformanceDateFormatted, &show.ContainerInfo,
			&show.AvailabilityType, &show.AvailabilityTypeStr,
			&show.ActiveState, &show.CreatedAt, &show.UpdatedAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan show"})
			return
		}
		shows = append(shows, show)
	}

	response := createPaginatedResponse(shows, params, total)
	c.JSON(http.StatusOK, response)
}

// GetShow returns a specific show by ID or container ID
func (h *CatalogHandler) GetShow(c *gin.Context) {
	showID := c.Param("id")

	query := `
		SELECT s.id, s.container_id, s.artist_id, a.name as artist_name, s.venue,
		       s.city, s.state, s.date, '' as performance_date_short,
		       '' as performance_date_formatted, '' as container_info, 0 as availability_type,
		       '' as availability_type_str, '' as active_state, s.created_at, s.updated_at
		FROM shows s
		JOIN artists a ON s.artist_id = a.id 
		WHERE s.id = ? OR s.container_id = ?
	`

	var show Show
	err := h.DB.QueryRow(query, showID, showID).Scan(
		&show.ID, &show.ContainerID, &show.ArtistID, &show.ArtistName,
		&show.VenueName, &show.VenueCity, &show.VenueState,
		&show.PerformanceDate, &show.PerformanceDateShort,
		&show.PerformanceDateFormatted, &show.ContainerInfo,
		&show.AvailabilityType, &show.AvailabilityTypeStr,
		&show.ActiveState, &show.CreatedAt, &show.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Show not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get show"})
		return
	}

	c.JSON(http.StatusOK, show)
}

// SearchShows performs a comprehensive search across shows
func (h *CatalogHandler) SearchShows(c *gin.Context) {
	// Parse pagination parameters
	params := PaginationParams{}
	if page := c.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil {
			params.Page = p
		}
	}
	if pageSize := c.Query("page_size"); pageSize != "" {
		if ps, err := strconv.Atoi(pageSize); err == nil {
			params.PageSize = ps
		}
	}
	validatePagination(&params)

	// Parse search parameters
	search := c.Query("search")
	artistFilter := c.Query("artist_id")

	// Build WHERE clause
	whereClause := "WHERE 1=1"
	args := []interface{}{}

	if search != "" {
		whereClause += ` AND (
			a.name LIKE ? OR 
			s.venue LIKE ? OR 
			s.city LIKE ? OR 
			s.date LIKE ?
		)`
		searchPattern := "%" + search + "%"
		args = append(args, searchPattern, searchPattern, searchPattern, searchPattern)
	}

	if artistFilter != "" {
		whereClause += " AND s.artist_id = ?"
		args = append(args, artistFilter)
	}

	// Count total records
	countQuery := "SELECT COUNT(*) FROM shows s JOIN artists a ON s.artist_id = a.id " + whereClause
	var total int64
	err := h.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count shows"})
		return
	}

	// Get paginated results
	query := `
		SELECT s.id, s.container_id, s.artist_id, a.name as artist_name, s.venue,
		       s.city, s.state, s.date, '' as performance_date_short,
		       '' as performance_date_formatted, '' as container_info, 0 as availability_type,
		       '' as availability_type_str, '' as active_state, s.created_at, s.updated_at
		FROM shows s
		JOIN artists a ON s.artist_id = a.id ` + whereClause + `
		ORDER BY s.date DESC, a.name ASC
		LIMIT ? OFFSET ?
	`

	args = append(args, params.PageSize, params.Offset)

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query shows"})
		return
	}
	defer rows.Close()

	var shows []Show
	for rows.Next() {
		var show Show
		err := rows.Scan(
			&show.ID, &show.ContainerID, &show.ArtistID, &show.ArtistName,
			&show.VenueName, &show.VenueCity, &show.VenueState,
			&show.PerformanceDate, &show.PerformanceDateShort,
			&show.PerformanceDateFormatted, &show.ContainerInfo,
			&show.AvailabilityType, &show.AvailabilityTypeStr,
			&show.ActiveState, &show.CreatedAt, &show.UpdatedAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan show"})
			return
		}
		shows = append(shows, show)
	}

	response := createPaginatedResponse(shows, params, total)
	c.JSON(http.StatusOK, response)
}
