package database

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// Database represents our SQLite database connection and operations
type Database struct {
	db   *sql.DB
	path string
}

// RepeaterRecord represents a unified repeater record in the database
type RepeaterRecord struct {
	ID               int       `db:"id"`
	Callsign         string    `db:"callsign"`
	SourceID         int       `db:"source_id"`
	ExternalID       string    `db:"external_id"`
	LocationID       *int      `db:"location_id"`
	TxFrequency      *float64  `db:"tx_frequency"`
	RxFrequency      *float64  `db:"rx_frequency"`
	Mode             string    `db:"mode"`
	ColorCode        *int      `db:"color_code"`
	Operational      bool      `db:"operational"`
	OnlineStatus     bool      `db:"online_status"`
	PowerWatts       *int      `db:"power_watts"`
	AntennaHeightAGL *int      `db:"antenna_height_agl"`
	Hardware         string    `db:"hardware"`
	Website          string    `db:"website"`
	Description      string    `db:"description"`
	CreatedAt        time.Time `db:"created_at"`
	UpdatedAt        time.Time `db:"updated_at"`
	LastAPISync      time.Time `db:"last_api_sync"`
}

// LocationRecord represents a location in the database
type LocationRecord struct {
	ID        int       `db:"id"`
	City      string    `db:"city"`
	State     string    `db:"state"`
	Country   string    `db:"country"`
	Latitude  float64   `db:"latitude"`
	Longitude float64   `db:"longitude"`
	CreatedAt time.Time `db:"created_at"`
}

// TalkgroupRecord represents a DMR talkgroup
type TalkgroupRecord struct {
	ID          int       `db:"id"`
	TalkgroupID int       `db:"talkgroup_id"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	Network     string    `db:"network"`
	Active      bool      `db:"active"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

// NewDatabase creates a new database instance
func NewDatabase(dbPath string) (*Database, error) {
	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %v", err)
	}

	// Open the database
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	database := &Database{
		db:   db,
		path: dbPath,
	}

	// Initialize the schema
	if err := database.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %v", err)
	}

	return database, nil
}

// initSchema creates the database tables if they don't exist
func (d *Database) initSchema() error {
	// Read the schema file
	schemaPath := filepath.Join("internal", "database", "schema.sql")
	schemaSQL, err := ioutil.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %v", err)
	}

	// Execute the schema
	if _, err := d.db.Exec(string(schemaSQL)); err != nil {
		return fmt.Errorf("failed to execute schema: %v", err)
	}

	return nil
}

// Close closes the database connection
func (d *Database) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

// GetSourceID returns the ID for a given source name
func (d *Database) GetSourceID(sourceName string) (int, error) {
	var id int
	query := "SELECT id FROM repeater_sources WHERE source_name = ?"
	err := d.db.QueryRow(query, sourceName).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to get source ID for %s: %v", sourceName, err)
	}
	return id, nil
}

// UpsertLocation inserts or updates a location record
func (d *Database) UpsertLocation(city, state, country string, lat, lng float64) (int, error) {
	// First, try to find existing location
	var id int
	query := "SELECT id FROM locations WHERE city = ? AND state = ? AND country = ?"
	err := d.db.QueryRow(query, city, state, country).Scan(&id)

	if err == sql.ErrNoRows {
		// Insert new location
		insertQuery := `
            INSERT INTO locations (city, state, country, latitude, longitude)
            VALUES (?, ?, ?, ?, ?)
        `
		result, err := d.db.Exec(insertQuery, city, state, country, lat, lng)
		if err != nil {
			return 0, fmt.Errorf("failed to insert location: %v", err)
		}

		lastID, err := result.LastInsertId()
		if err != nil {
			return 0, fmt.Errorf("failed to get last insert ID: %v", err)
		}

		return int(lastID), nil
	} else if err != nil {
		return 0, fmt.Errorf("failed to query location: %v", err)
	}

	// Update existing location with new coordinates if they're more precise
	updateQuery := `
        UPDATE locations 
        SET latitude = ?, longitude = ? 
        WHERE id = ? AND (latitude = 0 OR longitude = 0 OR ? != 0 AND ? != 0)
    `
	_, err = d.db.Exec(updateQuery, lat, lng, id, lat, lng)
	if err != nil {
		return 0, fmt.Errorf("failed to update location: %v", err)
	}

	return id, nil
}

