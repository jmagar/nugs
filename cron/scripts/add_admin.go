package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Open database
	db, err := sql.Open("sqlite3", "data/nugs_api.db?_foreign_keys=on")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	// Hash the password
	password := "password"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("Failed to hash password:", err)
	}

	// Insert admin user
	_, err = db.Exec(`
		INSERT OR REPLACE INTO users (id, username, email, password_hash, role, api_key, active, created_at, updated_at) 
		VALUES (1, 'admin', 'admin@nugs.local', ?, 'admin', 'nugs_api_key_admin_change_me', true, datetime('now'), datetime('now'))
	`, string(hashedPassword))

	if err != nil {
		log.Fatal("Failed to insert admin user:", err)
	}

	fmt.Println("Admin user created successfully!")
	fmt.Println("Username: admin")
	fmt.Println("Password: password")
}
