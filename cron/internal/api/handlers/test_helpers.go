package handlers

import (
	"database/sql"
	"testing"

	"github.com/jmagar/nugs/cron/internal/database"
	"github.com/stretchr/testify/require"
)

// setupTestDB initializes an in-memory database for testing
func setupTestDB(t *testing.T) *sql.DB {
	db, err := database.Initialize(":memory:")
	require.NoError(t, err)
	return db
}
