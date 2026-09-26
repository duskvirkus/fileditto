## Context

pxvault started as a Python + Go split: Python handled file scanning and metadata extraction
(Pillow, ffprobe, smbprotocol), Go handled the database layer. Two phases are complete and
merged (P1: database schema, P2: file scanner and ingestion). No database instances exist
in the wild — the project is still in active development.

The two-language split adds deployment complexity with no remaining benefit. The Go library
ecosystem covers all operations Python was doing. A single Go binary is a better fit for the
target deployment environment (NAS, Docker, personal server).

## Goals / Non-Goals

**Goals:**
- Remove all Python code and tooling (scanner/, db/ Python, tests/, pyproject.toml, uv.lock)
- Port file discovery and ingestion logic to Go
- Add ingestion_queue table to the initial SQLite schema
- Restructure as a two-stage pipeline: discovery enqueues all files, ingestion worker processes them
- Result is a pure Go project with no Python dependency

**Non-Goals:**
- SMB/network share support (dropped; can be added in a future change when needed)
- Backup engine or 3-2-1 logic (P3, separate change)
- TUI or CLI beyond what already exists in cmd/

## Decisions

**Two-stage pipeline with queue as handoff**
The system splits into two independent stages:

1. Discovery — walks the filesystem and enqueues every file path without filtering. No
   classification, no MIME detection, no skipping. This keeps discovery simple and ensures
   that if support for a new file type is added later, no re-discovery is needed — those
   files are already in the queue.

2. Ingestion worker — reads pending queue entries, classifies the file (MIME detection),
   hashes it, extracts metadata, and writes to Files/Photo/Video. Unsupported file types
   get status=unsupported rather than status=failed — a clean terminal state, not an error.

**Status enum**
`pending` → `processing` → `done` | `failed` | `unsupported`

`unsupported` is intentionally separate from `failed`. A failed file had an error during
processing. An unsupported file was simply not a type the system handles yet.

**SQLite as the ingestion queue**
The Python queue was an in-memory deque with optional JSON file persistence. The queue
table holds only what is needed to track processing state (path + status). All content
(hash, metadata) is computed in memory by the ingestion worker and written directly to
Files/Photo/Video on success.

**Ingestion queue added to initial migration**
Since no deployed database instances exist, the ingestion_queue table is added to
0001_initial_schema.sql rather than creating a new migration. Simpler schema history.

**Queue schema**
```sql
CREATE TABLE ingestion_queue (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    file_path     TEXT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'pending'
                      CHECK(status IN ('pending', 'processing', 'done', 'failed', 'unsupported')),
    attempt_count INTEGER,
    error         TEXT,
    created_at    TEXT NOT NULL,
    updated_at    TEXT NOT NULL
);
```

**Library choices**
- github.com/gabriel-vasile/mimetype — MIME detection in the ingestion worker (not discovery)
- github.com/rwcarlsen/goexif/exif — EXIF extraction for images
- golang.org/x/image — image dimension decoding (JPEG, PNG, etc.)
- ffprobe — video metadata via subprocess (no new dep)

**Package structure**
```
internal/
  db/
    migrations/
      0001_initial_schema.sql  (modified to include ingestion_queue)
    db.go                      (existing)
    uuid.go                    (existing)
  scanner/
    discover.go   -- walk FS, enqueue all file paths
  ingestion/
    queue.go      -- enqueue, dequeue, status updates
    worker.go     -- classify, hash, extract metadata, write to DB
```

## Risks / Trade-offs

[Go EXIF coverage] goexif covers common cases but is thinner than Pillow's EXIF support →
Acceptable; EXIF extraction is best-effort and stored as a JSON blob.

[ffprobe subprocess] Video metadata still requires ffprobe installed on the host →
Same constraint existed in Python; document as a runtime dependency.

[Test coverage] Python tests are deleted with no Go equivalents yet →
Go tests for the new packages are written as part of this change.

## Open Questions

None. All decisions resolved during design.
