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

func setupAdminTestRouter(t *testing.T) (*gin.Engine, *models.JobManager) {
	db := setupTestDB(t)
	jobManager := models.NewJobManager()

	gin.SetMode(gin.TestMode)
	router := gin.New()

	adminHandler := NewAdminHandler(db, jobManager)

	admin := router.Group("/admin")
	{
		admin.POST("/users", adminHandler.CreateUser)
		admin.GET("/users", adminHandler.GetUsers)
		admin.PUT("/users/:id", adminHandler.UpdateUser)
		admin.DELETE("/users/:id", adminHandler.DeleteUser)
		admin.GET("/config", adminHandler.GetSystemConfig)
		admin.PUT("/config/:key", adminHandler.UpdateConfig)
		admin.GET("/status", adminHandler.GetSystemStatus)
		admin.GET("/stats", adminHandler.GetAdminStats)
		admin.POST("/maintenance/cleanup", adminHandler.RunCleanup)
		admin.GET("/audit", adminHandler.GetAuditLogs)
		admin.GET("/jobs", adminHandler.GetJobs)
		admin.GET("/jobs/:id", adminHandler.GetJob)
		admin.DELETE("/jobs/:id", adminHandler.CancelJob)
		admin.POST("/database/backup", adminHandler.CreateDatabaseBackup)
		admin.POST("/database/optimize", adminHandler.OptimizeDatabase)
		admin.GET("/database/stats", adminHandler.GetDatabaseStats)
	}

	return router, jobManager
}

