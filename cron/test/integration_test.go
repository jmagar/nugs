package test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jmagar/nugs/cron/internal/api/handlers"
	"github.com/jmagar/nugs/cron/internal/api/middleware"
	"github.com/jmagar/nugs/cron/internal/database"
	"github.com/jmagar/nugs/cron/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type IntegrationTestSuite struct {
	suite.Suite
	db         *sql.DB
	router     *gin.Engine
	jobManager *models.JobManager
	authToken  string
}

func (suite *IntegrationTestSuite) SetupSuite() {
	// Setup test database
	db, err := database.Initialize(":memory:")
	require.NoError(suite.T(), err)
	suite.db = db

	// Setup job manager
	suite.jobManager = models.NewJobManager()

	// Setup test router
	suite.router = suite.setupTestRouter()

	// Get auth token
	suite.authToken = suite.getAuthToken()
}

func (suite *IntegrationTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
}

func (suite *IntegrationTestSuite) setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	jwtSecret := []byte("test-secret")

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(suite.db, jwtSecret)
	catalogHandler := handlers.NewCatalogHandler(suite.db)
	refreshHandler := handlers.NewRefreshHandler(suite.db, suite.jobManager)
	downloadHandler := handlers.NewDownloadHandler(suite.db, suite.jobManager)
	monitoringHandler := handlers.NewMonitoringHandler(suite.db, suite.jobManager)
	analyticsHandler := handlers.NewAnalyticsHandler(suite.db, suite.jobManager)

	// Setup routes
	v1 := router.Group("/api/v1")
	{
		// Auth routes (no auth required)
		auth := v1.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/logout", authHandler.Logout)
		}

		// Protected routes
		protected := v1.Group("/")
		protected.Use(middleware.JWTAuth(string(jwtSecret)))
		{
			// Catalog endpoints
			catalog := protected.Group("/catalog")
			{
				catalog.GET("/artists", catalogHandler.GetArtists)
				catalog.GET("/artists/:id", catalogHandler.GetArtist)
				catalog.GET("/shows/search", catalogHandler.SearchShows)
				catalog.POST("/refresh", refreshHandler.StartRefresh)
				catalog.GET("/refresh/status/:job_id", refreshHandler.GetRefreshStatus)
			}

			// Download endpoints
			downloads := protected.Group("/downloads")
			{
				downloads.POST("/queue", downloadHandler.QueueDownload)
				downloads.GET("/queue", downloadHandler.GetDownloadQueue)
				downloads.GET("/stats", downloadHandler.GetDownloadStats)
			}

			// Monitoring endpoints
			monitoring := protected.Group("/monitoring")
			{
				monitoring.POST("/monitors", monitoringHandler.CreateMonitor)
				monitoring.GET("/monitors", monitoringHandler.GetMonitors)
				monitoring.POST("/check/all", monitoringHandler.CheckAllMonitors)
			}

			// Analytics endpoints
			analytics := protected.Group("/analytics")
			{
				analytics.GET("/collection", analyticsHandler.GetCollectionStats)
				analytics.GET("/summary", analyticsHandler.GetDashboardSummary)
			}
		}
	}

	return router
}

func (suite *IntegrationTestSuite) getAuthToken() string {
	loginData := map[string]string{
		"username": "admin",
		"password": "admin123",
	}

	body, _ := json.Marshal(loginData)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if token, ok := response["token"].(string); ok {
		return token
	}
	return ""
}

func (suite *IntegrationTestSuite) makeAuthenticatedRequest(method, path string, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewBuffer(body))
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	if suite.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+suite.authToken)
	}

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	return w
}

