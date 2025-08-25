package database

import (
	"database/sql"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the database connection and provides utility methods
type DB struct {
	*sql.DB
}

// Initialize creates and initializes the database connection
func Initialize(databaseURL string) (*sql.DB, error) {
	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(databaseURL), 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %v", err)
	}

	// Open database connection
	db, err := sql.Open("sqlite3", databaseURL+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	// Run migrations
	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %v", err)
	}

	log.Printf("Database initialized successfully at: %s", databaseURL)
	return db, nil
}

// runMigrations executes all migration files
func runMigrations(db *sql.DB) error {
	// Create migrations table if it doesn't exist
	createMigrationsTable := `
	CREATE TABLE IF NOT EXISTS migrations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		filename TEXT UNIQUE NOT NULL,
		executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := db.Exec(createMigrationsTable); err != nil {
		return fmt.Errorf("failed to create migrations table: %v", err)
	}

	// Get list of executed migrations
	executedMigrations, err := getExecutedMigrations(db)
	if err != nil {
		return err
	}

	// Get migration files
	migrationFiles, err := getMigrationFiles()
	if err != nil {
		return err
	}
	log.Printf("Found %d migration files: %v", len(migrationFiles), migrationFiles)

	// Execute pending migrations
	for _, filename := range migrationFiles {
		if _, executed := executedMigrations[filename]; executed {
			continue
		}

		log.Printf("Running migration: %s", filename)
		if err := executeMigration(db, filename); err != nil {
			return fmt.Errorf("failed to execute migration %s: %v", filename, err)
		}

		// Record migration as executed
		if err := recordMigration(db, filename); err != nil {
			return fmt.Errorf("failed to record migration %s: %v", filename, err)
		}
	}

	return nil
}

// getExecutedMigrations returns a map of executed migrations
func getExecutedMigrations(db *sql.DB) (map[string]bool, error) {
	rows, err := db.Query("SELECT filename FROM migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	executed := make(map[string]bool)
	for rows.Next() {
		var filename string
		if err := rows.Scan(&filename); err != nil {
			return nil, err
		}
		executed[filename] = true
	}

	return executed, rows.Err()
}

// getMigrationFiles returns sorted list of migration files
func getMigrationFiles() ([]string, error) {
	// Try different possible paths based on where the code is being executed
	possiblePaths := []string{
		"internal/database/migrations",          // from project root
		"../../../internal/database/migrations", // from test directories
		"../../database/migrations",             // from internal/api/handlers
		"../internal/database/migrations",       // from test/ directory
		"database/migrations",                   // from internal/
		"./migrations",                          // from internal/database/
	}

	var migrationsDir string
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			migrationsDir = path
			break
		}
	}

	if migrationsDir == "" {
		// Last resort: try to find migrations directory recursively
		wd, _ := os.Getwd()
		log.Printf("Working directory: %s", wd)
		return nil, fmt.Errorf("could not find migrations directory in any of the expected paths: %v", possiblePaths)
	}

	log.Printf("Found migrations directory at: %s", migrationsDir)

	var files []string
	err := filepath.WalkDir(migrationsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".sql") {
			files = append(files, filepath.Base(path))
		}
		return nil
	})

	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	sort.Strings(files)
	return files, nil
}

// executeMigration executes a single migration file
func executeMigration(db *sql.DB, filename string) error {
	// Find the migrations directory using the same logic as getMigrationFiles
	possiblePaths := []string{
		"internal/database/migrations",          // from project root
		"../../../internal/database/migrations", // from test directories
		"../../database/migrations",             // from internal/api/handlers
		"../internal/database/migrations",       // from test/ directory
		"database/migrations",                   // from internal/
		"./migrations",                          // from internal/database/
	}

	var migrationsDir string
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			migrationsDir = path
			break
		}
	}

	if migrationsDir == "" {
		return fmt.Errorf("could not find migrations directory")
	}

	migrationPath := filepath.Join(migrationsDir, filename)

	content, err := os.ReadFile(migrationPath)
	if err != nil {
		return err
	}

	// Split by semicolon and execute each statement
	statements := strings.Split(string(content), ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		// Skip empty statements
		if stmt == "" {
			continue
		}

		// Remove comments from within statements
		lines := strings.Split(stmt, "\n")
		var cleanLines []string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "--") {
				cleanLines = append(cleanLines, line)
			}
		}

		if len(cleanLines) == 0 {
			continue
		}

		cleanStmt := strings.Join(cleanLines, "\n")
		if _, err := db.Exec(cleanStmt); err != nil {
			return fmt.Errorf("failed to execute statement '%s': %v", cleanStmt, err)
		}
	}

	return nil
}

// recordMigration records a migration as executed
func recordMigration(db *sql.DB, filename string) error {
	_, err := db.Exec("INSERT INTO migrations (filename) VALUES (?)", filename)
	return err
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}
