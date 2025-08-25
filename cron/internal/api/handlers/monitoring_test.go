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

func setupMonitoringTestRouter(t *testing.T) (*gin.Engine, *models.JobManager) {
	db := setupTestDB(t)
	jobManager := models.NewJobManager()

	gin.SetMode(gin.TestMode)
	router := gin.New()

	monitoringHandler := NewMonitoringHandler(db, jobManager)

	monitoring := router.Group("/monitoring")
	{
		monitoring.POST("/monitors", monitoringHandler.CreateMonitor)
		monitoring.POST("/monitors/bulk", monitoringHandler.CreateBulkMonitors)
		monitoring.GET("/monitors", monitoringHandler.GetMonitors)
		monitoring.GET("/monitors/:id", monitoringHandler.GetMonitor)
		monitoring.PUT("/monitors/:id", monitoringHandler.UpdateMonitor)
		monitoring.DELETE("/monitors/:id", monitoringHandler.DeleteMonitor)
		monitoring.POST("/check/all", monitoringHandler.CheckAllMonitors)
		monitoring.POST("/check/artist/:id", monitoringHandler.CheckArtist)
		monitoring.GET("/alerts", monitoringHandler.GetAlerts)
		monitoring.PUT("/alerts/:id/acknowledge", monitoringHandler.AcknowledgeAlert)
		monitoring.GET("/stats", monitoringHandler.GetMonitoringStats)
	}

	return router, jobManager
}

func TestMonitoringHandler_CreateMonitor(t *testing.T) {
	router, _ := setupMonitoringTestRouter(t)

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		checkFields    []string
	}{
		{
			name: "create valid monitor",
			requestBody: map[string]interface{}{
				"artist_id": 1,
				"settings": map[string]interface{}{
					"check_frequency": "hourly",
					"notify_new":      true,
				},
			},
			expectedStatus: http.StatusCreated,
			checkFields:    []string{"monitor_id", "message"},
		},
		{
			name: "create monitor with all settings",
			requestBody: map[string]interface{}{
				"artist_id": 2,
				"settings": map[string]interface{}{
					"check_frequency":     "daily",
					"notify_new":          true,
					"notify_updates":      true,
					"notify_removals":     false,
					"webhook_url":         "https://example.com/webhook",
					"email_notifications": true,
				},
			},
			expectedStatus: http.StatusCreated,
			checkFields:    []string{"monitor_id", "message"},
		},
		{
			name: "missing artist_id",
			requestBody: map[string]interface{}{
				"settings": map[string]interface{}{
					"check_frequency": "hourly",
				},
			},
			expectedStatus: http.StatusBadRequest,
			checkFields:    []string{"error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/monitoring/monitors", bytes.NewBuffer(body))
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

func TestMonitoringHandler_GetMonitors(t *testing.T) {
	router, _ := setupMonitoringTestRouter(t)

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		checkFields    []string
	}{
		{
			name:           "get all monitors",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			checkFields:    []string{"monitors", "pagination"},
		},
		{
			name:           "get monitors with pagination",
			queryParams:    "?page=1&page_size=10",
			expectedStatus: http.StatusOK,
			checkFields:    []string{"monitors", "pagination"},
		},
		{
			name:           "filter by artist_id",
			queryParams:    "?artist_id=1",
			expectedStatus: http.StatusOK,
			checkFields:    []string{"monitors", "pagination"},
		},
		{
			name:           "filter by active status",
			queryParams:    "?active=true",
			expectedStatus: http.StatusOK,
			checkFields:    []string{"monitors", "pagination"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/monitoring/monitors"+tt.queryParams, nil)
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

func TestMonitoringHandler_CheckAllMonitors(t *testing.T) {
	router, _ := setupMonitoringTestRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/monitoring/check/all", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "job_id")
	assert.Contains(t, response, "message")
}

func TestMonitoringHandler_GetAlerts(t *testing.T) {
	router, _ := setupMonitoringTestRouter(t)

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		checkFields    []string
	}{
		{
			name:           "get all alerts",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			checkFields:    []string{"alerts", "pagination"},
		},
		{
			name:           "filter unacknowledged alerts",
			queryParams:    "?acknowledged=false",
			expectedStatus: http.StatusOK,
			checkFields:    []string{"alerts", "pagination"},
		},
		{
			name:           "filter by severity",
			queryParams:    "?severity=high",
			expectedStatus: http.StatusOK,
			checkFields:    []string{"alerts", "pagination"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/monitoring/alerts"+tt.queryParams, nil)
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

func TestMonitoringHandler_GetMonitoringStats(t *testing.T) {
	router, _ := setupMonitoringTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/monitoring/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	expectedFields := []string{
		"total_monitors", "active_monitors", "total_alerts",
		"unacknowledged_alerts", "last_check_time", "next_check_time",
		"check_frequency", "average_response_time",
	}

	for _, field := range expectedFields {
		assert.Contains(t, response, field)
	}
}