func (suite *IntegrationTestSuite) TestCompleteWorkflow() {
	t := suite.T()

	// 1. Test catalog refresh workflow
	t.Run("CatalogRefreshWorkflow", func(t *testing.T) {
		// Start catalog refresh
		refreshData := map[string]bool{"force": true}
		body, _ := json.Marshal(refreshData)

		w := suite.makeAuthenticatedRequest(http.MethodPost, "/api/v1/catalog/refresh", body)
		assert.Equal(t, http.StatusAccepted, w.Code)

		var refreshResponse map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &refreshResponse)
		require.NoError(t, err)

		jobID, ok := refreshResponse["job_id"].(string)
		require.True(t, ok)
		require.NotEmpty(t, jobID)

		// Check refresh status
		w = suite.makeAuthenticatedRequest(http.MethodGet, "/api/v1/catalog/refresh/status/"+jobID, nil)
		assert.Equal(t, http.StatusOK, w.Code)

		var statusResponse map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &statusResponse)
		require.NoError(t, err)

		assert.Contains(t, statusResponse, "job")
		assert.Contains(t, statusResponse, "status")
	})

	// 2. Test download workflow
	t.Run("DownloadWorkflow", func(t *testing.T) {
		// Queue a download
		downloadData := map[string]interface{}{
			"container_id": 12345,
			"format":       "FLAC",
			"quality":      "16bit/44.1kHz",
		}
		body, _ := json.Marshal(downloadData)

		w := suite.makeAuthenticatedRequest(http.MethodPost, "/api/v1/downloads/queue", body)
		assert.Equal(t, http.StatusCreated, w.Code)

		var queueResponse map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &queueResponse)
		require.NoError(t, err)

		assert.Contains(t, queueResponse, "download_id")
		assert.Contains(t, queueResponse, "queue_position")

		// Check download queue
		w = suite.makeAuthenticatedRequest(http.MethodGet, "/api/v1/downloads/queue", nil)
		assert.Equal(t, http.StatusOK, w.Code)

		var queueStatusResponse map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &queueStatusResponse)
		require.NoError(t, err)

		assert.Contains(t, queueStatusResponse, "queue")
		assert.Contains(t, queueStatusResponse, "stats")

		// Get download stats
		w = suite.makeAuthenticatedRequest(http.MethodGet, "/api/v1/downloads/stats", nil)
		assert.Equal(t, http.StatusOK, w.Code)

		var statsResponse map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &statsResponse)
		require.NoError(t, err)

		assert.Contains(t, statsResponse, "total_downloads")
		assert.Contains(t, statsResponse, "queue_length")
	})

	// 3. Test monitoring workflow
	t.Run("MonitoringWorkflow", func(t *testing.T) {
		// Create a monitor
		monitorData := map[string]interface{}{
			"artist_id": 1,
			"settings": map[string]interface{}{
				"check_frequency": "hourly",
				"notify_new":      true,
			},
		}
		body, _ := json.Marshal(monitorData)

		w := suite.makeAuthenticatedRequest(http.MethodPost, "/api/v1/monitoring/monitors", body)
		assert.Equal(t, http.StatusCreated, w.Code)

		var createResponse map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &createResponse)
		require.NoError(t, err)

		assert.Contains(t, createResponse, "monitor_id")

		// Get monitors
		w = suite.makeAuthenticatedRequest(http.MethodGet, "/api/v1/monitoring/monitors", nil)
		assert.Equal(t, http.StatusOK, w.Code)

		var monitorsResponse map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &monitorsResponse)
		require.NoError(t, err)

		assert.Contains(t, monitorsResponse, "monitors")
		assert.Contains(t, monitorsResponse, "pagination")

		// Run monitor check
		w = suite.makeAuthenticatedRequest(http.MethodPost, "/api/v1/monitoring/check/all", nil)
		assert.Equal(t, http.StatusAccepted, w.Code)

		var checkResponse map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &checkResponse)
		require.NoError(t, err)

		assert.Contains(t, checkResponse, "job_id")
		assert.Contains(t, checkResponse, "message")
	})

	// 4. Test analytics workflow
	t.Run("AnalyticsWorkflow", func(t *testing.T) {
		// Get collection stats
		w := suite.makeAuthenticatedRequest(http.MethodGet, "/api/v1/analytics/collection", nil)
		assert.Equal(t, http.StatusOK, w.Code)

		var collectionResponse map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &collectionResponse)
		require.NoError(t, err)

		assert.Contains(t, collectionResponse, "total_artists")
		assert.Contains(t, collectionResponse, "total_shows")

		// Get dashboard summary
		w = suite.makeAuthenticatedRequest(http.MethodGet, "/api/v1/analytics/summary", nil)
		assert.Equal(t, http.StatusOK, w.Code)

		var summaryResponse map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &summaryResponse)
		require.NoError(t, err)

		assert.Contains(t, summaryResponse, "collection")
		assert.Contains(t, summaryResponse, "downloads")
		assert.Contains(t, summaryResponse, "monitoring")
		assert.Contains(t, summaryResponse, "system")
	})
}

func (suite *IntegrationTestSuite) TestAuthenticationFlow() {
	t := suite.T()

	// Test login
	t.Run("Login", func(t *testing.T) {
		loginData := map[string]string{
			"username": "admin",
			"password": "admin123",
		}
		body, _ := json.Marshal(loginData)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "token")
		assert.Contains(t, response, "user")
	})

	// Test protected route without token
	t.Run("ProtectedRouteWithoutToken", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/catalog/artists", nil)
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	// Test protected route with invalid token
	t.Run("ProtectedRouteWithInvalidToken", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/catalog/artists", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")

		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	// Test logout
	t.Run("Logout", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Contains(t, response, "message")
	})
}

func (suite *IntegrationTestSuite) TestJobManagement() {
	t := suite.T()

	// Create jobs of different types
	catalogJob := suite.jobManager.CreateJob(models.JobTypeCatalogRefresh)
	downloadJob := suite.jobManager.CreateJob(models.JobTypeDownload)
	monitorJob := suite.jobManager.CreateJob(models.JobTypeMonitorCheck)

	// Update job statuses
	suite.jobManager.UpdateJob(downloadJob.ID, func(j *models.Job) {
		j.Status = models.JobStatusRunning
		j.Progress = 50
		j.Message = "Downloading..."
	})
	suite.jobManager.UpdateJob(monitorJob.ID, func(j *models.Job) {
		j.Status = models.JobStatusCompleted
		j.Progress = 100
		j.Message = "Monitor check completed"
	})

	// Test job retrieval
	job1, exists1 := suite.jobManager.GetJob(catalogJob.ID)
	assert.True(t, exists1)
	assert.NotNil(t, job1)

	job2, exists2 := suite.jobManager.GetJob(downloadJob.ID)
	assert.True(t, exists2)
	assert.NotNil(t, job2)

	job3, exists3 := suite.jobManager.GetJob(monitorJob.ID)
	assert.True(t, exists3)
	assert.NotNil(t, job3)

	// Test job listing
	allJobs := suite.jobManager.ListJobs()
	assert.Equal(t, 3, len(allJobs))
}

func (suite *IntegrationTestSuite) TestErrorHandling() {
	t := suite.T()

	// Test invalid JSON
	t.Run("InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString("invalid-json"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+suite.authToken)

		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	// Test missing required fields
	t.Run("MissingFields", func(t *testing.T) {
		data := map[string]interface{}{}
		body, _ := json.Marshal(data)

		w := suite.makeAuthenticatedRequest(http.MethodPost, "/api/v1/downloads/queue", body)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	// Test non-existent resources
	t.Run("NotFound", func(t *testing.T) {
		w := suite.makeAuthenticatedRequest(http.MethodGet, "/api/v1/catalog/artists/99999", nil)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
