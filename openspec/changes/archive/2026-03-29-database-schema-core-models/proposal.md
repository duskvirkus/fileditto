## Why

The entire system depends on a well-designed data layer. Before any scanning, hashing, backup, or compliance logic can be built, the SQLite schema and migration infrastructure must exist. Getting this right first prevents costly data model changes later.

## What Changes

- Introduce the SQLite database with all core tables: `Files`, `Locations`, `Media`, `BlurayMedia`, `DriveMedia`, `Photo`, `Video`
- Implement a self-describing schema version table and forward-only migration runner bundled into the application
- Establish deterministic UUID generation (UUID v5) for `Files` entries, derived from SHA-256 content hash
- Define status/lifecycle flags on `Media` (active, inactive, lost) and `Locations` (unreadable, lost)
- Define the `skip_for_counting` flag on `Locations` for 3-2-1 compliance exclusions

## Capabilities

### New Capabilities

- `database-core`: SQLite database file management — creation, opening, version detection, and the migration runner that can upgrade any historical snapshot to the current schema
- `file-registry`: The `Files`, `Photo`, and `Video` tables and their relationships — stores canonical file records with deterministic UUIDs, content hashes, and type-specific metadata
- `media-registry`: The `Media`, `BlurayMedia`, and `DriveMedia` tables — tracks physical storage media with type-specific attributes and lifecycle status
- `location-registry`: The `Locations` table and `files-locations` junction — maps files to their physical locations on media, with per-location flags for compliance counting and loss tracking

### Modified Capabilities

## Impact

- No existing code is affected (greenfield)
- Python and Go both require SQLite client libraries (`sqlite3` stdlib in Python, `modernc.org/sqlite` or `mattn/go-sqlite3` in Go)
- All subsequent MVP features (scanner, hashing, backup engine, compliance tracker) depend on this schema being stable
