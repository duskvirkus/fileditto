# Go-Only Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove all Python code and replace it with a pure Go pipeline: discovery enqueues every file path into SQLite, then the ingestion worker classifies, hashes, extracts metadata, and writes to the database.

**Architecture:** Two-stage pipeline. Discovery (`internal/scanner`) walks the filesystem and enqueues all file paths without filtering. The ingestion worker (`internal/ingestion`) picks up pending entries, classifies via MIME, hashes, extracts metadata, and writes to `Files`/`Photo`/`Video`. The queue is a SQLite table added to the initial schema.

**Tech Stack:** Go, modernc.org/sqlite (existing), github.com/gabriel-vasile/mimetype, github.com/rwcarlsen/goexif/exif, golang.org/x/image, ffprobe (subprocess, runtime dep)

---

## File Map

**Delete:**
- `scanner/` — Python scanner module
- `db/` — Python db module (connection.py, uuid_util.py, __init__.py)
- `tests/` — Python test suite
- `pyproject.toml`
- `uv.lock`

**Modify:**
- `internal/db/migrations/0001_initial_schema.sql` — add `ingestion_queue` table

**Create:**
- `internal/ingestion/queue.go` — enqueue, dequeue, status updates, pending count
- `internal/ingestion/queue_test.go`
- `internal/ingestion/worker.go` — MIME classify, hash, metadata extraction, DB writes
- `internal/ingestion/worker_test.go`
- `internal/scanner/discover.go` — walk filesystem, enqueue all file paths
- `internal/scanner/discover_test.go`

---

## Task 1: Remove Python files

**Files:**
- Delete: `scanner/`, `db/`, `tests/`, `pyproject.toml`, `uv.lock`

- [ ] **Step 1: Delete Python directories and files**

```bash
rm -rf scanner/ db/ tests/ pyproject.toml uv.lock
```

- [ ] **Step 2: Verify Go builds cleanly**

```bash
go build ./...
```

Expected: no output (success). If errors appear, a Go file was importing from a Python path — investigate before continuing.

- [ ] **Step 3: Commit**

```bash
git add -A
git commit -m "refactor: remove Python scanner and tooling"
```

---

## Task 2: Add ingestion_queue to initial schema

**Files:**
- Modify: `internal/db/migrations/0001_initial_schema.sql`

- [ ] **Step 1: Write the failing test**

Add to `internal/db/db_test.go`:

```go
// --- ingestion_queue table ---

func TestIngestionQueue_TableExists(t *testing.T) {
	conn := openTestDB(t)

	_, err := conn.Exec(
		`INSERT INTO ingestion_queue (file_path, status, created_at, updated_at)
		 VALUES (?, 'pending', ?, ?)`,
		"/some/file.jpg", "2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("expected ingestion_queue to exist and accept insert, got: %v", err)
	}
}

func TestIngestionQueue_InvalidStatusRejected(t *testing.T) {
	conn := openTestDB(t)

	_, err := conn.Exec(
		`INSERT INTO ingestion_queue (file_path, status, created_at, updated_at)
		 VALUES (?, 'unknown', ?, ?)`,
		"/some/file.jpg", "2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z",
	)
	if err == nil {
		t.Error("expected CHECK constraint violation for invalid status, but insert succeeded")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/db/... -run "TestIngestionQueue" -v
```

Expected: FAIL — table `ingestion_queue` does not exist.

- [ ] **Step 3: Add ingestion_queue to 0001_initial_schema.sql**

Append to `internal/db/migrations/0001_initial_schema.sql`:

```sql
-- ingestion_queue: holds file paths discovered during scanning, pending ingestion.
CREATE TABLE ingestion_queue (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    file_path     TEXT    NOT NULL,
    status        TEXT    NOT NULL DEFAULT 'pending'
                          CHECK(status IN ('pending', 'processing', 'done', 'failed', 'unsupported')),
    attempt_count INTEGER,
    error         TEXT,
    created_at    TEXT    NOT NULL,
    updated_at    TEXT    NOT NULL
);
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/db/... -v
```

Expected: all tests PASS including the two new ones.

- [ ] **Step 5: Commit**

```bash
git add internal/db/migrations/0001_initial_schema.sql internal/db/db_test.go
git commit -m "feat: add ingestion_queue table to initial schema"
```

