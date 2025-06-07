package database

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/unklstewy/digiLogRT/internal/api" // Fixed module path
)

// ...existing code...

// SyncBrandmeisterData imports Brandmeister repeaters into the database (optimized)
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

	// Create a map to cache location IDs and avoid duplicate lookups
	locationCache := make(map[string]int)

	// Prepare statements for efficiency
	locationStmt, err := tx.Prepare(`
        INSERT OR IGNORE INTO locations (city, state, country, latitude, longitude)
        VALUES (?, ?, ?, ?, ?)
    `)
	if err != nil {
		return fmt.Errorf("failed to prepare location statement: %v", err)
	}
	defer locationStmt.Close()

	locationLookupStmt, err := tx.Prepare(`
        SELECT id FROM locations WHERE city = ? AND state = ? AND country = ? AND latitude = ? AND longitude = ?
    `)
	if err != nil {
		return fmt.Errorf("failed to prepare location lookup statement: %v", err)
	}
	defer locationLookupStmt.Close()

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

	// Process in smaller batches to show progress and avoid locks
	batchSize := 100
	totalProcessed := 0

	for i := 0; i < len(repeaters); i += batchSize {
		end := i + batchSize
		if end > len(repeaters) {
			end = len(repeaters)
		}

		batch := repeaters[i:end]
		fmt.Printf("  Processing batch %d-%d of %d repeaters...\n", i+1, end, len(repeaters))

		for _, rep := range batch {
			// Create location cache key
			locationKey := fmt.Sprintf("%s|%s|%s|%.6f|%.6f", rep.City, "", rep.Country, rep.Latitude, rep.Longitude)

			var locationID sql.NullInt64

			// Check cache first
			if cachedID, exists := locationCache[locationKey]; exists {
				locationID.Int64 = int64(cachedID)
				locationID.Valid = true
			} else if rep.City != "" || rep.Country != "" || rep.Latitude != 0 || rep.Longitude != 0 {
				// Insert location
				_, err = locationStmt.Exec(rep.City, "", rep.Country, rep.Latitude, rep.Longitude)
				if err != nil {
					fmt.Printf("Warning: failed to insert location for %s: %v\n", rep.Callsign, err)
				} else {
					// Look up the location ID
					var locID int
					err = locationLookupStmt.QueryRow(rep.City, "", rep.Country, rep.Latitude, rep.Longitude).Scan(&locID)
					if err == nil {
						locationID.Int64 = int64(locID)
						locationID.Valid = true
						locationCache[locationKey] = locID
					}
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

			totalProcessed++
		}

		// Show progress every batch
		fmt.Printf("  ✓ Processed %d/%d repeaters (%.1f%%)\n",
			totalProcessed, len(repeaters),
			float64(totalProcessed)/float64(len(repeaters))*100)
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

// ...existing code...

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

// ...existing code...

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

		// Use the same nullable scanning logic as SearchRepeaters
		var locationID sql.NullInt64
		var txFreq, rxFreq, offsetFreq, toneFreq sql.NullFloat64
		var colorCode sql.NullInt64
		var digitalModes, hardware, firmware, website, description sql.NullString
		var lastSeen sql.NullTime
		var powerWatts, antennaHeightAGL, antennaHeightMSL sql.NullInt64
		var city, state, country sql.NullString
		var lat, lng sql.NullFloat64

		err := rows.Scan(
			&r.ID, &r.Callsign, &r.SourceID, &r.ExternalID, &locationID,
			&txFreq, &rxFreq, &offsetFreq, &toneFreq,
			&r.Mode, &colorCode, &digitalModes, &r.Operational, &r.OnlineStatus,
			&lastSeen, &powerWatts, &antennaHeightAGL, &antennaHeightMSL,
			&hardware, &firmware, &website, &description,
			&r.CreatedAt, &r.UpdatedAt, &r.LastAPISync,
			&city, &state, &country, &lat, &lng,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan repeater: %v", err)
		}

		// ...existing code...

		// Convert nullable fields (complete all conversions from SearchRepeaters)
		if locationID.Valid {
			id := int(locationID.Int64)
			r.LocationID = &id
		}
		if txFreq.Valid {
			r.TxFrequency = &txFreq.Float64
		}
		if rxFreq.Valid {
			r.RxFrequency = &rxFreq.Float64
		}
		if offsetFreq.Valid {
			r.OffsetFrequency = &offsetFreq.Float64
		}
		if toneFreq.Valid {
			r.ToneFrequency = &toneFreq.Float64
		}
		if colorCode.Valid {
			cc := int(colorCode.Int64)
			r.ColorCode = &cc
		}
		if digitalModes.Valid {
			r.DigitalModes = &digitalModes.String
		}
		if lastSeen.Valid {
			r.LastSeen = &lastSeen.Time
		}
		if powerWatts.Valid {
			pw := int(powerWatts.Int64)
			r.PowerWatts = &pw
		}
		if antennaHeightAGL.Valid {
			agl := int(antennaHeightAGL.Int64)
			r.AntennaHeightAGL = &agl
		}
		if antennaHeightMSL.Valid {
			msl := int(antennaHeightMSL.Int64)
			r.AntennaHeightMSL = &msl
		}
		if hardware.Valid {
			r.Hardware = &hardware.String
		}
		if firmware.Valid {
			r.Firmware = &firmware.String
		}
		if website.Valid {
			r.Website = &website.String
		}
		if description.Valid {
			r.Description = &description.String
		}
		if city.Valid {
			r.City = &city.String
		}
		if state.Valid {
			r.State = &state.String
		}
		if country.Valid {
			r.Country = &country.String
		}
		if lat.Valid {
			r.Latitude = &lat.Float64
		}
		if lng.Valid {
			r.Longitude = &lng.Float64
		}

		repeaters = append(repeaters, r)

		// ...existing code...

		repeaters = append(repeaters, r)
	}

	return repeaters, nil
}

// ...existing code...
