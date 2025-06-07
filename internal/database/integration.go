package database

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/unklstewy/digiLog/internal/api"
)

// SyncBrandmeisterData imports Brandmeister repeaters into the database
func (d *Database) SyncBrandmeisterData(repeaters []api.BrandmeisterRepeater) error {
	sourceID, err := d.GetSourceID("brandmeister")
	if err != nil {
		return fmt.Errorf("failed to get Brandmeister source ID: %v", err)
	}

	// Begin transaction for better performance
	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Prepare statements for efficiency
	repeaterStmt, err := tx.Prepare(`
        INSERT OR REPLACE INTO repeaters (
            callsign, source_id, external_id, location_id,
            tx_frequency, rx_frequency, mode, color_code,
            operational, online_status, power_watts, antenna_height_agl,
            hardware, website, description, last_api_sync
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `)
	if err != nil {
		return fmt.Errorf("failed to prepare repeater statement: %v", err)
	}
	defer repeaterStmt.Close()

	fmt.Printf("Syncing %d Brandmeister repeaters to database...\n", len(repeaters))

	for i, rep := range repeaters {
		if i%1000 == 0 {
			fmt.Printf("  Processed %d/%d repeaters...\n", i, len(repeaters))
		}

		// Use UpsertLocation to handle location insertion properly
		var locationID sql.NullInt64
		if rep.City != "" || rep.Country != "" || rep.Latitude != 0 || rep.Longitude != 0 {
			locID, err := d.UpsertLocation(rep.City, "", rep.Country, rep.Latitude, rep.Longitude)
			if err == nil {
				locationID.Int64 = int64(locID)
				locationID.Valid = true
			}
		}

		// Parse frequencies
		var txFreq, rxFreq sql.NullFloat64
		if rep.TxFreq != "" {
			if freq, err := strconv.ParseFloat(rep.TxFreq, 64); err == nil {
				txFreq.Float64 = freq
				txFreq.Valid = true
			}
		}
		if rep.RxFreq != "" {
			if freq, err := strconv.ParseFloat(rep.RxFreq, 64); err == nil {
				rxFreq.Float64 = freq
				rxFreq.Valid = true
			}
		}

		// Determine if online (status > 0 in Brandmeister)
		isOnline := rep.Status > 0

		// Insert repeater
		_, err = repeaterStmt.Exec(
			rep.Callsign,
			sourceID,
			rep.ID,
			locationID,
			txFreq,
			rxFreq,
			"DMR", // Brandmeister is DMR
			rep.ColorCode,
			true, // Assume operational if in database
			isOnline,
			rep.PEP,
			rep.AGL,
			rep.Hardware,
			rep.Website,
			rep.Description,
			time.Now(),
		)
		if err != nil {
			fmt.Printf("Warning: failed to insert repeater %s: %v\n", rep.Callsign, err)
			continue
		}
	}

	// Update source sync time
	_, err = tx.Exec(
		"UPDATE repeater_sources SET last_sync = ?, total_records = ? WHERE source_name = ?",
		time.Now(), len(repeaters), "brandmeister",
	)
	if err != nil {
		return fmt.Errorf("failed to update source sync time: %v", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	fmt.Printf("✓ Successfully synced %d Brandmeister repeaters to database\n", len(repeaters))
	return nil
}

// SyncTGIFData imports TGIF talkgroups into the database
func (d *Database) SyncTGIFData(talkgroups []api.TGIFTalkgroup) error {
	// Begin transaction
	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Prepare statement
	stmt, err := tx.Prepare(`
        INSERT OR REPLACE INTO talkgroups (
            talkgroup_id, name, description, network, active
        ) VALUES (?, ?, ?, ?, ?)
    `)
	if err != nil {
		return fmt.Errorf("failed to prepare talkgroup statement: %v", err)
	}
	defer stmt.Close()

	fmt.Printf("Syncing %d TGIF talkgroups to database...\n", len(talkgroups))

	for _, tg := range talkgroups {
		// Parse talkgroup ID
		tgID, err := strconv.Atoi(tg.ID)
		if err != nil {
			fmt.Printf("Warning: invalid talkgroup ID %s: %v\n", tg.ID, err)
			continue
		}

		_, err = stmt.Exec(tgID, tg.Name, tg.Description, "tgif", true)
		if err != nil {
			fmt.Printf("Warning: failed to insert talkgroup %s: %v\n", tg.ID, err)
			continue
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	fmt.Printf("✓ Successfully synced %d TGIF talkgroups to database\n", len(talkgroups))
	return nil
}

// SyncHearhamData imports hearham repeaters into the database
func (d *Database) SyncHearhamData(repeaters []api.HearhamRepeater) error {
	sourceID, err := d.GetSourceID("hearham")
	if err != nil {
		return fmt.Errorf("failed to get hearham source ID: %v", err)
	}

	// Begin transaction
	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Prepare statements
	locationStmt, err := tx.Prepare(`
        INSERT OR IGNORE INTO locations (city, state, country, latitude, longitude)
        VALUES (?, ?, ?, ?, ?)
    `)
	if err != nil {
		return fmt.Errorf("failed to prepare location statement: %v", err)
	}
	defer locationStmt.Close()

	repeaterStmt, err := tx.Prepare(`
        INSERT OR REPLACE INTO repeaters (
            callsign, source_id, external_id, location_id,
            tx_frequency, rx_frequency, mode, operational, last_api_sync
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
    `)
	if err != nil {
		return fmt.Errorf("failed to prepare repeater statement: %v", err)
	}
	defer repeaterStmt.Close()

	fmt.Printf("Syncing %d hearham repeaters to database...\n", len(repeaters))

	for i, rep := range repeaters {
		if i%1000 == 0 {
			fmt.Printf("  Processed %d/%d repeaters...\n", i, len(repeaters))
		}

		// Insert location
		_, err = locationStmt.Exec(rep.City, "", "", 0, 0) // hearham doesn't have coordinates
		if err != nil {
			fmt.Printf("Warning: failed to insert location for %s: %v\n", rep.Callsign, err)
			continue
		}

		// Get location ID
		var locationID sql.NullInt64
		err = tx.QueryRow(`
			SELECT id FROM locations WHERE city = ? AND state = ? AND country = ?
		`, rep.City, "", "").Scan(&locationID)

		if err != nil {
			fmt.Printf("Warning: failed to get location ID for %s: %v\n", rep.Callsign, err)
		}

		// Parse frequency
		var txFreq sql.NullFloat64
		if rep.Frequency != 0 {
			txFreq.Float64 = float64(rep.Frequency)
			txFreq.Valid = true
		}

		// Insert repeater
		_, err = repeaterStmt.Exec(
			rep.Callsign,
			sourceID,
			rep.Callsign, // Use callsign as external ID for hearham
			locationID,
			txFreq,
			nil, // No RX frequency from hearham
			rep.Mode,
			true, // Assume operational
			time.Now(),
		)
		if err != nil {
			fmt.Printf("Warning: failed to insert repeater %s: %v\n", rep.Callsign, err)
			continue
		}
	}

	// Update source sync time
	_, err = tx.Exec(
		"UPDATE repeater_sources SET last_sync = ?, total_records = ? WHERE source_name = ?",
		time.Now(), len(repeaters), "hearham",
	)
	if err != nil {
		return fmt.Errorf("failed to update source sync time: %v", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	fmt.Printf("✓ Successfully synced %d hearham repeaters to database\n", len(repeaters))
	return nil
}

// GetRepeatersByFrequency finds repeaters near a specific frequency
func (d *Database) GetRepeatersByFrequency(frequency float64, rangeMHz float64, limit int) ([]RepeaterRecord, error) {
	query := `
        SELECT r.id, r.callsign, r.source_id, r.external_id, r.location_id,
               r.tx_frequency, r.rx_frequency, r.offset_frequency, r.tone_frequency,
               r.mode, r.color_code, r.digital_modes, r.operational, r.online_status,
               r.last_seen, r.power_watts, r.antenna_height_agl, r.antenna_height_msl,
               r.hardware, r.firmware, r.website, r.description,
               r.created_at, r.updated_at, r.last_api_sync,
               l.city, l.state, l.country, l.latitude, l.longitude
        FROM repeaters r
        LEFT JOIN locations l ON r.location_id = l.id
        WHERE r.tx_frequency BETWEEN ? AND ?
        ORDER BY ABS(r.tx_frequency - ?) ASC
        LIMIT ?
    `

	minFreq := frequency - rangeMHz
	maxFreq := frequency + rangeMHz

	rows, err := d.db.Query(query, minFreq, maxFreq, frequency, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search by frequency: %v", err)
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