---

## Task 3: Add Go dependencies

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Add dependencies**

```bash
go get github.com/gabriel-vasile/mimetype
go get github.com/rwcarlsen/goexif/exif
go get golang.org/x/image
```

- [ ] **Step 2: Verify build still passes**

```bash
go build ./...
```

Expected: no output.

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add mimetype, goexif, and x/image dependencies"
```

---

## Task 4: Implement ingestion queue

**Files:**
- Create: `internal/ingestion/queue.go`
- Create: `internal/ingestion/queue_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/ingestion/queue_test.go`:

```go
package ingestion_test

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/duskvirkus/pxvault/internal/db"
	"github.com/duskvirkus/pxvault/internal/ingestion"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	conn, err := db.OpenDB(path)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	if err := db.RunMigrations(conn); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func TestEnqueue_CreatesPendingEntry(t *testing.T) {
	conn := openTestDB(t)
	if err := ingestion.Enqueue(conn, "/some/file.jpg"); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	var status string
	if err := conn.QueryRow(
		`SELECT status FROM ingestion_queue WHERE file_path = ?`, "/some/file.jpg",
	).Scan(&status); err != nil {
		t.Fatalf("query: %v", err)
	}
	if status != "pending" {
		t.Errorf("expected status=pending, got %q", status)
	}
}

func TestDequeueNext_ReturnsNilWhenEmpty(t *testing.T) {
	conn := openTestDB(t)
	entry, err := ingestion.DequeueNext(conn)
	if err != nil {
		t.Fatalf("DequeueNext: %v", err)
	}
	if entry != nil {
		t.Errorf("expected nil, got %+v", entry)
	}
}

func TestDequeueNext_MarksEntryAsProcessing(t *testing.T) {
	conn := openTestDB(t)
	if err := ingestion.Enqueue(conn, "/some/file.jpg"); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	entry, err := ingestion.DequeueNext(conn)
	if err != nil {
		t.Fatalf("DequeueNext: %v", err)
	}
	if entry == nil {
		t.Fatal("expected entry, got nil")
	}
	if entry.Status != ingestion.StatusProcessing {
		t.Errorf("expected Status=processing, got %q", entry.Status)
	}

	var dbStatus string
	conn.QueryRow(`SELECT status FROM ingestion_queue WHERE id = ?`, entry.ID).Scan(&dbStatus)
	if dbStatus != "processing" {
		t.Errorf("expected DB status=processing, got %q", dbStatus)
	}
}

func TestSetStatus_Done(t *testing.T) {
	conn := openTestDB(t)
	ingestion.Enqueue(conn, "/some/file.jpg")
	entry, _ := ingestion.DequeueNext(conn)

	if err := ingestion.SetStatus(conn, entry.ID, ingestion.StatusDone, nil); err != nil {
		t.Fatalf("SetStatus: %v", err)
	}
	var status string
	conn.QueryRow(`SELECT status FROM ingestion_queue WHERE id = ?`, entry.ID).Scan(&status)
	if status != "done" {
		t.Errorf("expected status=done, got %q", status)
	}
}

func TestSetStatus_FailedIncrementsAttemptCount(t *testing.T) {
	conn := openTestDB(t)
	ingestion.Enqueue(conn, "/bad/file.jpg")
	entry, _ := ingestion.DequeueNext(conn)

	errMsg := "something went wrong"
	if err := ingestion.SetStatus(conn, entry.ID, ingestion.StatusFailed, &errMsg); err != nil {
		t.Fatalf("SetStatus: %v", err)
	}

	var status, errCol string
	var attempts *int
	conn.QueryRow(
		`SELECT status, error, attempt_count FROM ingestion_queue WHERE id = ?`, entry.ID,
	).Scan(&status, &errCol, &attempts)

	if status != "failed" {
		t.Errorf("expected status=failed, got %q", status)
	}
	if errCol != errMsg {
		t.Errorf("expected error=%q, got %q", errMsg, errCol)
	}
	if attempts == nil || *attempts != 1 {
		t.Errorf("expected attempt_count=1, got %v", attempts)
	}
}

