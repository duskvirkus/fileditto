# Multi-database support

## Goal

Allow fileditto to run against either SQLite or PostgreSQL, selected at runtime via `fileditto db configure`. SQLite remains the default for local use; PostgreSQL supports shared/server deployments.

## Package structure

```
internal/db/
    db.go              — DB interface, repository interfaces, domain types, Dialect type
    migrate.go         — dialect-aware migration runner
    uuid.go            — unchanged UUID helpers
    migrations/
        shared/
            0001_initial_schema.sql
        sqlite/        — empty for now, structure unit tested
        postgres/      — empty for now, structure unit tested
    sqlite/
        db.go          — sqlite.Open(), implements db.DB
        queue.go       — sqlite.QueueRepository
        files.go       — sqlite.FileRepository
        devices.go     — sqlite.DeviceRepository
        media.go       — sqlite.MediaRepository
    postgres/
        db.go          — postgres.Open(), implements db.DB
        queue.go       — postgres.QueueRepository
        files.go       — postgres.FileRepository
        devices.go     — postgres.DeviceRepository
        media.go       — postgres.MediaRepository

internal/app/
    db.go              — OpenDB(cfg) dispatches to sqlite.Open or postgres.Open
```

The `internal/db/` package owns the interfaces. Sub-packages own implementations. `internal/app/` owns the wiring. This avoids a circular import: `sqlite` and `postgres` import `db` for the interfaces; `db` imports neither.

## DB interface

```go
// internal/db/db.go

type Dialect string

const (
    SQLite     Dialect = "sqlite"
    PostgreSQL Dialect = "postgres"
)

type DB interface {
    Queue()   QueueRepository
    Files()   FileRepository
    Devices() DeviceRepository
    Media()   MediaRepository
    Migrate() error
    Close()   error
}
```

## Repository interfaces

```go
type QueueRepository interface {
    Enqueue(filePath string, deviceID string) error
    DequeueNext() (*QueueEntry, error)
    SetStatus(id string, status QueueStatus, errMsg *string) error
    ResetStuck() error
    PendingCount() (int64, error)
}

type FileRepository interface {
    // EnsureFile inserts f if no row with the same SHA256 exists, then returns
    // the canonical ID and whether the row was newly created. Callers use isNew
    // to decide whether to extract and store Photo/Video metadata.
    EnsureFile(f *File) (id string, isNew bool, err error)
    InsertPhoto(fileID string, p *Photo) error
    InsertVideo(fileID string, v *Video) error
    InsertLocation(fileID string, mediaID string, pathOnMedia string) error
}

type DeviceRepository interface {
    EnsureLocal() (string, error) // returns UUID string; was int64
}

type MediaRepository interface {
    EnsureDriveForPath(path string, deviceID string) (string, error)
}
```

Domain types (`QueueEntry`, `File`, `QueueStatus`) move into `internal/db/` so both implementations and consumers share them.

## Migration system

The runner lives in `internal/db/migrate.go` and takes a `*sql.DB` plus `Dialect`. Resolution logic for migration N:

1. Check `migrations/shared/` — if found, use it for all dialects.
2. If not in shared, check both `migrations/sqlite/` and `migrations/postgres/`. If both exist, use the dialect-appropriate one.
3. If only one dialect directory has migration N, fail at startup — this is a broken state, not a runtime error.
4. If neither shared nor both dialects have N, fail.

Checksum verification, version tracking, and transactional application per migration are unchanged.

The migration resolution logic is unit tested with in-memory fake file trees covering: shared migration used, both dialect files used, and one dialect file missing triggers an error.

## Schema changes in 0001

`0001_initial_schema.sql` moves to `migrations/shared/` with two changes to make it dialect-neutral:

- `INTEGER PRIMARY KEY AUTOINCREMENT` on `Devices` and `IngestionQueue` replaced with `TEXT PRIMARY KEY`. UUIDs are generated in Go before insertion, consistent with the rest of the schema.
- `BLOB` on `Photo.metadata_blob` and `Video.metadata_blob` replaced with `TEXT`. Metadata is stored as JSON. PostgreSQL does not accept `BLOB`; `TEXT` works on both.

All other types (`REAL`, `CHECK`, `REFERENCES`, `ON DELETE CASCADE`) are standard SQL accepted by both dialects. SQLite PRAGMAs (`journal_mode=WAL`, `foreign_keys=ON`) remain in `sqlite.Open()`, not in migrations.

## Config

```go
type Config struct {
    DBType string `json:"db_type"`           // "sqlite" | "postgres"
    DBPath string `json:"db_path,omitempty"` // SQLite only
    DBDSN  string `json:"db_dsn,omitempty"`  // PostgreSQL only
}
```

The PostgreSQL implementation uses `github.com/jackc/pgx/v5` (stdlib-compatible mode via `pgx/v5/stdlib`) as the driver — it is actively maintained and supports `database/sql`. SQLite keeps `modernc.org/sqlite`.

Existing `fileditto db set-path` and `fileditto db path` commands are unchanged. A new `fileditto db configure` command provides an interactive setup flow using `github.com/manifoldco/promptui`:

```
? Select database type:
  > SQLite
    PostgreSQL

# SQLite branch:
? Database file path: /home/user/.local/share/fileditto/fileditto.db

# PostgreSQL branch:
? Connection string (DSN): postgres://user:pass@localhost:5432/fileditto
```

`internal/app/db.go` reads `DBType` and dispatches:

```go
func OpenDB(cfg *config.Config) (db.DB, error) {
    switch cfg.DBType {
    case string(db.SQLite):
        return sqlite.Open(cfg.DBPath)
    case string(db.PostgreSQL):
        return postgres.Open(cfg.DBDSN)
    default:
        return nil, fmt.Errorf("unknown db type %q", cfg.DBType)
    }
}
```

## Testing

Each implementation package has an `openTestDB(t *testing.T) db.DB` helper following the existing pattern (temp dir, run migrations, `t.Cleanup`).

PostgreSQL tests check for `TEST_POSTGRES_DSN`. If absent the test suite skips — `go test ./...` works locally without a running PostgreSQL instance.

Existing tests in `internal/ingestion/` require two updates: `openTestDB` returns `db.DB` instead of `*sql.DB`, and `testDeviceID` returns `string` instead of `int64`. Test logic is otherwise unchanged.
