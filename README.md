# fileditto

File archive and backup manager implementing a 3-2-1 backup strategy. Tracks files by content hash across multiple storage destinations, deduplicates across locations, and records backup health in a local database.

Designed to handle any file type, with initial focus on photos and videos.

## Requirements

- Go 1.26+
- `ffprobe` (part of [ffmpeg](https://ffmpeg.org/)) — used for video metadata extraction
- SQLite (default) or a PostgreSQL instance

## Quick start

```bash
# Build
go build -o fileditto ./cmd/fileditto

# Configure the database interactively (SQLite or PostgreSQL)
./fileditto db configure

# Run migrations
./fileditto migrate

# Ingest files from a source directory
./fileditto ingest /path/to/files
```

## Commands

- `fileditto ingest <path>` — scan and ingest files from a directory
- `fileditto migrate` — run pending database migrations
- `fileditto db configure` — interactively set up the database connection
- `fileditto db set-path <path>` — shortcut to configure SQLite at a specific path
- `fileditto db path` — print the configured database path or DSN

## Configuration

Config is stored at `$XDG_CONFIG_HOME/fileditto/config.json` (defaults to `~/.config/fileditto/config.json`).

### SQLite

```json
{
  "db_type": "sqlite",
  "db_path": "/home/user/.local/share/fileditto/fileditto.db"
}
```

### PostgreSQL

```json
{
  "db_type": "postgres",
  "db_dsn": "postgres://user:pass@localhost:5432/fileditto"
}
```

## Project structure

```
cmd/fileditto/     CLI entry point
internal/
  app/             Database factory (OpenDB)
  config/          Config file management
  db/              Interfaces, migrations, UUID helpers
    sqlite/        SQLite implementation
    postgres/      PostgreSQL implementation
  hardware/        Disk detection
  scanner/         File discovery
  ingestion/       Ingestion worker
```

## Development

### Running tests

```bash
# All tests (PostgreSQL tests skipped if Docker is unavailable)
go test ./...

# PostgreSQL integration tests (requires Docker)
go test ./internal/db/postgres/... -timeout 120s
```

PostgreSQL tests spin up a `postgres:16-alpine` container automatically via [testcontainers-go](https://testcontainers.com/guides/getting-started-with-testcontainers-for-go/) and tear it down when done.

See [concept.md](concept.md) for the full design and MVP roadmap.