func TestPendingCount(t *testing.T) {
	conn := openTestDB(t)
	ingestion.Enqueue(conn, "/file1.jpg")
	ingestion.Enqueue(conn, "/file2.jpg")
	ingestion.Enqueue(conn, "/file3.jpg")

	count, err := ingestion.PendingCount(conn)
	if err != nil {
		t.Fatalf("PendingCount: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3, got %d", count)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/ingestion/... -v
```

Expected: FAIL — package `ingestion` does not exist.

- [ ] **Step 3: Implement queue.go**

Create `internal/ingestion/queue.go`:

```go
package ingestion

import (
	"database/sql"
	"fmt"
	"time"
)

type Status string

const (
	StatusPending     Status = "pending"
	StatusProcessing  Status = "processing"
	StatusDone        Status = "done"
	StatusFailed      Status = "failed"
	StatusUnsupported Status = "unsupported"
)

type QueueEntry struct {
	ID           int64
	FilePath     string
	Status       Status
	AttemptCount *int
	Error        *string
	CreatedAt    string
	UpdatedAt    string
}

// Enqueue adds a file path to the ingestion queue with status=pending.
func Enqueue(db *sql.DB, filePath string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(
		`INSERT INTO ingestion_queue (file_path, status, created_at, updated_at)
		 VALUES (?, 'pending', ?, ?)`,
		filePath, now, now,
	)
	if err != nil {
		return fmt.Errorf("enqueue %s: %w", filePath, err)
	}
	return nil
}

// DequeueNext atomically claims the oldest pending entry and marks it processing.
// Returns nil if the queue is empty.
func DequeueNext(db *sql.DB) (*QueueEntry, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck

	var entry QueueEntry
	err = tx.QueryRow(
		`SELECT id, file_path, status, attempt_count, error, created_at, updated_at
		 FROM ingestion_queue WHERE status = 'pending' ORDER BY id LIMIT 1`,
	).Scan(
		&entry.ID, &entry.FilePath, &entry.Status,
		&entry.AttemptCount, &entry.Error,
		&entry.CreatedAt, &entry.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("dequeue: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := tx.Exec(
		`UPDATE ingestion_queue SET status = 'processing', updated_at = ? WHERE id = ?`,
		now, entry.ID,
	); err != nil {
		return nil, fmt.Errorf("mark processing: %w", err)
	}

	entry.Status = StatusProcessing
	return &entry, tx.Commit()
}

// SetStatus updates the status and optional error message for a queue entry.
// StatusFailed also increments attempt_count.
func SetStatus(db *sql.DB, id int64, status Status, errMsg *string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	var err error
	if status == StatusFailed {
		_, err = db.Exec(
			`UPDATE ingestion_queue
			 SET status = ?, error = ?, updated_at = ?,
			     attempt_count = COALESCE(attempt_count, 0) + 1
			 WHERE id = ?`,
			status, errMsg, now, id,
		)
	} else {
		_, err = db.Exec(
			`UPDATE ingestion_queue SET status = ?, error = ?, updated_at = ? WHERE id = ?`,
			status, errMsg, now, id,
		)
	}
	if err != nil {
		return fmt.Errorf("set status %s for id %d: %w", status, id, err)
	}
	return nil
}

// PendingCount returns the number of entries with status=pending.
func PendingCount(db *sql.DB) (int, error) {
	var count int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM ingestion_queue WHERE status = 'pending'`,
	).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/ingestion/... -v -run "TestEnqueue|TestDequeue|TestSetStatus|TestPending"
```

Expected: all 6 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/ingestion/queue.go internal/ingestion/queue_test.go
git commit -m "feat: add ingestion queue backed by SQLite"
```

---

## Task 5: Implement file discovery

**Files:**
- Create: `internal/scanner/discover.go`
- Create: `internal/scanner/discover_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/scanner/discover_test.go`:

```go
package scanner_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/duskvirkus/pxvault/internal/db"
	"github.com/duskvirkus/pxvault/internal/scanner"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	conn, err := db.OpenDB(path)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	if err := db.RunMigrations(conn); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func TestDiscover_EnqueuesAllFiles(t *testing.T) {
	conn := openTestDB(t)
	dir := t.TempDir()

	paths := []string{
		"photo.jpg",
		"video.mp4",
		"document.pdf",
		filepath.Join("subdir", "image.png"),
	}
	for _, p := range paths {
		full := filepath.Join(dir, p)
		os.MkdirAll(filepath.Dir(full), 0755)
		os.WriteFile(full, []byte("data"), 0644)
	}

	if err := scanner.Discover(conn, dir); err != nil {
		t.Fatalf("Discover: %v", err)
	}

	var count int
	conn.QueryRow(`SELECT COUNT(*) FROM ingestion_queue`).Scan(&count)
	if count != 4 {
		t.Errorf("expected 4 queue entries, got %d", count)
	}
}

func TestDiscover_AllEntriesArePending(t *testing.T) {
	conn := openTestDB(t)
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("data"), 0644)

	scanner.Discover(conn, dir)

	var status string
	conn.QueryRow(`SELECT status FROM ingestion_queue`).Scan(&status)
	if status != "pending" {
		t.Errorf("expected status=pending, got %q", status)
	}
}

func TestDiscover_SkipsDirectories(t *testing.T) {
	conn := openTestDB(t)
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "subdir"), 0755)
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("data"), 0644)

	scanner.Discover(conn, dir)

	var count int
	conn.QueryRow(`SELECT COUNT(*) FROM ingestion_queue`).Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 entry (file only, not directory), got %d", count)
	}
}

