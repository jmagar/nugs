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

func setupDownloadTestRouter(t *testing.T) (*gin.Engine, *models.JobManager) {
	db := setupTestDB(t)
	jobManager := models.NewJobManager()

	gin.SetMode(gin.TestMode)
	router := gin.New()

	downloadHandler := NewDownloadHandler(db, jobManager)

	downloads := router.Group("/downloads")
	{
		downloads.GET("/", downloadHandler.GetDownloads)
		downloads.POST("/queue", downloadHandler.QueueDownload)
		downloads.GET("/queue", downloadHandler.GetDownloadQueue)
		downloads.POST("/queue/reorder", downloadHandler.ReorderQueue)
		downloads.GET("/stats", downloadHandler.GetDownloadStats)
		downloads.GET("/:id", downloadHandler.GetDownload)
		downloads.DELETE("/:id", downloadHandler.CancelDownload)
	}

	return router, jobManager
}

func TestDownloadHandler_QueueDownload(t *testing.T) {
	router, _ := setupDownloadTestRouter(t)

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		checkFields    []string
	}{
		{
			name: "valid download request",
			requestBody: map[string]interface{}{
				"show_id": 1,
				"format":  "flac",
				"quality": "lossless",
			},
			expectedStatus: http.StatusCreated,
			checkFields:    []string{"download_id", "message", "queue_position"},
		},
		{
			name: "download with custom path",
			requestBody: map[string]interface{}{
				"show_id":       2,
				"format":        "mp3",
				"quality":       "hd",
				"download_path": "/custom/path",
			},
			expectedStatus: http.StatusCreated,
			checkFields:    []string{"download_id", "message", "queue_position"},
		},
		{
			name: "missing show_id",
			requestBody: map[string]interface{}{
				"format":  "flac",
				"quality": "lossless",
			},
			expectedStatus: http.StatusBadRequest,
			checkFields:    []string{"error"},
		},
		{
			name: "invalid format",
			requestBody: map[string]interface{}{
				"show_id": 3,
				"format":  "INVALID",
				"quality": "lossless",
			},
			expectedStatus: http.StatusBadRequest,
			checkFields:    []string{"error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/downloads/queue", bytes.NewBuffer(body))
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

func TestDownloadHandler_GetDownloads(t *testing.T) {
	router, _ := setupDownloadTestRouter(t)

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		checkFields    []string
	}{
		{
			name:           "get all downloads",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			checkFields:    []string{"downloads", "pagination"},
		},
		{
			name:           "get downloads with pagination",
			queryParams:    "?page=1&page_size=10",
			expectedStatus: http.StatusOK,
			checkFields:    []string{"downloads", "pagination"},
		},
		{
			name:           "filter by status",
			queryParams:    "?status=completed",
			expectedStatus: http.StatusOK,
			checkFields:    []string{"downloads", "pagination"},
		},
		{
			name:           "filter by format",
			queryParams:    "?format=FLAC",
			expectedStatus: http.StatusOK,
			checkFields:    []string{"downloads", "pagination"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/downloads/"+tt.queryParams, nil)
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

func TestDownloadHandler_GetDownloadQueue(t *testing.T) {
	router, _ := setupDownloadTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/downloads/queue", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "queue")
	assert.Contains(t, response, "stats")

	// Check stats structure
	if stats, ok := response["stats"].(map[string]interface{}); ok {
		assert.Contains(t, stats, "total_items")
		assert.Contains(t, stats, "pending_items")
		assert.Contains(t, stats, "processing_items")
	}
}

func TestDownloadHandler_ReorderQueue(t *testing.T) {
	router, _ := setupDownloadTestRouter(t)

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		checkError     bool
	}{
		{
			name: "valid reorder request",
			requestBody: map[string]interface{}{
				"download_ids": []string{"1", "2", "3"},
			},
			expectedStatus: http.StatusOK,
			checkError:     false,
		},
		{
			name: "empty download_ids",
			requestBody: map[string]interface{}{
				"download_ids": []string{},
			},
			expectedStatus: http.StatusBadRequest,
			checkError:     true,
		},
		{
			name:           "missing download_ids",
			requestBody:    map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
			checkError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/downloads/queue/reorder", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			if tt.checkError {
				assert.Contains(t, response, "error")
			} else {
				assert.Contains(t, response, "message")
			}
		})
	}
}

func TestDownloadHandler_GetDownloadStats(t *testing.T) {
	router, _ := setupDownloadTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/downloads/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	expectedFields := []string{
		"total_downloads", "completed_downloads", "failed_downloads",
		"pending_downloads", "total_size_gb", "queue_length",
		"active_downloads", "average_speed_mbps",
	}

	for _, field := range expectedFields {
		assert.Contains(t, response, field)
	}
}

func TestDownloadHandler_CancelDownload(t *testing.T) {
	router, _ := setupDownloadTestRouter(t)

	// Create a test download first
	downloadReq := map[string]interface{}{
		"show_id": 1,
		"format":  "flac",
		"quality": "lossless",
	}
	body, _ := json.Marshal(downloadReq)
	req := httptest.NewRequest(http.MethodPost, "/downloads/queue", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Extract download_id from response for use in cancel test
	var queueResp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &queueResp); err != nil {
		t.Logf("Warning: failed to unmarshal queue response: %v", err)
	}
	createdDownloadID := "1" // Use ID 1 as that's what gets created

	tests := []struct {
		name           string
		downloadID     string
		expectedStatus int
		checkError     bool
	}{
		{
			name:           "cancel existing download",
			downloadID:     createdDownloadID,
			expectedStatus: http.StatusOK,
			checkError:     false,
		},
		{
			name:           "cancel non-existent download",
			downloadID:     "99999",
			expectedStatus: http.StatusNotFound,
			checkError:     true,
		},
		{
			name:           "invalid download ID",
			downloadID:     "invalid",
			expectedStatus: http.StatusBadRequest,
			checkError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/downloads/"+tt.downloadID, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			if tt.checkError {
				assert.Contains(t, response, "error")
			} else {
				assert.Contains(t, response, "message")
			}
		})
	}
}
