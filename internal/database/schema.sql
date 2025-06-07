-- Core location and frequency tables for normalization
CREATE TABLE IF NOT EXISTS locations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    city TEXT,
    state TEXT,
    country TEXT,
    latitude REAL,
    longitude REAL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(city, state, country)
);

CREATE TABLE IF NOT EXISTS frequency_bands (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE,
    min_frequency REAL,
    max_frequency REAL,
    band_type TEXT -- '2m', '70cm', '23cm', etc.
);

-- Repeater sources - track which API provided the data
CREATE TABLE IF NOT EXISTS repeater_sources (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT UNIQUE, -- 'brandmeister', 'hearham', 'repeaterbook'
    base_url TEXT,
    last_sync DATETIME,
    total_records INTEGER
);

-- Unified repeater table combining all sources
CREATE TABLE IF NOT EXISTS repeaters (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    
    -- Core identification
    callsign TEXT NOT NULL,
    source_id INTEGER, -- Which API this came from
    external_id TEXT,  -- Original ID from the source API
    
    -- Location (foreign key to locations table)
    location_id INTEGER,
    
    -- Frequencies
    tx_frequency REAL,
    rx_frequency REAL,
    offset_frequency REAL,
    tone_frequency REAL,
    
    -- Technical details
    mode TEXT, -- 'FM', 'DMR', 'D-STAR', 'System Fusion'
    color_code INTEGER, -- DMR color code
    digital_modes TEXT, -- JSON array of supported digital modes
    
    -- Status and metadata
    operational BOOLEAN DEFAULT true,
    online_status BOOLEAN DEFAULT false,
    last_seen DATETIME,
    
    -- Power and antenna
    power_watts INTEGER,
    antenna_height_agl INTEGER,
    antenna_height_msl INTEGER,
    
    -- Technical specifications
    hardware TEXT,
    firmware TEXT,
    website TEXT,
    description TEXT,
    
    -- Timestamps
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_api_sync DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    FOREIGN KEY (location_id) REFERENCES locations(id),
    FOREIGN KEY (source_id) REFERENCES repeater_sources(id),
    UNIQUE(source_id, external_id) -- Prevent duplicates from same source
);

-- DMR Talkgroups
CREATE TABLE IF NOT EXISTS talkgroups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    
    -- Core identification
    talkgroup_id INTEGER NOT NULL, -- The actual DMR talkgroup number
    name TEXT,
    description TEXT,
    
    -- Network information
    network TEXT, -- 'brandmeister', 'tgif', etc.
    active BOOLEAN DEFAULT true,
    
    -- Timestamps
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(talkgroup_id, network)
);

-- Link repeaters to talkgroups (many-to-many relationship)
CREATE TABLE IF NOT EXISTS repeater_talkgroups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repeater_id INTEGER,
    talkgroup_id INTEGER,
    timeslot INTEGER, -- 1 or 2 for DMR
    static_link BOOLEAN DEFAULT false, -- Always connected vs on-demand
    
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (repeater_id) REFERENCES repeaters(id),
    FOREIGN KEY (talkgroup_id) REFERENCES talkgroups(id),
    UNIQUE(repeater_id, talkgroup_id, timeslot)
);

-- APRS stations (separate table due to different data model)
CREATE TABLE IF NOT EXISTS aprs_stations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    
    -- Core identification
    callsign TEXT NOT NULL UNIQUE,
    
    -- Location
    latitude REAL,
    longitude REAL,
    
    -- Status
    last_seen DATETIME,
    comment TEXT,
    
    -- Timestamps
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_repeaters_callsign ON repeaters(callsign);
CREATE INDEX IF NOT EXISTS idx_repeaters_tx_frequency ON repeaters(tx_frequency);
CREATE INDEX IF NOT EXISTS idx_repeaters_location ON repeaters(location_id);
CREATE INDEX IF NOT EXISTS idx_repeaters_mode ON repeaters(mode);
CREATE INDEX IF NOT EXISTS idx_repeaters_source ON repeaters(source_id);
CREATE INDEX IF NOT EXISTS idx_locations_coords ON locations(latitude, longitude);
CREATE INDEX IF NOT EXISTS idx_talkgroups_number ON talkgroups(talkgroup_id);
CREATE INDEX IF NOT EXISTS idx_aprs_callsign ON aprs_stations(callsign);

-- Insert initial frequency bands
INSERT OR IGNORE INTO frequency_bands (name, min_frequency, max_frequency, band_type) VALUES
('2 meters', 144.0, 148.0, '2m'),
('1.25 meters', 219.0, 225.0, '1.25m'),
('70 centimeters', 420.0, 450.0, '70cm'),
('33 centimeters', 902.0, 928.0, '33cm'),
('23 centimeters', 1240.0, 1300.0, '23cm');

-- Insert initial repeater sources
INSERT OR IGNORE INTO repeater_sources (source_name, base_url) VALUES
('brandmeister', 'https://api.brandmeister.network'),
('hearham', 'https://hearham.com'),
('repeaterbook', 'https://repeaterbook.com'),
('tgif', 'https://tgif.network'),
('aprs', 'https://api.aprs.fi');