func TestDiscover_ReturnsErrorForNonexistentPath(t *testing.T) {
	conn := openTestDB(t)
	err := scanner.Discover(conn, "/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Error("expected error for nonexistent path, got nil")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/scanner/... -v
```

Expected: FAIL — package `scanner` does not exist.

- [ ] **Step 3: Implement discover.go**

Create `internal/scanner/discover.go`:

```go
package scanner

import (
	"database/sql"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/duskvirkus/pxvault/internal/ingestion"
)

// Discover walks root and enqueues every file it finds, without filtering.
// Directories are skipped. All file types are enqueued — classification
// happens in the ingestion worker.
func Discover(db *sql.DB, root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk %s: %w", path, err)
		}
		if d.IsDir() {
			return nil
		}
		return ingestion.Enqueue(db, path)
	})
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/scanner/... -v
```

Expected: all 4 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/scanner/discover.go internal/scanner/discover_test.go
git commit -m "feat: add filesystem discovery that enqueues all files"
```

---

## Task 6: Implement ingestion worker

**Files:**
- Create: `internal/ingestion/worker.go`
- Create: `internal/ingestion/worker_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/ingestion/worker_test.go`:

```go
package ingestion_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/duskvirkus/pxvault/internal/ingestion"
)

// writePNG creates a minimal valid 1x1 PNG at path.
func writePNG(t *testing.T, path string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.White)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode PNG: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		t.Fatalf("write PNG: %v", err)
	}
}

func TestProcessNext_ReturnsFalseWhenQueueEmpty(t *testing.T) {
	conn := openTestDB(t)
	processed, err := ingestion.ProcessNext(conn)
	if err != nil {
		t.Fatalf("ProcessNext: %v", err)
	}
	if processed {
		t.Error("expected false (empty queue), got true")
	}
}

func TestProcessNext_PhotoFileWrittenToFilesAndPhoto(t *testing.T) {
	conn := openTestDB(t)

	imgPath := filepath.Join(t.TempDir(), "test.png")
	writePNG(t, imgPath)

	ingestion.Enqueue(conn, imgPath)

	processed, err := ingestion.ProcessNext(conn)
	if err != nil {
		t.Fatalf("ProcessNext: %v", err)
	}
	if !processed {
		t.Fatal("expected processed=true")
	}

	var fileCount int
	conn.QueryRow(`SELECT COUNT(*) FROM Files WHERE file_type = 'photo'`).Scan(&fileCount)
	if fileCount != 1 {
		t.Errorf("expected 1 Files row with file_type=photo, got %d", fileCount)
	}

	var photoCount int
	conn.QueryRow(`SELECT COUNT(*) FROM Photo`).Scan(&photoCount)
	if photoCount != 1 {
		t.Errorf("expected 1 Photo row, got %d", photoCount)
	}

	var status string
	conn.QueryRow(`SELECT status FROM ingestion_queue WHERE file_path = ?`, imgPath).Scan(&status)
	if status != "done" {
		t.Errorf("expected queue status=done, got %q", status)
	}
}

func TestProcessNext_UnsupportedFileSetToUnsupported(t *testing.T) {
	conn := openTestDB(t)

	txtPath := filepath.Join(t.TempDir(), "notes.txt")
	os.WriteFile(txtPath, []byte("hello"), 0644)

	ingestion.Enqueue(conn, txtPath)
	ingestion.ProcessNext(conn)

	var status string
	conn.QueryRow(`SELECT status FROM ingestion_queue WHERE file_path = ?`, txtPath).Scan(&status)
	if status != "unsupported" {
		t.Errorf("expected status=unsupported, got %q", status)
	}

	var fileCount int
	conn.QueryRow(`SELECT COUNT(*) FROM Files`).Scan(&fileCount)
	if fileCount != 0 {
		t.Errorf("expected no Files rows for unsupported file, got %d", fileCount)
	}
}

func TestProcessNext_DuplicateFileNotReinserted(t *testing.T) {
	conn := openTestDB(t)

	imgPath := filepath.Join(t.TempDir(), "test.png")
	writePNG(t, imgPath)

	ingestion.Enqueue(conn, imgPath)
	ingestion.ProcessNext(conn)

	// Enqueue the same file again
	ingestion.Enqueue(conn, imgPath)
	ingestion.ProcessNext(conn)

	var fileCount int
	conn.QueryRow(`SELECT COUNT(*) FROM Files`).Scan(&fileCount)
	if fileCount != 1 {
		t.Errorf("expected 1 Files row (dedup), got %d", fileCount)
	}

	var doneCount int
	conn.QueryRow(
		`SELECT COUNT(*) FROM ingestion_queue WHERE status = 'done'`,
	).Scan(&doneCount)
	if doneCount != 2 {
		t.Errorf("expected both queue entries to be done, got %d done", doneCount)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/ingestion/... -run "TestProcessNext" -v
```

