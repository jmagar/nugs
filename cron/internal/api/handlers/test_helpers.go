package handlers

import (
	"database/sql"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jmagar/nugs/cron/internal/database"
	"github.com/jmagar/nugs/cron/internal/models"
	"github.com/stretchr/testify/require"
)

// setupTestDB initializes an in-memory database for testing
func setupTestDB(t *testing.T) *sql.DB {
	db, err := database.Initialize(":memory:")
	require.NoError(t, err)
	return db
}

// setupTestJobManager creates a new job manager for testing
func setupTestJobManager() *models.JobManager {
	return models.NewJobManager()
}

// createTestUser creates a test user in the database
func createTestUser(t *testing.T, db *sql.DB, username, email, role string) int64 {
	query := `INSERT INTO users (username, email, password_hash, role, active, created_at, updated_at) 
	          VALUES (?, ?, 'test-hash', ?, true, datetime('now'), datetime('now'))`

	result, err := db.Exec(query, username, email, role)
	require.NoError(t, err)

	id, err := result.LastInsertId()
	require.NoError(t, err)

	return id
}

// setupGinTestMode sets gin to test mode
func setupGinTestMode() {
	gin.SetMode(gin.TestMode)
}
