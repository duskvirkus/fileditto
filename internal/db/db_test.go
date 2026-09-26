package db_test

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/duskvirkus/pxvault/internal/db"
)

// openTestDB opens a temporary SQLite database, runs migrations, and returns it.
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

// --- 6.1 schema_version ---

func TestMigration_SchemaVersion(t *testing.T) {
	conn := openTestDB(t)

	var version int
	if err := conn.QueryRow("SELECT version FROM schema_version").Scan(&version); err != nil {
		t.Fatalf("query schema_version: %v", err)
	}
	if version != 1 {
		t.Errorf("expected schema_version = 1, got %d", version)
	}
}

// --- 6.2 Photo CASCADE ---

func TestCascade_PhotoDeletedWithFile(t *testing.T) {
	conn := openTestDB(t)

	fileID := "file-photo-cascade-test"
	sha256 := "aabbccdd" + "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"[8:]
	_, err := conn.Exec(
		`INSERT INTO Files (id, sha256, size_bytes, file_type) VALUES (?, ?, ?, ?)`,
		fileID, sha256, 1024, "photo",
	)
	if err != nil {
		t.Fatalf("insert Files: %v", err)
	}

	_, err = conn.Exec(`INSERT INTO Photo (id, width_px, height_px) VALUES (?, ?, ?)`, fileID, 1920, 1080)
	if err != nil {
		t.Fatalf("insert Photo: %v", err)
	}

	if _, err := conn.Exec("DELETE FROM Files WHERE id = ?", fileID); err != nil {
		t.Fatalf("delete Files: %v", err)
	}

	var count int
	conn.QueryRow("SELECT COUNT(*) FROM Photo WHERE id = ?", fileID).Scan(&count)
	if count != 0 {
		t.Errorf("expected Photo row to be deleted via CASCADE, but it still exists")
	}
}

// --- 6.3 Video CASCADE ---

func TestCascade_VideoDeletedWithFile(t *testing.T) {
	conn := openTestDB(t)

	fileID := "file-video-cascade-test"
	sha256 := "bbccdd00" + "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"[8:]
	_, err := conn.Exec(
		`INSERT INTO Files (id, sha256, size_bytes, file_type) VALUES (?, ?, ?, ?)`,
		fileID, sha256, 2048, "video",
	)
	if err != nil {
		t.Fatalf("insert Files: %v", err)
	}

	_, err = conn.Exec(
		`INSERT INTO Video (id, duration_seconds, width_px, height_px) VALUES (?, ?, ?, ?)`,
		fileID, 120.5, 3840, 2160,
	)
	if err != nil {
		t.Fatalf("insert Video: %v", err)
	}

	if _, err := conn.Exec("DELETE FROM Files WHERE id = ?", fileID); err != nil {
		t.Fatalf("delete Files: %v", err)
	}

	var count int
	conn.QueryRow("SELECT COUNT(*) FROM Video WHERE id = ?", fileID).Scan(&count)
	if count != 0 {
		t.Errorf("expected Video row to be deleted via CASCADE, but it still exists")
	}
}

// --- 6.4 Locations CASCADE when Media deleted ---

func TestCascade_LocationsDeletedWithMedia(t *testing.T) {
	conn := openTestDB(t)

	fileID := "file-loc-cascade"
	sha256 := "ccdd0011" + "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"[8:]
	conn.Exec(
		`INSERT INTO Files (id, sha256, size_bytes, file_type) VALUES (?, ?, ?, ?)`,
		fileID, sha256, 512, "generic",
	)

	mediaID := "media-loc-cascade"
	conn.Exec(
		`INSERT INTO Media (id, media_type, status) VALUES (?, ?, ?)`,
		mediaID, "drive", "connected",
	)

	locID := "loc-cascade"
	_, err := conn.Exec(
		`INSERT INTO Locations (id, file_id, media_id, path_on_media, status) VALUES (?, ?, ?, ?, ?)`,
		locID, fileID, mediaID, "/files/photo.jpg", "healthy",
	)
	if err != nil {
		t.Fatalf("insert Locations: %v", err)
	}

	if _, err := conn.Exec("DELETE FROM Media WHERE id = ?", mediaID); err != nil {
		t.Fatalf("delete Media: %v", err)
	}

	var count int
	conn.QueryRow("SELECT COUNT(*) FROM Locations WHERE id = ?", locID).Scan(&count)
	if count != 0 {
		t.Errorf("expected Locations row to be deleted via CASCADE, but it still exists")
	}
}

// --- 6.5 invalid file_type rejected ---