// SearchRepeaters performs a complex search across all repeater data
func (d *Database) SearchRepeaters(query string, limit int) ([]RepeaterRecord, error) {
	sqlQuery := `
        SELECT r.id, r.callsign, r.source_id, r.external_id, r.location_id,
               r.tx_frequency, r.rx_frequency, r.offset_frequency, r.tone_frequency,
               r.mode, r.color_code, r.digital_modes, r.operational, r.online_status,
               r.last_seen, r.power_watts, r.antenna_height_agl, r.antenna_height_msl,
               r.hardware, r.firmware, r.website, r.description,
               r.created_at, r.updated_at, r.last_api_sync,
               l.city, l.state, l.country, l.latitude, l.longitude
        FROM repeaters r
        LEFT JOIN locations l ON r.location_id = l.id
        WHERE r.callsign LIKE ? 
           OR l.city LIKE ?
           OR l.state LIKE ?
           OR l.country LIKE ?
           OR r.description LIKE ?
        ORDER BY r.callsign
        LIMIT ?
    `

	searchTerm := "%" + query + "%"
	rows, err := d.db.Query(sqlQuery, searchTerm, searchTerm, searchTerm, searchTerm, searchTerm, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search repeaters: %v", err)
	}
	defer rows.Close()

	var repeaters []RepeaterRecord
	for rows.Next() {
		var r RepeaterRecord
		var city, state, country sql.NullString
		var lat, lng sql.NullFloat64
		var offsetFreq, toneFreq sql.NullFloat64
		var digitalModes, firmware sql.NullString
		var lastSeen sql.NullTime
		var antennaHeightMSL sql.NullInt64

		err := rows.Scan(
			&r.ID, &r.Callsign, &r.SourceID, &r.ExternalID, &r.LocationID,
			&r.TxFrequency, &r.RxFrequency, &offsetFreq, &toneFreq,
			&r.Mode, &r.ColorCode, &digitalModes, &r.Operational, &r.OnlineStatus,
			&lastSeen, &r.PowerWatts, &r.AntennaHeightAGL, &antennaHeightMSL,
			&r.Hardware, &firmware, &r.Website, &r.Description,
			&r.CreatedAt, &r.UpdatedAt, &r.LastAPISync,
			&city, &state, &country, &lat, &lng,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan repeater: %v", err)
		}

		repeaters = append(repeaters, r)
	}

	return repeaters, nil
}

// GetRepeaterStats returns statistics about the repeater database
func (d *Database) GetRepeaterStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total repeaters
	var total int
	err := d.db.QueryRow("SELECT COUNT(*) FROM repeaters").Scan(&total)
	if err != nil {
		return nil, err
	}
	stats["total_repeaters"] = total

	// By source
	sourceQuery := `
        SELECT rs.source_name, COUNT(r.id) 
        FROM repeater_sources rs 
        LEFT JOIN repeaters r ON rs.id = r.source_id 
        GROUP BY rs.id, rs.source_name
    `
	rows, err := d.db.Query(sourceQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sources := make(map[string]int)
	for rows.Next() {
		var source string
		var count int
		if err := rows.Scan(&source, &count); err != nil {
			return nil, err
		}
		sources[source] = count
	}
	stats["by_source"] = sources

	// Online status
	var online int
	err = d.db.QueryRow("SELECT COUNT(*) FROM repeaters WHERE online_status = true").Scan(&online)
	if err != nil {
		return nil, err
	}
	stats["online_repeaters"] = online

	return stats, nil
}
