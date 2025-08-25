package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRouter(db *sql.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	jwtSecret := []byte("test-secret")
	authHandler := NewAuthHandler(db, jwtSecret)

	auth := router.Group("/auth")
	{
		auth.POST("/login", authHandler.Login)
		auth.POST("/logout", authHandler.Logout)
	}

	return router
}

func TestAuthHandler_Login(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	router := setupTestRouter(db)

	tests := []struct {
		name           string
		requestBody    map[string]string
		expectedStatus int
		expectedFields []string
	}{
		{
			name: "valid login",
			requestBody: map[string]string{
				"username": "admin",
				"password": "admin123",
			},
			expectedStatus: http.StatusOK,
			expectedFields: []string{"token", "user"},
		},
		{
			name: "invalid credentials",
			requestBody: map[string]string{
				"username": "admin",
				"password": "wrongpassword",
			},
			expectedStatus: http.StatusUnauthorized,
			expectedFields: []string{"error"},
		},
		{
			name: "missing username",
			requestBody: map[string]string{
				"password": "admin123",
			},
			expectedStatus: http.StatusBadRequest,
			expectedFields: []string{"error"},
		},
		{
			name: "missing password",
			requestBody: map[string]string{
				"username": "admin",
			},
			expectedStatus: http.StatusBadRequest,
			expectedFields: []string{"error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			for _, field := range tt.expectedFields {
				assert.Contains(t, response, field)
			}
		})
	}
}

func TestAuthHandler_Logout(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	router := setupTestRouter(db)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "message")
	assert.Equal(t, "Logged out successfully", response["message"])
}
