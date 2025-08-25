package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Open database
	db, err := sql.Open("sqlite3", "data/nugs_api.db?_foreign_keys=on")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// Check shows table schema
	fmt.Println("=== SHOWS TABLE SCHEMA ===")
	rows, err := db.Query("PRAGMA table_info(shows)")
	if err != nil {
		log.Fatal("Failed to get shows schema:", err)
	}
	defer rows.Close()

	fmt.Printf("%-5s %-25s %-10s %-10s %-10s %-5s\n", "CID", "NAME", "TYPE", "NOTNULL", "DEFAULT", "PK")
	fmt.Println(strings.Repeat("-", 70))

	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull int
		var defaultVal sql.NullString
		var pk int

		rows.Scan(&cid, &name, &dataType, &notNull, &defaultVal, &pk)
		defaultStr := "NULL"
		if defaultVal.Valid {
			defaultStr = defaultVal.String
		}
		fmt.Printf("%-5d %-25s %-10s %-10d %-10s %-5d\n", cid, name, dataType, notNull, defaultStr, pk)
	}

	// Check if we have any shows data
	var showCount int
	err = db.QueryRow("SELECT COUNT(*) FROM shows").Scan(&showCount)
	if err != nil {
		log.Fatal("Failed to count shows:", err)
	}
	fmt.Printf("\nTotal shows in database: %d\n", showCount)

	// Check sample show
	if showCount > 0 {
		fmt.Println("\n=== SAMPLE SHOW RECORD ===")
		var id, artistID, containerID int
		var venueName, venueCity, venueState, performanceDate, containerInfo, activeState, createdAt sql.NullString
		var availabilityType sql.NullInt64
		var availabilityTypeStr, performanceDateShort, performanceDateFormatted sql.NullString

		err = db.QueryRow("SELECT id, artist_id, container_id, venue_name, venue_city, venue_state, performance_date, performance_date_short, performance_date_formatted, container_info, availability_type, availability_type_str, active_state, created_at FROM shows LIMIT 1").Scan(
			&id, &artistID, &containerID, &venueName, &venueCity, &venueState,
			&performanceDate, &performanceDateShort, &performanceDateFormatted,
			&containerInfo, &availabilityType, &availabilityTypeStr, &activeState, &createdAt)

		if err != nil {
			log.Fatal("Failed to get sample show:", err)
		}

		fmt.Printf("ID: %d\n", id)
		fmt.Printf("Artist ID: %d\n", artistID)
		fmt.Printf("Container ID: %d\n", containerID)
		fmt.Printf("Venue: %s, %s, %s\n", venueName.String, venueCity.String, venueState.String)
		fmt.Printf("Date: %s (%s)\n", performanceDate.String, performanceDateFormatted.String)
		fmt.Printf("Created: %s\n", createdAt.String)
	}
}
