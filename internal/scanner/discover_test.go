package scanner_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/duskvirkus/fileditto/internal/db"
	"github.com/duskvirkus/fileditto/internal/ingestion"
	"github.com/duskvirkus/fileditto/internal/scanner"
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

func testDeviceID(t *testing.T, conn *sql.DB) int64 {
	t.Helper()
	id, err := ingestion.EnsureLocalDevice(conn)
	if err != nil {
		t.Fatalf("EnsureLocalDevice: %v", err)
	}
	return id
}

func TestDiscover_EnqueuesAllFiles(t *testing.T) {
	conn := openTestDB(t)
	devID := testDeviceID(t, conn)
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

	if err := scanner.Discover(conn, dir, devID); err != nil {
		t.Fatalf("Discover: %v", err)
	}

	var count int
	conn.QueryRow(`SELECT COUNT(*) FROM IngestionQueue`).Scan(&count)
	if count != 4 {
		t.Errorf("expected 4 queue entries, got %d", count)
	}
}

func TestDiscover_AllEntriesArePending(t *testing.T) {
	conn := openTestDB(t)
	devID := testDeviceID(t, conn)
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("data"), 0644)

	scanner.Discover(conn, dir, devID)

	var status string
	conn.QueryRow(`SELECT status FROM IngestionQueue`).Scan(&status)
	if status != "pending" {
		t.Errorf("expected status=pending, got %q", status)
	}
}

func TestDiscover_SkipsDirectories(t *testing.T) {
	conn := openTestDB(t)
	devID := testDeviceID(t, conn)
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "subdir"), 0755)
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("data"), 0644)

	scanner.Discover(conn, dir, devID)

	var count int
	conn.QueryRow(`SELECT COUNT(*) FROM IngestionQueue`).Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 entry (file only, not directory), got %d", count)
	}
}

func TestDiscover_ReturnsErrorForNonexistentPath(t *testing.T) {
	conn := openTestDB(t)
	devID := testDeviceID(t, conn)
	err := scanner.Discover(conn, "/nonexistent/path/that/does/not/exist", devID)
	if err == nil {
		t.Error("expected error for nonexistent path, got nil")
	}
}
