-- Migration 0001: Initial schema
-- Creates all core tables for the fileditto database.

-- schema_version: single-row table identifying the current schema version.
CREATE TABLE schema_version (
    version INTEGER NOT NULL
);
INSERT INTO schema_version (version) VALUES (1);

-- schema_migrations: records every applied migration with checksum for integrity checks.
CREATE TABLE schema_migrations (
    migration_number INTEGER PRIMARY KEY,
    applied_at       TEXT    NOT NULL,  -- ISO-8601 UTC timestamp
    checksum         TEXT    NOT NULL   -- SHA-256 hex of the migration file content
);

-- schema_metadata: key/value store for database-level configuration and sanity checks.
CREATE TABLE schema_metadata (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
INSERT INTO schema_metadata (key, value)
    VALUES ('uuid_namespace', 'a8b4c6d2-e1f9-4a5b-88c7-d3e9f2a1b4c6');

-- Files: canonical record for each unique file, identified by SHA-256 content hash.
CREATE TABLE Files (
    id         TEXT    PRIMARY KEY,                 -- UUID v5 derived from sha256
    sha256     TEXT    NOT NULL UNIQUE,
    size_bytes INTEGER NOT NULL,
    file_type  TEXT    NOT NULL CHECK(file_type IN ('generic', 'photo', 'video')),
    created_at TEXT,                                -- ISO-8601 UTC
    updated_at TEXT                                 -- ISO-8601 UTC
);

-- Photo: photo-specific metadata, shares PK with Files.
CREATE TABLE Photo (
    id            TEXT PRIMARY KEY REFERENCES Files(id) ON DELETE CASCADE,
    width_px      INTEGER,
    height_px     INTEGER,
    color_profile TEXT,
    metadata_blob BLOB
);

-- Video: video-specific metadata, shares PK with Files.
CREATE TABLE Video (
    id               TEXT PRIMARY KEY REFERENCES Files(id) ON DELETE CASCADE,
    duration_seconds REAL,
    width_px         INTEGER,
    height_px        INTEGER,
    frame_rate       REAL,
    codec            TEXT,
    metadata_blob    BLOB
);

-- PhysicalLocations: named physical locations where media is stored (shelf, cabinet, off-site, etc.).
CREATE TABLE PhysicalLocations (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    notes      TEXT,
    created_at TEXT,                   -- ISO-8601 UTC
    updated_at TEXT                    -- ISO-8601 UTC
);

-- Media: tracks physical storage media. BlurayMedia and DriveMedia share this PK.
CREATE TABLE Media (
    id                   TEXT    NOT NULL PRIMARY KEY,
    media_type           TEXT    NOT NULL CHECK(media_type IN ('bluray', 'drive')),
    label                TEXT,
    status               TEXT    NOT NULL CHECK(status IN ('connected', 'disconnected', 'lost', 'damaged')),
    device_id            INTEGER REFERENCES Devices(id),  -- set when the media is connected to a device
    physical_location_id TEXT    REFERENCES PhysicalLocations(id),
    created_at           TEXT,         -- ISO-8601 UTC
    updated_at           TEXT          -- ISO-8601 UTC
);

-- BlurayMedia: Blu-ray disc specific attributes. Shares PK with Media.
CREATE TABLE BlurayMedia (
    id              TEXT    PRIMARY KEY REFERENCES Media(id) ON DELETE CASCADE,
    disc_label      TEXT,
    capacity_gb     REAL,
    burn_date       TEXT,               -- ISO-8601 date
    disc_set_number INTEGER,
    verified        INTEGER NOT NULL DEFAULT 0  -- boolean: 0=false, 1=true
);

-- DriveMedia: external drive specific attributes. Shares PK with Media.
CREATE TABLE DriveMedia (
    id             TEXT PRIMARY KEY REFERENCES Media(id) ON DELETE CASCADE,
    serial_number  TEXT UNIQUE,
    make           TEXT,
    model          TEXT,
    capacity_gb    REAL,
    interface_type TEXT,
    acquired_date  TEXT                -- ISO-8601 date
);

-- Locations: maps a file copy to a physical location on a media item.
CREATE TABLE Locations (
    id                TEXT    PRIMARY KEY,
    file_id           TEXT    NOT NULL REFERENCES Files(id) ON DELETE CASCADE,
    media_id          TEXT    NOT NULL REFERENCES Media(id) ON DELETE CASCADE,
    path_on_media     TEXT    NOT NULL,
    skip_for_counting INTEGER NOT NULL DEFAULT 0,  -- boolean: 1 = exclude from 3-2-1 count
    status            TEXT    NOT NULL CHECK(status IN ('healthy', 'error')),
    created_at        TEXT,                        -- ISO-8601 UTC
    updated_at        TEXT                         -- ISO-8601 UTC
);

-- Devices: known source devices that can contribute files to the ingestion queue.
CREATE TABLE Devices (
    id          INTEGER PRIMARY KEY,
    name        TEXT    NOT NULL UNIQUE,
    created_at  TEXT    NOT NULL,
    updated_at  TEXT    NOT NULL
);

-- IngestionQueue: holds file paths discovered during scanning, pending ingestion.
CREATE TABLE IngestionQueue (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    file_path     TEXT    NOT NULL,
    device_id     INTEGER NOT NULL REFERENCES Devices(id),
    status        TEXT    NOT NULL DEFAULT 'pending'
                          CHECK(status IN ('pending', 'processing', 'done', 'failed', 'unsupported')),
    attempt_count INTEGER,
    error         TEXT,
    created_at    TEXT    NOT NULL,
    updated_at    TEXT    NOT NULL,
    UNIQUE(file_path, device_id)
);
