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

func setupSchedulerTestRouter(t *testing.T) (*gin.Engine, *models.JobManager) {
	db := setupTestDB(t)
	jobManager := models.NewJobManager()

	gin.SetMode(gin.TestMode)
	router := gin.New()

	schedulerHandler := NewSchedulerHandler(db, jobManager)

	scheduler := router.Group("/scheduler")
	{
		scheduler.POST("/start", schedulerHandler.StartScheduler)
		scheduler.POST("/stop", schedulerHandler.StopScheduler)
		scheduler.GET("/status", schedulerHandler.GetSchedulerStatus)
		scheduler.GET("/stats", schedulerHandler.GetSchedulerStats)
		scheduler.POST("/schedules", schedulerHandler.CreateSchedule)
		scheduler.GET("/schedules", schedulerHandler.GetSchedules)
		scheduler.GET("/schedules/:id", schedulerHandler.GetSchedule)
		scheduler.PUT("/schedules/:id", schedulerHandler.UpdateSchedule)
		scheduler.DELETE("/schedules/:id", schedulerHandler.DeleteSchedule)
		scheduler.POST("/schedules/bulk", schedulerHandler.BulkScheduleOperation)
		scheduler.GET("/schedules/:id/executions", schedulerHandler.GetScheduleExecutions)
		scheduler.GET("/executions", schedulerHandler.GetAllExecutions)
		scheduler.GET("/templates", schedulerHandler.GetScheduleTemplates)
		scheduler.GET("/cron-patterns", schedulerHandler.GetCronPatterns)
	}

	return router, jobManager
}

func TestSchedulerHandler_StartScheduler(t *testing.T) {
	router, _ := setupSchedulerTestRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/scheduler/start", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "message")
	assert.Contains(t, response, "status")
}

func TestSchedulerHandler_StopScheduler(t *testing.T) {
	router, _ := setupSchedulerTestRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/scheduler/stop", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "message")
	assert.Contains(t, response, "status")
}

func TestSchedulerHandler_GetSchedulerStatus(t *testing.T) {
	router, _ := setupSchedulerTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/scheduler/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	expectedFields := []string{
		"running", "uptime_seconds", "total_schedules", "active_schedules",
		"next_execution", "last_execution", "total_executions",
	}

	for _, field := range expectedFields {
		assert.Contains(t, response, field)
	}
}

