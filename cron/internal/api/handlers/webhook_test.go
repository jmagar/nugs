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

func setupWebhookTestRouter(t *testing.T) (*gin.Engine, *models.JobManager) {
	db := setupTestDB(t)
	jobManager := models.NewJobManager()

	gin.SetMode(gin.TestMode)
	router := gin.New()

	webhookHandler := NewWebhookHandler(db, jobManager)

	webhooks := router.Group("/webhooks")
	{
		webhooks.POST("/", webhookHandler.CreateWebhook)
		webhooks.GET("/", webhookHandler.GetWebhooks)
		webhooks.GET("/:id", webhookHandler.GetWebhook)
		webhooks.PUT("/:id", webhookHandler.UpdateWebhook)
		webhooks.DELETE("/:id", webhookHandler.DeleteWebhook)
		webhooks.POST("/:id/test", webhookHandler.TestWebhook)
		webhooks.GET("/:id/deliveries", webhookHandler.GetWebhookDeliveries)
		webhooks.GET("/deliveries", webhookHandler.GetAllDeliveries)
		webhooks.GET("/events", webhookHandler.GetAvailableEvents)
		webhooks.GET("/stats", webhookHandler.GetWebhookStats)
	}

	return router, jobManager
}

func TestWebhookHandler_CreateWebhook(t *testing.T) {
	router, _ := setupWebhookTestRouter(t)

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		checkFields    []string
	}{
		{
			name: "create valid webhook",
			requestBody: map[string]interface{}{
				"url": "https://example.com/webhook",
				"events": []string{
					"download.completed",
					"monitor.alert",
				},
				"secret": "webhook-secret",
			},
			expectedStatus: http.StatusCreated,
			checkFields:    []string{"webhook_id", "message"},
		},
		{
			name: "create webhook with custom headers",
			requestBody: map[string]interface{}{
				"url": "https://api.example.com/webhooks",
				"events": []string{
					"catalog.refreshed",
				},
				"secret": "my-secret",
				"headers": map[string]string{
					"Authorization": "Bearer token",
					"X-Custom":      "value",
				},
			},
			expectedStatus: http.StatusCreated,
			checkFields:    []string{"webhook_id", "message"},
		},
		{
			name: "missing url",
			requestBody: map[string]interface{}{
				"events": []string{"download.completed"},
			},
			expectedStatus: http.StatusBadRequest,
			checkFields:    []string{"error"},
		},
		{
			name: "invalid url",
			requestBody: map[string]interface{}{
				"url":    "not-a-url",
				"events": []string{"download.completed"},
			},
			expectedStatus: http.StatusBadRequest,
			checkFields:    []string{"error"},
		},
		{
			name: "missing events",
			requestBody: map[string]interface{}{
				"url": "https://example.com/webhook",
			},
			expectedStatus: http.StatusBadRequest,
			checkFields:    []string{"error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/webhooks/", bytes.NewBuffer(body))
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

func TestWebhookHandler_GetWebhooks(t *testing.T) {
	router, _ := setupWebhookTestRouter(t)

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		checkFields    []string
	}{
		{
			name:           "get all webhooks",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			checkFields:    []string{"webhooks", "pagination"},
		},
		{
			name:           "get webhooks with pagination",
			queryParams:    "?page=1&page_size=10",
			expectedStatus: http.StatusOK,
			checkFields:    []string{"webhooks", "pagination"},
		},
		{
			name:           "filter by active status",
			queryParams:    "?active=true",
			expectedStatus: http.StatusOK,
			checkFields:    []string{"webhooks", "pagination"},
		},
		{
			name:           "filter by event type",
			queryParams:    "?event=download.completed",
			expectedStatus: http.StatusOK,
			checkFields:    []string{"webhooks", "pagination"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/webhooks/"+tt.queryParams, nil)
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

func TestWebhookHandler_UpdateWebhook(t *testing.T) {
	router, _ := setupWebhookTestRouter(t)

	tests := []struct {
		name           string
		webhookID      string
		requestBody    map[string]interface{}
		expectedStatus int
		checkError     bool
	}{
		{
			name:      "update webhook url",
			webhookID: "1",
			requestBody: map[string]interface{}{
				"url": "https://new-url.com/webhook",
			},
			expectedStatus: http.StatusOK,
			checkError:     false,
		},
		{
			name:      "update webhook events",
			webhookID: "1",
			requestBody: map[string]interface{}{
				"events": []string{
					"download.completed",
					"catalog.refreshed",
				},
			},
			expectedStatus: http.StatusOK,
			checkError:     false,
		},
		{
			name:      "update non-existent webhook",
			webhookID: "99999",
			requestBody: map[string]interface{}{
				"url": "https://example.com/webhook",
			},
			expectedStatus: http.StatusNotFound,
			checkError:     true,
		},
		{
			name:      "invalid webhook ID",
			webhookID: "invalid",
			requestBody: map[string]interface{}{
				"url": "https://example.com/webhook",
			},
			expectedStatus: http.StatusBadRequest,
			checkError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPut, "/webhooks/"+tt.webhookID, bytes.NewBuffer(body))
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

func TestWebhookHandler_TestWebhook(t *testing.T) {
	router, _ := setupWebhookTestRouter(t)

	tests := []struct {
		name           string
		webhookID      string
		requestBody    map[string]interface{}
		expectedStatus int
		checkError     bool
	}{
		{
			name:      "test webhook with custom payload",
			webhookID: "1",
			requestBody: map[string]interface{}{
				"event":   "download.completed",
				"payload": map[string]interface{}{"test": "data"},
			},
			expectedStatus: http.StatusOK,
			checkError:     false,
		},
		{
			name:           "test webhook with default payload",
			webhookID:      "1",
			requestBody:    map[string]interface{}{},
			expectedStatus: http.StatusOK,
			checkError:     false,
		},
		{
			name:           "test non-existent webhook",
			webhookID:      "99999",
			requestBody:    map[string]interface{}{},
			expectedStatus: http.StatusNotFound,
			checkError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/webhooks/"+tt.webhookID+"/test", bytes.NewBuffer(body))
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
				assert.Contains(t, response, "delivery_id")
			}
		})
	}
}

func TestWebhookHandler_GetAvailableEvents(t *testing.T) {
	router, _ := setupWebhookTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/webhooks/events", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "events")

	if events, ok := response["events"].([]interface{}); ok {
		assert.Greater(t, len(events), 0)

		// Check that each event has required fields
		if len(events) > 0 {
			if event, ok := events[0].(map[string]interface{}); ok {
				assert.Contains(t, event, "name")
				assert.Contains(t, event, "description")
			}
		}
	}
}

func TestWebhookHandler_GetWebhookStats(t *testing.T) {
	router, _ := setupWebhookTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/webhooks/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	expectedFields := []string{
		"total_webhooks", "active_webhooks", "total_deliveries",
		"successful_deliveries", "failed_deliveries", "average_response_time",
		"most_triggered_events", "recent_deliveries",
	}

	for _, field := range expectedFields {
		assert.Contains(t, response, field)
	}
}
