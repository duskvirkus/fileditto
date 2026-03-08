## Context

This is a greenfield project. There is no existing database or schema to migrate from. The design must serve as the stable foundation for all subsequent MVP features (scanner, hashing, backup engine, compliance tracker, Blu-ray burn). Both Python and Go components will read and write to this database, so the schema and access patterns must work well from both languages.

The database file itself will be copied to backup media repeatedly over the life of the system, so compactness and long-term format stability are hard constraints.

## Goals / Non-Goals

**Goals:**
- Define all core tables with correct relationships and constraints
- Implement a self-describing version table so any database snapshot can identify its own schema version
- Implement a forward-only migration runner bundled into the application binary
- Establish deterministic UUID v5 generation for `Files` entries
- Define all lifecycle/status flags needed by the compliance tracker and lifecycle manager
- Provide a stable, tested data layer that all other features can build on

**Non-Goals:**
- ORM or query builder — raw SQL with parameterized queries only
- Async/concurrent write support — single writer at a time (SQLite WAL mode handles concurrent reads)
- Full-text search or indexing beyond what correctness requires
- Any application logic above the data layer (scanning, hashing, compliance rules belong in later phases)

## Decisions

### SQLite with WAL mode
SQLite in WAL (Write-Ahead Logging) mode allows concurrent reads while a write is in progress. This matters because the Go engine and Python scanner may query the database simultaneously during a backup run. WAL mode also produces slightly better write performance for our access pattern (bulk inserts during scans, reads during compliance checks).

Alternatives considered: PostgreSQL (requires a server process, too heavy for a portable backup tool), DuckDB (excellent analytics but less battle-tested for long-term archival file formats).

### UUID v5 for File identity
File UUIDs are generated with UUID v5 using a fixed project namespace UUID and the file's SHA-256 content hash as the name input. This produces a deterministic, collision-resistant identifier. The same file ingested on two different machines — or after a database rebuild from an old snapshot — produces the same UUID, enabling correct deduplication without coordination.

The UUID is assigned once and never changes, even if the hashing strategy evolves. If we later add xxHash or a second hash column, the UUID remains stable and all foreign key relationships are preserved.

Alternatives considered: random UUID v4 (loses determinism, breaks rebuild scenario), using the hash directly as a primary key (ties the PK to the hash algorithm, complicates future changes).

### Single migration runner in Go, shared schema definition
Migrations are numbered SQL files embedded in the Go binary at compile time. On startup, the runner checks the `schema_version` table and applies any outstanding migrations in order. Python reads the current schema but never runs migrations — Go owns schema lifecycle. This avoids dual-language migration conflicts.

Each migration file is append-only and immutable once released. Migration history is stored in the `schema_migrations` table (migration number, applied timestamp, checksum) so the database is always self-auditing.

### Inheritance via foreign key, not polymorphic columns
`Media` has a `media_type` discriminator column and a `subtype_id` foreign key pointing to either `BlurayMedia` or `DriveMedia`. This is the explicit table-per-type pattern. It avoids nullable columns or JSON blobs for type-specific fields, keeps type-specific queries simple, and is easy to extend with new media types.

Alternatives considered: single table with nullable columns (schema becomes messy as types grow), JSON blob for subtype data (loses column-level constraints and indexing).

### BLOB for open-ended metadata, typed columns for queryable fields
Fields that need to be queried or indexed (resolution, duration, hash, file_type, status flags) are typed columns. Open-ended metadata that we want to store but not query against (EXIF tags, camera model, GPS, labels) is stored as a compressed BLOB. This keeps the database file smaller — important since it is backed up many times across many media. The TUI will include a metadata decoder to make this human-inspectable when needed.

### Separate Photo and Video tables
`Photo` and `Video` are separate tables, each sharing the same UUID primary key as their corresponding `Files` entry. Photo-specific fields (resolution, color profile) and video-specific fields (duration, frame rate, codec) don't overlap, so a single table would require many nullable columns. Separate tables keep the schema clean and queries simple. A `JOIN` to the appropriate subtype table is always cheap since it's a PK lookup.

## Risks / Trade-offs

[Risk] Schema mistakes are expensive — all later features depend on this layer → Mitigation: write integration tests that exercise every table and relationship before any other feature begins. Add a schema linter step to CI.

[Risk] SQLite concurrent access from Python and Go processes → Mitigation: WAL mode + short write transactions + Go owns all writes during a backup run. Python is read-only during active backup jobs.

[Risk] UUID v5 namespace collision if namespace UUID is ever changed → Mitigation: hardcode the namespace UUID as a named constant in both Python and Go, document it explicitly, never change it. Store it in the `schema_metadata` table as a sanity check.

[Risk] Migration runner divergence if Python ever runs migrations → Mitigation: Python DB access layer asserts that `schema_version` matches the expected version at startup and raises a hard error if not. Only Go migrates.

## Migration Plan

This is the initial schema — no prior state to migrate from. The migration runner itself is implemented here and will process all future migrations.

Deployment:
1. Application starts, checks for `schema_version` table
2. If absent, runs migration 0001 (creates all tables including `schema_version` and `schema_migrations`)
3. Subsequent startups check current version and apply any pending migrations in order

Rollback: not supported for forward-only migrations. Rollback strategy is restoring from a pre-migration database snapshot (covered by P7 — Database backup with versioning).

## Open Questions

- What is the project namespace UUID for UUID v5 generation? Needs to be chosen and hardcoded before first use. Defer to the `database-core` spec for the exact value.
