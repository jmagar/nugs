package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupCatalogTestRouter(t *testing.T) *gin.Engine {
	db := setupTestDB(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()

	catalogHandler := NewCatalogHandler(db)

	catalog := router.Group("/catalog")
	{
		catalog.GET("/artists", catalogHandler.GetArtists)
		catalog.GET("/artists/:id", catalogHandler.GetArtist)
		catalog.GET("/artists/:id/shows", catalogHandler.GetArtistShows)
		catalog.GET("/shows/search", catalogHandler.SearchShows)
		catalog.GET("/shows/:id", catalogHandler.GetShow)
	}

	return router
}

func TestCatalogHandler_GetArtists(t *testing.T) {
	router := setupCatalogTestRouter(t)

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		checkFields    []string
	}{
		{
			name:           "get artists without pagination",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			checkFields:    []string{"data", "total", "page"},
		},
		{
			name:           "get artists with pagination",
			queryParams:    "?page=1&page_size=10",
			expectedStatus: http.StatusOK,
			checkFields:    []string{"data", "total", "page"},
		},
		{
			name:           "get artists with search",
			queryParams:    "?search=grateful",
			expectedStatus: http.StatusOK,
			checkFields:    []string{"data", "total", "page"},
		},
		{
			name:           "get artists with sorting",
			queryParams:    "?sort_by=name&sort_order=desc",
			expectedStatus: http.StatusOK,
			checkFields:    []string{"data", "total", "page"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/catalog/artists"+tt.queryParams, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			for _, field := range tt.checkFields {
				assert.Contains(t, response, field)
			}
		})
	}
}

func TestCatalogHandler_GetArtist(t *testing.T) {
	router := setupCatalogTestRouter(t)

	tests := []struct {
		name           string
		artistID       string
		expectedStatus int
		checkError     bool
	}{
		{
			name:           "get existing artist",
			artistID:       "1",
			expectedStatus: http.StatusOK,
			checkError:     false,
		},
		{
			name:           "get non-existent artist",
			artistID:       "99999",
			expectedStatus: http.StatusNotFound,
			checkError:     true,
		},
		{
			name:           "invalid artist ID",
			artistID:       "invalid",
			expectedStatus: http.StatusNotFound,
			checkError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/catalog/artists/"+tt.artistID, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			if tt.checkError {
				assert.Contains(t, response, "error")
			} else {
				// API returns artist data directly, check for artist fields
				assert.Contains(t, response, "id")
				assert.Contains(t, response, "name")
				assert.Contains(t, response, "slug")
			}
		})
	}
}

func TestCatalogHandler_SearchShows(t *testing.T) {
	router := setupCatalogTestRouter(t)

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		checkFields    []string
	}{
		{
			name:           "search shows by artist",
			queryParams:    "?artist=grateful+dead",
			expectedStatus: http.StatusOK,
			checkFields:    []string{"data", "total", "page"},
		},
		{
			name:           "search shows by venue",
			queryParams:    "?venue=madison+square+garden",
			expectedStatus: http.StatusOK,
			checkFields:    []string{"data", "total", "page"},
		},
		{
			name:           "search shows by date range",
			queryParams:    "?date_from=2020-01-01&date_to=2020-12-31",
			expectedStatus: http.StatusOK,
			checkFields:    []string{"data", "total", "page"},
		},
		{
			name:           "search shows with all filters",
			queryParams:    "?artist=dead&venue=garden&date_from=2020-01-01&page=1&page_size=5",
			expectedStatus: http.StatusOK,
			checkFields:    []string{"data", "total", "page"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/catalog/shows/search"+tt.queryParams, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			for _, field := range tt.checkFields {
				assert.Contains(t, response, field)
			}
		})
	}
}

func TestCatalogHandler_GetShow(t *testing.T) {
	router := setupCatalogTestRouter(t)

	tests := []struct {
		name           string
		showID         string
		expectedStatus int
		checkError     bool
	}{
		{
			name:           "get existing show",
			showID:         "1",
			expectedStatus: http.StatusOK,
			checkError:     false,
		},
		{
			name:           "get non-existent show",
			showID:         "99999",
			expectedStatus: http.StatusNotFound,
			checkError:     true,
		},
		{
			name:           "invalid show ID",
			showID:         "invalid",
			expectedStatus: http.StatusNotFound,
			checkError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/catalog/shows/"+tt.showID, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			if tt.checkError {
				assert.Contains(t, response, "error")
			} else {
				// API returns show data directly, check for show fields
				assert.Contains(t, response, "id")
				assert.Contains(t, response, "container_id")
				assert.Contains(t, response, "artist_name")
				assert.Contains(t, response, "venue_name")
			}
		})
	}
}