func TestAdminHandler_CreateUser(t *testing.T) {
	router, _ := setupAdminTestRouter(t)

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		checkFields    []string
	}{
		{
			name: "create valid user",
			requestBody: map[string]interface{}{
				"username": "testuser",
				"email":    "test@example.com",
				"password": "testpassword123",
				"role":     "user",
			},
			expectedStatus: http.StatusCreated,
			checkFields:    []string{"user_id", "message"},
		},
		{
			name: "create admin user",
			requestBody: map[string]interface{}{
				"username": "adminuser",
				"email":    "newadmin@example.com",
				"password": "adminpass123",
				"role":     "admin",
			},
			expectedStatus: http.StatusCreated,
			checkFields:    []string{"user_id", "message"},
		},
		{
			name: "missing username",
			requestBody: map[string]interface{}{
				"email":    "test@example.com",
				"password": "testpass123",
				"role":     "user",
			},
			expectedStatus: http.StatusBadRequest,
			checkFields:    []string{"error"},
		},
		{
			name: "invalid email",
			requestBody: map[string]interface{}{
				"username": "testuser",
				"email":    "invalid-email",
				"password": "testpass123",
				"role":     "user",
			},
			expectedStatus: http.StatusBadRequest,
			checkFields:    []string{"error"},
		},
		{
			name: "weak password",
			requestBody: map[string]interface{}{
				"username": "testuser",
				"email":    "test@example.com",
				"password": "123",
				"role":     "user",
			},
			expectedStatus: http.StatusBadRequest,
			checkFields:    []string{"error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/admin/users", bytes.NewBuffer(body))
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

func TestAdminHandler_GetUsers(t *testing.T) {
	router, _ := setupAdminTestRouter(t)

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		checkFields    []string
	}{
		{
			name:           "get all users",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			checkFields:    []string{"data", "page", "page_size", "total"},
		},
		{
			name:           "get users with pagination",
			queryParams:    "?page=1&page_size=10",
			expectedStatus: http.StatusOK,
			checkFields:    []string{"data", "page", "page_size", "total"},
		},
		{
			name:           "filter by role",
			queryParams:    "?role=admin",
			expectedStatus: http.StatusOK,
			checkFields:    []string{"data", "page", "page_size", "total"},
		},
		{
			name:           "filter by active status",
			queryParams:    "?active=true",
			expectedStatus: http.StatusOK,
			checkFields:    []string{"data", "page", "page_size", "total"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/admin/users"+tt.queryParams, nil)
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

func TestAdminHandler_GetSystemConfig(t *testing.T) {
	router, _ := setupAdminTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/config", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "data")

	if configs, ok := response["data"].([]interface{}); ok {
		assert.True(t, len(configs) > 0, "Should have configuration data")

		// Check that we have at least some expected keys
		configKeys := make(map[string]bool)
		for _, configItem := range configs {
			if configMap, ok := configItem.(map[string]interface{}); ok {
				if key, exists := configMap["key"]; exists {
					configKeys[key.(string)] = true
				}
			}
		}

		expectedKeys := []string{"auto_refresh_enabled", "max_concurrent_downloads", "webhook_timeout_seconds"}
		for _, key := range expectedKeys {
			assert.True(t, configKeys[key], "Config should contain key: "+key)
		}
	}
}

func TestAdminHandler_UpdateConfig(t *testing.T) {
	router, _ := setupAdminTestRouter(t)

	tests := []struct {
		name           string
		configKey      string
		requestBody    map[string]interface{}
		expectedStatus int
		checkError     bool
	}{
		{
			name:      "update valid config key",
			configKey: "max_concurrent_downloads",
			requestBody: map[string]interface{}{
				"value": 5,
			},
			expectedStatus: http.StatusOK,
			checkError:     false,
		},
		{
			name:      "update string config",
			configKey: "default_download_path",
			requestBody: map[string]interface{}{
				"value": "/new/downloads",
			},
			expectedStatus: http.StatusOK,
			checkError:     false,
		},
		{
			name:      "update boolean config",
			configKey: "auto_refresh_enabled",
			requestBody: map[string]interface{}{
				"value": true,
			},
			expectedStatus: http.StatusOK,
			checkError:     false,
		},
		{
			name:           "missing value",
			configKey:      "max_concurrent_downloads",
			requestBody:    map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
			checkError:     true,
		},
		{
			name:      "invalid config key",
			configKey: "invalid_key",
			requestBody: map[string]interface{}{
				"value": "test",
			},
			expectedStatus: http.StatusBadRequest,
			checkError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPut, "/admin/config/"+tt.configKey, bytes.NewBuffer(body))
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
				assert.Contains(t, response, "success")
				assert.True(t, response["success"].(bool))
			}
		})
	}
}

func TestAdminHandler_GetSystemStatus(t *testing.T) {
	router, _ := setupAdminTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	expectedFields := []string{
		"status", "uptime", "version", "database",
		"jobs", "performance", "health", "storage", "last_updated",
	}

	for _, field := range expectedFields {
		assert.Contains(t, response, field)
	}

	// Verify nested structure
	if database, ok := response["database"].(map[string]interface{}); ok {
		assert.Contains(t, database, "connected")
		assert.Contains(t, database, "record_counts")
	}

	if jobs, ok := response["jobs"].(map[string]interface{}); ok {
		assert.Contains(t, jobs, "active_jobs")
		assert.Contains(t, jobs, "pending_jobs")
	}
}

func TestAdminHandler_RunCleanup(t *testing.T) {
	router, _ := setupAdminTestRouter(t)

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		checkFields    []string
	}{
		{
			name: "run cleanup with options",
			requestBody: map[string]interface{}{
				"clean_old_jobs":     true,
				"clean_old_logs":     true,
				"optimize_database":  false,
				"job_retention_days": 30,
			},
			expectedStatus: http.StatusOK,
			checkFields:    []string{"job_id", "message"},
		},
		{
			name:           "run cleanup with defaults",
			requestBody:    map[string]interface{}{},
			expectedStatus: http.StatusOK,
			checkFields:    []string{"job_id", "message"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/admin/maintenance/cleanup", bytes.NewBuffer(body))
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

func TestAdminHandler_GetJobs(t *testing.T) {
	router, jobManager := setupAdminTestRouter(t)

	// Create test jobs
	jobManager.CreateJob(models.JobTypeCatalogRefresh)
	jobManager.CreateJob(models.JobTypeDownload)

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		checkFields    []string
	}{
		{
			name:           "get all jobs",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			checkFields:    []string{"data", "total"},
		},
		{
			name:           "filter by status",
			queryParams:    "?status=pending",
			expectedStatus: http.StatusOK,
			checkFields:    []string{"data", "total"},
		},
		{
			name:           "filter by type",
			queryParams:    "?type=catalog_refresh",
			expectedStatus: http.StatusOK,
			checkFields:    []string{"data", "total"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/admin/jobs"+tt.queryParams, nil)
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

func TestAdminHandler_GetDatabaseStats(t *testing.T) {
	router, _ := setupAdminTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/database/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	expectedFields := []string{
		"table_counts", "total_tables",
	}

	for _, field := range expectedFields {
		assert.Contains(t, response, field)
	}
}
