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

func setupRefreshTestRouter(t *testing.T) (*gin.Engine, *models.JobManager) {
	db := setupTestDB(t)
	jobManager := models.NewJobManager()

	gin.SetMode(gin.TestMode)
	router := gin.New()

	refreshHandler := NewRefreshHandler(db, jobManager)

	catalog := router.Group("/catalog")
	{
		catalog.POST("/refresh", refreshHandler.StartRefresh)
		catalog.GET("/refresh/status/:job_id", refreshHandler.GetRefreshStatus)
		catalog.GET("/refresh/jobs", refreshHandler.ListRefreshJobs)
		catalog.DELETE("/refresh/:job_id", refreshHandler.CancelRefresh)
		catalog.GET("/refresh/info", refreshHandler.GetRefreshInfo)
	}

	return router, jobManager
}

func TestRefreshHandler_StartRefresh(t *testing.T) {
	router, _ := setupRefreshTestRouter(t)

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		checkFields    []string
	}{
		{
			name: "start refresh with force",
			requestBody: map[string]interface{}{
				"force": true,
			},
			expectedStatus: http.StatusAccepted,
			checkFields:    []string{"job_id", "message"},
		},
		{
			name: "start refresh without force",
			requestBody: map[string]interface{}{
				"force": false,
			},
			expectedStatus: http.StatusAccepted,
			checkFields:    []string{"job_id", "message"},
		},
		{
			name:           "start refresh with empty body",
			requestBody:    map[string]interface{}{},
			expectedStatus: http.StatusAccepted,
			checkFields:    []string{"job_id", "message"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/catalog/refresh", bytes.NewBuffer(body))
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

func TestRefreshHandler_GetRefreshStatus(t *testing.T) {
	router, jobManager := setupRefreshTestRouter(t)

	// Create a test job
	job := jobManager.CreateJob(models.JobTypeCatalogRefresh)

	tests := []struct {
		name           string
		jobID          string
		expectedStatus int
		checkError     bool
	}{
		{
			name:           "get status of existing job",
			jobID:          job.ID,
			expectedStatus: http.StatusOK,
			checkError:     false,
		},
		{
			name:           "get status of non-existent job",
			jobID:          "non-existent",
			expectedStatus: http.StatusNotFound,
			checkError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/catalog/refresh/status/"+tt.jobID, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			if tt.checkError {
				assert.Contains(t, response, "error")
			} else {
				assert.Contains(t, response, "job")
				assert.Contains(t, response, "status")
			}
		})
	}
}

func TestRefreshHandler_ListRefreshJobs(t *testing.T) {
	router, jobManager := setupRefreshTestRouter(t)

	// Create test jobs
	jobManager.CreateJob(models.JobTypeCatalogRefresh)
	jobManager.CreateJob(models.JobTypeCatalogRefresh)

	req := httptest.NewRequest(http.MethodGet, "/catalog/refresh/jobs", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "jobs")
	assert.Contains(t, response, "pagination")

	if jobs, ok := response["jobs"].([]interface{}); ok {
		assert.Equal(t, 2, len(jobs))
	}
}

func TestRefreshHandler_CancelRefresh(t *testing.T) {
	router, jobManager := setupRefreshTestRouter(t)

	// Create a test job
	job := jobManager.CreateJob(models.JobTypeCatalogRefresh)

	tests := []struct {
		name           string
		jobID          string
		expectedStatus int
		checkError     bool
	}{
		{
			name:           "cancel existing job",
			jobID:          job.ID,
			expectedStatus: http.StatusOK,
			checkError:     false,
		},
		{
			name:           "cancel non-existent job",
			jobID:          "non-existent",
			expectedStatus: http.StatusNotFound,
			checkError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/catalog/refresh/"+tt.jobID, nil)
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

func TestRefreshHandler_GetRefreshInfo(t *testing.T) {
	router, _ := setupRefreshTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/catalog/refresh/info", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	expectedFields := []string{
		"last_refresh", "next_scheduled_refresh", "refresh_frequency",
		"total_artists", "total_shows", "auto_refresh_enabled",
	}

	for _, field := range expectedFields {
		assert.Contains(t, response, field)
	}
}
