package sqlite_test

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/duskvirkus/fileditto/internal/db"
	"github.com/duskvirkus/fileditto/internal/db/sqlite"
)

func openTestDB(t *testing.T) db.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	conn, err := sqlite.Open(path)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	if err := conn.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func rawConn(t *testing.T, d db.DB) *sql.DB {
	t.Helper()
	rc, ok := d.(db.RawConner)
	if !ok {
		t.Fatal("db.DB does not implement db.RawConner")
	}
	return rc.RawConn()
}

func testDeviceID(t *testing.T, d db.DB) string {
	t.Helper()
	id, err := d.Devices().EnsureLocal()
	if err != nil {
		t.Fatalf("EnsureLocal: %v", err)
	}
	return id
}

// --- Schema tests ---

func TestOpen_MigratesSetsSchemaVersion1(t *testing.T) {
	d := openTestDB(t)
	conn := rawConn(t, d)
	var version int
	if err := conn.QueryRow("SELECT version FROM schema_version").Scan(&version); err != nil {
		t.Fatalf("query schema_version: %v", err)
	}
	if version != 1 {
		t.Errorf("expected schema_version=1, got %d", version)
	}
}

func TestCascade_PhotoDeletedWithFile(t *testing.T) {
	d := openTestDB(t)
	conn := rawConn(t, d)

	fileID := "file-photo-cascade"
	sha256 := "aabbccdd00112233445566778899aabbccddeeff00112233445566778899aabb"
	conn.Exec(`INSERT INTO Files (id, sha256, size_bytes, file_type) VALUES (?, ?, ?, ?)`,
		fileID, sha256, 1024, "photo")
	conn.Exec(`INSERT INTO Photo (id, width_px, height_px) VALUES (?, ?, ?)`, fileID, 1920, 1080)
	conn.Exec(`DELETE FROM Files WHERE id = ?`, fileID)

	var count int
	conn.QueryRow("SELECT COUNT(*) FROM Photo WHERE id = ?", fileID).Scan(&count)
	if count != 0 {
		t.Error("expected Photo row deleted via CASCADE, but it still exists")
	}
}

func TestCascade_VideoDeletedWithFile(t *testing.T) {
	d := openTestDB(t)
	conn := rawConn(t, d)

	fileID := "file-video-cascade"
	sha256 := "bbccdd0011223344556677889900aabbccddeeff00112233445566778899bbcc"
	conn.Exec(`INSERT INTO Files (id, sha256, size_bytes, file_type) VALUES (?, ?, ?, ?)`,
		fileID, sha256, 2048, "video")
	conn.Exec(`INSERT INTO Video (id, duration_seconds, width_px, height_px) VALUES (?, ?, ?, ?)`,
		fileID, 120.5, 3840, 2160)
	conn.Exec(`DELETE FROM Files WHERE id = ?`, fileID)

	var count int
	conn.QueryRow("SELECT COUNT(*) FROM Video WHERE id = ?", fileID).Scan(&count)
	if count != 0 {
		t.Error("expected Video row deleted via CASCADE, but it still exists")
	}
}

func TestCascade_LocationsDeletedWithMedia(t *testing.T) {
	d := openTestDB(t)
	conn := rawConn(t, d)

	fileID := "file-loc-cascade"
	sha256 := "ccdd001122334455667788990011aabbccddeeff001122334455667788990000"
	conn.Exec(`INSERT INTO Files (id, sha256, size_bytes, file_type) VALUES (?, ?, ?, ?)`,
		fileID, sha256, 512, "generic")
	mediaID := "media-loc-cascade"
	conn.Exec(`INSERT INTO Media (id, media_type, status) VALUES (?, ?, ?)`, mediaID, "drive", "connected")
	locID := "loc-cascade"
	conn.Exec(`INSERT INTO Locations (id, file_id, media_id, path_on_media, status) VALUES (?, ?, ?, ?, ?)`,
		locID, fileID, mediaID, "/files/photo.jpg", "healthy")

	conn.Exec(`DELETE FROM Media WHERE id = ?`, mediaID)

	var count int
	conn.QueryRow("SELECT COUNT(*) FROM Locations WHERE id = ?", locID).Scan(&count)
	if count != 0 {
		t.Error("expected Location deleted via CASCADE, but it still exists")
	}
}

func TestConstraint_InvalidFileTypeRejected(t *testing.T) {
	d := openTestDB(t)
	conn := rawConn(t, d)
	_, err := conn.Exec(`INSERT INTO Files (id, sha256, size_bytes, file_type) VALUES (?, ?, ?, ?)`,
		"bad", "dead000000000000000000000000000000000000000000000000000000000000", 100, "document")
	if err == nil {
		t.Error("expected CHECK violation for invalid file_type")
	}
}

func TestConstraint_InvalidMediaStatusRejected(t *testing.T) {
	d := openTestDB(t)
	conn := rawConn(t, d)
	_, err := conn.Exec(`INSERT INTO Media (id, media_type, status) VALUES (?, ?, ?)`,
		"bad-media", "drive", "destroyed")
	if err == nil {
		t.Error("expected CHECK violation for invalid Media.status")
	}
}

