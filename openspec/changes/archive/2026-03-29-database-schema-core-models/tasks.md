## 1. Project Structure and Dependencies

- [x] 1.1 Create the `internal/db` package directory in the Go module
- [x] 1.2 Add `modernc.org/sqlite` (or `mattn/go-sqlite3`) to `go.mod` and `go.sum`
- [x] 1.3 Create `internal/db/migrations/` directory for embedded SQL migration files
- [x] 1.4 Add `db/` package to the Python project with `__init__.py` and `connection.py`

## 2. Initial Migration SQL

- [x] 2.1 Write migration `0001_initial_schema.sql`: create `schema_version` table (single-row, version INTEGER)
- [x] 2.2 Extend `0001`: create `schema_migrations` table (migration_number, applied_at, checksum)
- [x] 2.3 Extend `0001`: create `schema_metadata` table (key TEXT PK, value TEXT) and insert the hardcoded project namespace UUID row
- [x] 2.4 Extend `0001`: create `Files` table with all columns, unique constraint on `sha256`, CHECK on `file_type`
- [x] 2.5 Extend `0001`: create `Photo` table with FK → `Files.id` ON DELETE CASCADE
- [x] 2.6 Extend `0001`: create `Video` table with FK → `Files.id` ON DELETE CASCADE
- [x] 2.7 Extend `0001`: create `BlurayMedia` table with all columns
- [x] 2.8 Extend `0001`: create `DriveMedia` table with unique constraint on `serial_number`
- [x] 2.9 Extend `0001`: create `Media` table with CHECK on `media_type` and `status`, FK `subtype_id` references handled at application layer
- [x] 2.10 Extend `0001`: create `Locations` table with FK → `Files.id` ON DELETE CASCADE, FK → `Media.id` ON DELETE CASCADE, CHECK on `status`, default `skip_for_counting = 0`

## 3. Go Migration Runner

- [x] 3.1 Embed migration files into the Go binary using `//go:embed migrations/*.sql`
- [x] 3.2 Implement `OpenDB(path string) (*sql.DB, error)` — opens/creates the SQLite file and sets `PRAGMA journal_mode=WAL`
- [x] 3.3 Implement `RunMigrations(db *sql.DB) error` — reads `schema_version`, applies pending migrations in order, updates `schema_version` and inserts into `schema_migrations` per migration
- [x] 3.4 Implement checksum verification: compute SHA-256 of each migration file and compare against stored `schema_migrations.checksum`; halt on mismatch
- [x] 3.5 Implement namespace UUID sanity check: after migrations, read `schema_metadata` and compare against compiled-in constant; halt on mismatch

## 4. UUID v5 Generation

- [x] 4.1 Define the project namespace UUID as a named constant in Go (`internal/db/uuid.go`)
- [x] 4.2 Implement `FileUUID(sha256hex string) (uuid.UUID, error)` in Go using UUID v5 with the project namespace
- [x] 4.3 Define the same project namespace UUID as a constant in Python (`db/uuid_util.py`)
- [x] 4.4 Implement `file_uuid(sha256_hex: str) -> str` in Python using `uuid.uuid5` with the project namespace

## 5. Python DB Access Layer

- [x] 5.1 Implement `open_db(path: str) -> sqlite3.Connection` in Python — opens the database with WAL mode enabled
- [x] 5.2 Implement schema version assertion in `open_db`: read `schema_version` and raise `RuntimeError` if it does not match the compiled-in expected version constant
- [x] 5.3 Implement namespace UUID assertion: read `schema_metadata` and raise `RuntimeError` if `uuid_namespace` does not match the Python constant

## 6. Integration Tests

- [x] 6.1 Write Go integration test: open a fresh in-memory (or temp file) DB, run migrations, assert `schema_version = 1`
- [x] 6.2 Write Go integration test: insert a `Files` row, insert a `Photo` row, delete the `Files` row, assert `Photo` row is gone (CASCADE)
- [x] 6.3 Write Go integration test: insert a `Files` row, insert a `Video` row, delete the `Files` row, assert `Video` row is gone (CASCADE)
- [x] 6.4 Write Go integration test: insert `Media` and `Locations` rows, delete the `Media` row, assert `Locations` row is gone (CASCADE)
- [x] 6.5 Write Go integration test: attempt insert with invalid `file_type`, assert constraint violation
- [x] 6.6 Write Go integration test: attempt insert with invalid `Media.status`, assert constraint violation
- [x] 6.7 Write Go integration test: attempt insert with invalid `Locations.status`, assert constraint violation
- [x] 6.8 Write Go integration test: generate UUID v5 twice for same SHA-256 hash, assert identical results
- [x] 6.9 Write Python integration test: open DB, assert version check passes when version matches
- [x] 6.10 Write Python integration test: assert Python and Go produce identical UUID v5 for a known SHA-256 hash
- [x] 6.11 Write Go integration test: simulate checksum mismatch in `schema_migrations`, assert runner halts with error
