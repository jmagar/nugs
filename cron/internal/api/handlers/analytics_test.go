package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jmagar/nugs/cron/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAnalyticsTestRouter(t *testing.T) (*gin.Engine, *models.JobManager) {
	db := setupTestDB(t)
	jobManager := models.NewJobManager()

	gin.SetMode(gin.TestMode)
	router := gin.New()

	analyticsHandler := NewAnalyticsHandler(db, jobManager)

	analytics := router.Group("/analytics")
	{
		analytics.POST("/reports", analyticsHandler.GenerateReport)
		analytics.GET("/collection", analyticsHandler.GetCollectionStats)
		analytics.GET("/artists", analyticsHandler.GetArtistAnalytics)
		analytics.GET("/downloads", analyticsHandler.GetDownloadAnalytics)
		analytics.GET("/system", analyticsHandler.GetSystemMetrics)
		analytics.GET("/performance", analyticsHandler.GetPerformanceMetrics)
		analytics.GET("/top/artists", analyticsHandler.GetTopArtists)
		analytics.GET("/top/venues", analyticsHandler.GetTopVenues)
		analytics.GET("/trends/downloads", analyticsHandler.GetDownloadTrends)
		analytics.GET("/summary", analyticsHandler.GetDashboardSummary)
		analytics.GET("/health", analyticsHandler.GetHealthScore)
	}

	return router, jobManager
}

func TestAnalyticsHandler_GenerateReport(t *testing.T) {
	router, _ := setupAnalyticsTestRouter(t)

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		checkFields    []string
	}{
		{
			name: "generate collection report",
			requestBody: map[string]interface{}{
				"report_type": "collection",
				"format":      "json",
				"date_range": map[string]string{
					"start": "2024-01-01",
					"end":   "2024-12-31",
				},
			},
			expectedStatus: http.StatusOK,
			checkFields:    []string{"report_id", "report_type", "generated_at"},
		},
		{
			name: "generate download report",
			requestBody: map[string]interface{}{
				"report_type": "downloads",
				"format":      "csv",
				"filters": map[string]interface{}{
					"status": "completed",
				},
			},
			expectedStatus: http.StatusOK,
			checkFields:    []string{"report_id", "report_type", "generated_at"},
		},
		{
			name: "invalid report type",
			requestBody: map[string]interface{}{
				"report_type": "invalid",
				"format":      "json",
			},
			expectedStatus: http.StatusBadRequest,
			checkFields:    []string{"error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/analytics/reports", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

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

func TestAnalyticsHandler_GetCollectionStats(t *testing.T) {
	router, _ := setupAnalyticsTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/analytics/collection", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	expectedFields := []string{
		"total_artists", "total_shows", "total_downloads",
		"total_size_gb", "average_shows_per_artist", "recent_activity",
	}

	for _, field := range expectedFields {
		assert.Contains(t, response, field)
	}
}

func TestAnalyticsHandler_GetArtistAnalytics(t *testing.T) {
	router, _ := setupAnalyticsTestRouter(t)

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		checkFields    []string
	}{
		{
			name:           "get all artist analytics",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			checkFields:    []string{"data", "total", "timeframe"},
		},
		{
			name:           "get top artists by show count",
			queryParams:    "?sort_by=show_count&limit=10",
			expectedStatus: http.StatusOK,
			checkFields:    []string{"data", "total", "timeframe"},
		},
		{
			name:           "filter by date range",
			queryParams:    "?date_from=2020-01-01&date_to=2024-01-01",
			expectedStatus: http.StatusOK,
			checkFields:    []string{"data", "total", "timeframe"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/analytics/artists"+tt.queryParams, nil)
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

func TestAnalyticsHandler_GetDownloadAnalytics(t *testing.T) {
	router, _ := setupAnalyticsTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/analytics/downloads", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Check top-level structure
	assert.Contains(t, response, "data")
	assert.Contains(t, response, "timeframe")

	// Check nested data structure
	if data, ok := response["data"].(map[string]interface{}); ok {
		expectedDataFields := []string{
			"total_downloads", "completed_downloads", "failed_downloads",
			"total_size_gb", "format_breakdown", "quality_breakdown",
			"peak_download_hours",
		}

		for _, field := range expectedDataFields {
			assert.Contains(t, data, field)
		}
	}
}

func TestAnalyticsHandler_GetSystemMetrics(t *testing.T) {
	router, _ := setupAnalyticsTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/analytics/system", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	expectedFields := []string{
		"database_size_mb", "available_storage_gb", "total_storage_gb",
		"active_downloads", "active_monitors", "api_requests",
		"system_uptime", "job_stats", "total_files",
	}

	for _, field := range expectedFields {
		assert.Contains(t, response, field)
	}
}

func TestAnalyticsHandler_GetDashboardSummary(t *testing.T) {
	router, _ := setupAnalyticsTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/analytics/summary", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	expectedSections := []string{
		"collection", "popular_formats", "recent_activity", "system_status",
	}

	for _, section := range expectedSections {
		assert.Contains(t, response, section)
	}
}

func TestAnalyticsHandler_GetHealthScore(t *testing.T) {
	router, _ := setupAnalyticsTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/analytics/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	expectedFields := []string{
		"overall", "categories", "issues", "last_updated", "recommendations",
	}

	for _, field := range expectedFields {
		assert.Contains(t, response, field)
	}
}