func TestMigrationChecksumMismatchHalts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "checksum.db")
	conn, err := sqlite.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer conn.Close()
	if err := conn.Migrate(); err != nil {
		t.Fatalf("first Migrate: %v", err)
	}
	raw := conn.RawConn()
	raw.Exec("UPDATE schema_migrations SET checksum = 'deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef' WHERE migration_number = 1")
	if err := conn.Migrate(); err == nil {
		t.Error("expected error on checksum mismatch, got nil")
	}
}

// --- Queue repository tests ---

func TestQueue_EnqueueCreatesPendingEntry(t *testing.T) {
	d := openTestDB(t)
	devID := testDeviceID(t, d)
	if err := d.Queue().Enqueue("/some/file.jpg", devID); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	conn := rawConn(t, d)
	var status string
	conn.QueryRow(`SELECT status FROM IngestionQueue WHERE file_path = ?`, "/some/file.jpg").Scan(&status)
	if status != "pending" {
		t.Errorf("expected pending, got %q", status)
	}
}

func TestQueue_DequeueNextReturnsNilWhenEmpty(t *testing.T) {
	d := openTestDB(t)
	entry, err := d.Queue().DequeueNext()
	if err != nil {
		t.Fatalf("DequeueNext: %v", err)
	}
	if entry != nil {
		t.Errorf("expected nil, got %+v", entry)
	}
}

func TestQueue_DequeueNextMarksProcessing(t *testing.T) {
	d := openTestDB(t)
	devID := testDeviceID(t, d)
	d.Queue().Enqueue("/some/file.jpg", devID)
	entry, err := d.Queue().DequeueNext()
	if err != nil {
		t.Fatalf("DequeueNext: %v", err)
	}
	if entry == nil {
		t.Fatal("expected entry, got nil")
	}
	if entry.Status != db.QueueStatusProcessing {
		t.Errorf("expected processing, got %q", entry.Status)
	}
}

func TestQueue_SetStatusDone(t *testing.T) {
	d := openTestDB(t)
	devID := testDeviceID(t, d)
	d.Queue().Enqueue("/some/file.jpg", devID)
	entry, _ := d.Queue().DequeueNext()
	if err := d.Queue().SetStatus(entry.ID, db.QueueStatusDone, nil); err != nil {
		t.Fatalf("SetStatus: %v", err)
	}
	conn := rawConn(t, d)
	var status string
	conn.QueryRow(`SELECT status FROM IngestionQueue WHERE id = ?`, entry.ID).Scan(&status)
	if status != "done" {
		t.Errorf("expected done, got %q", status)
	}
}

func TestQueue_SetStatusFailedIncrementsAttempts(t *testing.T) {
	d := openTestDB(t)
	devID := testDeviceID(t, d)
	d.Queue().Enqueue("/bad/file.jpg", devID)
	entry, _ := d.Queue().DequeueNext()
	msg := "something went wrong"
	d.Queue().SetStatus(entry.ID, db.QueueStatusFailed, &msg)
	conn := rawConn(t, d)
	var attempts *int
	conn.QueryRow(`SELECT attempt_count FROM IngestionQueue WHERE id = ?`, entry.ID).Scan(&attempts)
	if attempts == nil || *attempts != 1 {
		t.Errorf("expected attempt_count=1, got %v", attempts)
	}
}

func TestQueue_RetriesFailedBelowMaxAttempts(t *testing.T) {
	d := openTestDB(t)
	devID := testDeviceID(t, d)
	d.Queue().Enqueue("/some/file.jpg", devID)
	entry, _ := d.Queue().DequeueNext()
	msg := "transient"
	d.Queue().SetStatus(entry.ID, db.QueueStatusFailed, &msg)
	retry, err := d.Queue().DequeueNext()
	if err != nil {
		t.Fatalf("DequeueNext on retry: %v", err)
	}
	if retry == nil {
		t.Fatal("expected entry to be retried, got nil")
	}
	if retry.ID != entry.ID {
		t.Errorf("expected same entry id")
	}
}

func TestQueue_DoesNotRetryExhaustedEntry(t *testing.T) {
	d := openTestDB(t)
	devID := testDeviceID(t, d)
	d.Queue().Enqueue("/some/file.jpg", devID)
	msg := "persistent"
	for range db.MaxAttempts {
		entry, _ := d.Queue().DequeueNext()
		if entry == nil {
			t.Fatal("expected entry during retry loop")
		}
		d.Queue().SetStatus(entry.ID, db.QueueStatusFailed, &msg)
	}
	entry, err := d.Queue().DequeueNext()
	if err != nil {
		t.Fatalf("DequeueNext after exhaustion: %v", err)
	}
	if entry != nil {
		t.Errorf("expected nil after max attempts, got %+v", entry)
	}
}

func TestQueue_PendingCount(t *testing.T) {
	d := openTestDB(t)
	devID := testDeviceID(t, d)
	d.Queue().Enqueue("/f1.jpg", devID)
	d.Queue().Enqueue("/f2.jpg", devID)
	d.Queue().Enqueue("/f3.jpg", devID)
	count, err := d.Queue().PendingCount()
	if err != nil {
		t.Fatalf("PendingCount: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3, got %d", count)
	}
}