func TestSchedulerHandler_CreateSchedule(t *testing.T) {
	router, _ := setupSchedulerTestRouter(t)

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		checkFields    []string
	}{
		{
			name: "create catalog refresh schedule",
			requestBody: map[string]interface{}{
				"name":        "Daily Catalog Refresh",
				"cron":        "0 2 * * *",
				"job_type":    "catalog_refresh",
				"enabled":     true,
				"description": "Refresh catalog every day at 2 AM",
			},
			expectedStatus: http.StatusCreated,
			checkFields:    []string{"schedule_id", "message"},
		},
		{
			name: "create download schedule with config",
			requestBody: map[string]interface{}{
				"name":     "Auto Download Check",
				"cron":     "*/15 * * * *",
				"job_type": "download_check",
				"enabled":  true,
				"config": map[string]interface{}{
					"max_downloads": 3,
					"format":        "FLAC",
				},
			},
			expectedStatus: http.StatusCreated,
			checkFields:    []string{"schedule_id", "message"},
		},
		{
			name: "missing required fields",
			requestBody: map[string]interface{}{
				"name": "Test Schedule",
			},
			expectedStatus: http.StatusBadRequest,
			checkFields:    []string{"error"},
		},
		{
			name: "invalid cron expression",
			requestBody: map[string]interface{}{
				"name":     "Invalid Schedule",
				"cron":     "invalid-cron",
				"job_type": "catalog_refresh",
			},
			expectedStatus: http.StatusBadRequest,
			checkFields:    []string{"error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/scheduler/schedules", bytes.NewBuffer(body))
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

func TestSchedulerHandler_GetSchedules(t *testing.T) {
	router, _ := setupSchedulerTestRouter(t)

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		checkFields    []string
	}{
		{
			name:           "get all schedules",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			checkFields:    []string{"schedules", "pagination"},
		},
		{
			name:           "filter by enabled status",
			queryParams:    "?enabled=true",
			expectedStatus: http.StatusOK,
			checkFields:    []string{"schedules", "pagination"},
		},
		{
			name:           "filter by job type",
			queryParams:    "?job_type=catalog_refresh",
			expectedStatus: http.StatusOK,
			checkFields:    []string{"schedules", "pagination"},
		},
		{
			name:           "search by name",
			queryParams:    "?search=refresh",
			expectedStatus: http.StatusOK,
			checkFields:    []string{"schedules", "pagination"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/scheduler/schedules"+tt.queryParams, nil)
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

func TestSchedulerHandler_UpdateSchedule(t *testing.T) {
	router, _ := setupSchedulerTestRouter(t)

	tests := []struct {
		name           string
		scheduleID     string
		requestBody    map[string]interface{}
		expectedStatus int
		checkError     bool
	}{
		{
			name:       "update schedule cron",
			scheduleID: "1",
			requestBody: map[string]interface{}{
				"cron": "0 3 * * *",
			},
			expectedStatus: http.StatusOK,
			checkError:     false,
		},
		{
			name:       "disable schedule",
			scheduleID: "1",
			requestBody: map[string]interface{}{
				"enabled": false,
			},
			expectedStatus: http.StatusOK,
			checkError:     false,
		},
		{
			name:       "update non-existent schedule",
			scheduleID: "99999",
			requestBody: map[string]interface{}{
				"enabled": true,
			},
			expectedStatus: http.StatusNotFound,
			checkError:     true,
		},
		{
			name:       "invalid schedule ID",
			scheduleID: "invalid",
			requestBody: map[string]interface{}{
				"enabled": true,
			},
			expectedStatus: http.StatusBadRequest,
			checkError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPut, "/scheduler/schedules/"+tt.scheduleID, bytes.NewBuffer(body))
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

func TestSchedulerHandler_GetScheduleTemplates(t *testing.T) {
	router, _ := setupSchedulerTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/scheduler/templates", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "templates")

	if templates, ok := response["templates"].([]interface{}); ok {
		assert.Greater(t, len(templates), 0)

		// Check template structure
		if len(templates) > 0 {
			if template, ok := templates[0].(map[string]interface{}); ok {
				assert.Contains(t, template, "name")
				assert.Contains(t, template, "description")
				assert.Contains(t, template, "cron")
				assert.Contains(t, template, "job_type")
			}
		}
	}
}

func TestSchedulerHandler_GetCronPatterns(t *testing.T) {
	router, _ := setupSchedulerTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/scheduler/cron-patterns", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "patterns")

	if patterns, ok := response["patterns"].([]interface{}); ok {
		assert.Greater(t, len(patterns), 0)

		// Check pattern structure
		if len(patterns) > 0 {
			if pattern, ok := patterns[0].(map[string]interface{}); ok {
				assert.Contains(t, pattern, "name")
				assert.Contains(t, pattern, "cron")
				assert.Contains(t, pattern, "description")
			}
		}
	}
}

func TestSchedulerHandler_BulkScheduleOperation(t *testing.T) {
	router, _ := setupSchedulerTestRouter(t)

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		checkFields    []string
	}{
		{
			name: "bulk enable schedules",
			requestBody: map[string]interface{}{
				"operation":    "enable",
				"schedule_ids": []string{"1", "2", "3"},
			},
			expectedStatus: http.StatusOK,
			checkFields:    []string{"message", "affected_count"},
		},
		{
			name: "bulk disable schedules",
			requestBody: map[string]interface{}{
				"operation":    "disable",
				"schedule_ids": []string{"1", "2"},
			},
			expectedStatus: http.StatusOK,
			checkFields:    []string{"message", "affected_count"},
		},
		{
			name: "bulk delete schedules",
			requestBody: map[string]interface{}{
				"operation":    "delete",
				"schedule_ids": []string{"1"},
			},
			expectedStatus: http.StatusOK,
			checkFields:    []string{"message", "affected_count"},
		},
		{
			name: "invalid operation",
			requestBody: map[string]interface{}{
				"operation":    "invalid",
				"schedule_ids": []string{"1"},
			},
			expectedStatus: http.StatusBadRequest,
			checkFields:    []string{"error"},
		},
		{
			name: "missing schedule IDs",
			requestBody: map[string]interface{}{
				"operation": "enable",
			},
			expectedStatus: http.StatusBadRequest,
			checkFields:    []string{"error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/scheduler/schedules/bulk", bytes.NewBuffer(body))
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
