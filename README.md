# fileditto

File archive and backup manager implementing a 3-2-1 backup strategy. Tracks files by content hash across multiple storage destinations, deduplicates across locations, and records backup health in a local SQLite database.

Designed to handle any file type, with initial focus on photos and videos.

## Requirements

- Go 1.26+

## Quick start

```bash
# Build
go build -o fileditto ./cmd/fileditto

# Set the database path
./fileditto db set-path ~/.local/share/fileditto/fileditto.db

# Run migrations
./fileditto migrate

# Ingest files from a source directory
./fileditto ingest /path/to/files
```

## Commands

- `fileditto ingest <path>` — scan and ingest files from a directory
- `fileditto migrate` — run pending database migrations
- `fileditto db set-path <path>` — configure the database file location
- `fileditto db path` — print the configured database path

## Configuration

Config is stored at `$XDG_CONFIG_HOME/fileditto/config.json` (defaults to `~/.config/fileditto/config.json`).

## Project structure

```
cmd/fileditto/     CLI entry point
internal/
  config/          Config file management
  db/              Database connection, migrations, UUID generation
  scanner/         File discovery
  ingestion/       Ingestion queue and worker
```

## Development

See [concept.md](concept.md) for the full design and MVP roadmap.