Expected: FAIL — `ProcessNext` not defined.

- [ ] **Step 3: Implement worker.go**

Create `internal/ingestion/worker.go`:

```go
package ingestion

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/duskvirkus/pxvault/internal/db"
	"github.com/gabriel-vasile/mimetype"
	"github.com/rwcarlsen/goexif/exif"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// ProcessNext dequeues and processes one pending entry.
// Returns (true, nil) if an entry was processed, (false, nil) if the queue is empty.
func ProcessNext(sqlDB *sql.DB) (bool, error) {
	entry, err := DequeueNext(sqlDB)
	if err != nil {
		return false, err
	}
	if entry == nil {
		return false, nil
	}

	if err := processEntry(sqlDB, entry); err != nil {
		errStr := err.Error()
		_ = SetStatus(sqlDB, entry.ID, StatusFailed, &errStr)
	}
	return true, nil
}

func processEntry(sqlDB *sql.DB, entry *QueueEntry) error {
	mime, err := mimetype.DetectFile(entry.FilePath)
	if err != nil {
		return fmt.Errorf("detect mime: %w", err)
	}

	fileType := classifyMIME(mime.String())
	if fileType == "unsupported" {
		return SetStatus(sqlDB, entry.ID, StatusUnsupported, nil)
	}

	sha, size, err := hashFile(entry.FilePath)
	if err != nil {
		return err
	}

	fileUUID, err := db.FileUUID(sha)
	if err != nil {
		return fmt.Errorf("derive uuid: %w", err)
	}

	var existing string
	err = sqlDB.QueryRow(`SELECT id FROM Files WHERE sha256 = ?`, sha).Scan(&existing)
	if err == nil {
		return SetStatus(sqlDB, entry.ID, StatusDone, nil)
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("dedup check: %w", err)
	}

	meta, err := extractMetadata(entry.FilePath, fileType)
	if err != nil {
		meta = map[string]interface{}{}
	}

	if err := writeFile(sqlDB, fileUUID.String(), sha, size, fileType, meta); err != nil {
		return err
	}

	return SetStatus(sqlDB, entry.ID, StatusDone, nil)
}

func classifyMIME(mime string) string {
	switch {
	case strings.HasPrefix(mime, "image/"):
		return "photo"
	case strings.HasPrefix(mime, "video/"):
		return "video"
	default:
		return "unsupported"
	}
}

func hashFile(path string) (sha256hex string, size int64, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, fmt.Errorf("open for hash: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	size, err = io.Copy(h, f)
	if err != nil {
		return "", 0, fmt.Errorf("hash: %w", err)
	}
	return fmt.Sprintf("%x", h.Sum(nil)), size, nil
}

func extractMetadata(path, fileType string) (map[string]interface{}, error) {
	switch fileType {
	case "photo":
		return extractImageMetadata(path)
	case "video":
		return extractVideoMetadata(path)
	default:
		return map[string]interface{}{}, nil
	}
}

func extractImageMetadata(path string) (map[string]interface{}, error) {
	meta := map[string]interface{}{}

	f, err := os.Open(path)
	if err != nil {
		return meta, err
	}
	defer f.Close()

	cfg, _, err := image.DecodeConfig(f)
	if err == nil {
		meta["width"] = cfg.Width
		meta["height"] = cfg.Height
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return meta, nil
	}
	x, err := exif.Decode(f)
	if err == nil {
		if lat, long, err := x.LatLong(); err == nil {
			meta["latitude"] = lat
			meta["longitude"] = long
		}
		if tm, err := x.DateTime(); err == nil {
			meta["taken_at"] = tm.UTC().Format(time.RFC3339)
		}
	}

	return meta, nil
}

type ffprobeOut struct {
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
	Streams []struct {
		CodecType string `json:"codec_type"`
		CodecName string `json:"codec_name"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
	} `json:"streams"`
}