func TestConstraint_InvalidFileTypeRejected(t *testing.T) {
	conn := openTestDB(t)

	_, err := conn.Exec(
		`INSERT INTO Files (id, sha256, size_bytes, file_type) VALUES (?, ?, ?, ?)`,
		"bad-type", "deadbeef00000000000000000000000000000000000000000000000000000000", 100, "document",
	)
	if err == nil {
		t.Error("expected CHECK constraint violation for invalid file_type, but insert succeeded")
	}
}

// --- 6.6 invalid Media.status rejected ---

func TestConstraint_InvalidMediaStatusRejected(t *testing.T) {
	conn := openTestDB(t)

	_, err := conn.Exec(
		`INSERT INTO Media (id, media_type, status) VALUES (?, ?, ?)`,
		"bad-media-status", "drive", "destroyed",
	)
	if err == nil {
		t.Error("expected CHECK constraint violation for invalid Media.status, but insert succeeded")
	}
}

// --- 6.7 invalid Locations.status rejected ---

func TestConstraint_InvalidLocationsStatusRejected(t *testing.T) {
	conn := openTestDB(t)

	fileID := "file-loc-bad-status"
	sha256 := "eeff0011" + "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"[8:]
	conn.Exec(
		`INSERT INTO Files (id, sha256, size_bytes, file_type) VALUES (?, ?, ?, ?)`,
		fileID, sha256, 256, "generic",
	)
	mediaID := "media-loc-bad-status"
	conn.Exec(`INSERT INTO Media (id, media_type, status) VALUES (?, ?, ?)`, mediaID, "drive", "connected")

	_, err := conn.Exec(
		`INSERT INTO Locations (id, file_id, media_id, path_on_media, status) VALUES (?, ?, ?, ?, ?)`,
		"loc-bad", fileID, mediaID, "/file.dat", "missing",
	)
	if err == nil {
		t.Error("expected CHECK constraint violation for invalid Locations.status, but insert succeeded")
	}
}

// --- 6.8 UUID v5 determinism ---

func TestUUIDv5_Deterministic(t *testing.T) {
	hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	u1, err := db.FileUUID(hash)
	if err != nil {
		t.Fatalf("FileUUID first call: %v", err)
	}
	u2, err := db.FileUUID(hash)
	if err != nil {
		t.Fatalf("FileUUID second call: %v", err)
	}
	if u1 != u2 {
		t.Errorf("expected identical UUIDs, got %v and %v", u1, u2)
	}
}

// --- 6.11 checksum mismatch halts runner ---

func TestMigrationRunner_ChecksumMismatchHaltsRunner(t *testing.T) {
	path := filepath.Join(t.TempDir(), "checksum_mismatch.db")
	conn, err := db.OpenDB(path)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer conn.Close()

	// Apply migrations normally first.
	if err := db.RunMigrations(conn); err != nil {
		t.Fatalf("RunMigrations (first run): %v", err)
	}

	// Corrupt the stored checksum for migration 1.
	_, err = conn.Exec(
		"UPDATE schema_migrations SET checksum = 'deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef' WHERE migration_number = 1",
	)
	if err != nil {
		t.Fatalf("corrupt checksum: %v", err)
	}

	// Running migrations again should detect the mismatch and return an error.
	if err := db.RunMigrations(conn); err == nil {
		t.Error("expected RunMigrations to return an error on checksum mismatch, but it succeeded")
	}
}

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

// --- devices table ---

func TestDevices_LocalDeviceSeeded(t *testing.T) {
	conn := openTestDB(t)

	var id int
	var name string
	if err := conn.QueryRow(`SELECT id, name FROM devices WHERE id = 0`).Scan(&id, &name); err != nil {
		t.Fatalf("expected device id=0 to exist, got: %v", err)
	}
	if name != "local" {
		t.Errorf("expected device name=local, got %q", name)
	}
}

func TestIngestionQueue_DefaultDeviceIsLocal(t *testing.T) {
	conn := openTestDB(t)

	conn.Exec(
		`INSERT INTO ingestion_queue (file_path, status, created_at, updated_at)
		 VALUES (?, 'pending', ?, ?)`,
		"/some/file.jpg", "2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z",
	)

	var deviceID int
	conn.QueryRow(`SELECT device_id FROM ingestion_queue WHERE file_path = ?`, "/some/file.jpg").Scan(&deviceID)
	if deviceID != 0 {
		t.Errorf("expected device_id=0 (local), got %d", deviceID)
	}
}

func TestIngestionQueue_InvalidDeviceRejected(t *testing.T) {
	conn := openTestDB(t)

	_, err := conn.Exec(
		`INSERT INTO ingestion_queue (file_path, device_id, status, created_at, updated_at)
		 VALUES (?, 999, 'pending', ?, ?)`,
		"/some/file.jpg", "2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z",
	)
	if err == nil {
		t.Error("expected foreign key violation for unknown device_id, but insert succeeded")
	}
}