func extractVideoMetadata(path string) (map[string]interface{}, error) {
	out, err := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration:stream=codec_name,width,height,codec_type",
		"-of", "json",
		path,
	).Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe: %w", err)
	}

	var fp ffprobeOut
	if err := json.Unmarshal(out, &fp); err != nil {
		return nil, err
	}

	meta := map[string]interface{}{}
	if d, err := strconv.ParseFloat(fp.Format.Duration, 64); err == nil {
		meta["duration_seconds"] = d
	}
	for _, s := range fp.Streams {
		if s.CodecType == "video" {
			meta["codec"] = s.CodecName
			meta["width"] = s.Width
			meta["height"] = s.Height
			break
		}
	}
	return meta, nil
}

func writeFile(sqlDB *sql.DB, fileID, sha string, size int64, fileType string, meta map[string]interface{}) error {
	tx, err := sqlDB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := tx.Exec(
		`INSERT INTO Files (id, sha256, size_bytes, file_type, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		fileID, sha, size, fileType, now, now,
	); err != nil {
		return fmt.Errorf("insert Files: %w", err)
	}

	metaJSON, _ := json.Marshal(meta)

	switch fileType {
	case "photo":
		width, _ := meta["width"].(int)
		height, _ := meta["height"].(int)
		if _, err := tx.Exec(
			`INSERT INTO Photo (id, width_px, height_px, metadata_blob) VALUES (?, ?, ?, ?)`,
			fileID, width, height, metaJSON,
		); err != nil {
			return fmt.Errorf("insert Photo: %w", err)
		}
	case "video":
		width, _ := meta["width"].(int)
		height, _ := meta["height"].(int)
		codec, _ := meta["codec"].(string)
		durSec, _ := meta["duration_seconds"].(float64)
		if _, err := tx.Exec(
			`INSERT INTO Video (id, duration_seconds, width_px, height_px, codec, metadata_blob)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			fileID, durSec, width, height, codec, metaJSON,
		); err != nil {
			return fmt.Errorf("insert Video: %w", err)
		}
	}

	return tx.Commit()
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/ingestion/... -v
```

Expected: all tests PASS. (Video tests skip if ffprobe not in PATH.)

- [ ] **Step 5: Commit**

```bash
git add internal/ingestion/worker.go internal/ingestion/worker_test.go
git commit -m "feat: add ingestion worker with MIME classification, hashing, and metadata extraction"
```

---

## Task 7: Full verification

- [ ] **Step 1: Run the complete test suite**

```bash
go test ./... -v
```

Expected: all tests PASS. No compilation errors, no skipped packages.

- [ ] **Step 2: Verify no Python remnants**

```bash
find . -name "*.py" -not -path "./.git/*"
```

Expected: no output.

- [ ] **Step 3: Verify the build is clean**

```bash
go build ./...
go vet ./...
```

Expected: no output from either command.

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "chore: verify go-only refactor complete"
```
