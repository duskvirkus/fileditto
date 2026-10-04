package scanner_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/duskvirkus/fileditto/internal/db"
	"github.com/duskvirkus/fileditto/internal/db/sqlite"
	"github.com/duskvirkus/fileditto/internal/scanner"
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

func testDeviceID(t *testing.T, d db.DB) string {
	t.Helper()
	id, err := d.Devices().EnsureLocal()
	if err != nil {
		t.Fatalf("EnsureLocal: %v", err)
	}
	return id
}

func TestDiscover_EnqueuesAllFiles(t *testing.T) {
	d := openTestDB(t)
	devID := testDeviceID(t, d)
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

	if err := scanner.Discover(d, dir, devID); err != nil {
		t.Fatalf("Discover: %v", err)
	}

	rc := d.(db.RawConner).RawConn()
	var count int
	rc.QueryRow(`SELECT COUNT(*) FROM IngestionQueue`).Scan(&count)
	if count != 4 {
		t.Errorf("expected 4 queue entries, got %d", count)
	}
}

func TestDiscover_AllEntriesArePending(t *testing.T) {
	d := openTestDB(t)
	devID := testDeviceID(t, d)
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("data"), 0644)
	scanner.Discover(d, dir, devID)

	rc := d.(db.RawConner).RawConn()
	var status string
	rc.QueryRow(`SELECT status FROM IngestionQueue`).Scan(&status)
	if status != "pending" {
		t.Errorf("expected pending, got %q", status)
	}
}

func TestDiscover_SkipsDirectories(t *testing.T) {
	d := openTestDB(t)
	devID := testDeviceID(t, d)
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "subdir"), 0755)
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("data"), 0644)
	scanner.Discover(d, dir, devID)

	rc := d.(db.RawConner).RawConn()
	var count int
	rc.QueryRow(`SELECT COUNT(*) FROM IngestionQueue`).Scan(&count)
	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}
}

func TestDiscover_ReturnsErrorForNonexistentPath(t *testing.T) {
	d := openTestDB(t)
	devID := testDeviceID(t, d)
	err := scanner.Discover(d, "/nonexistent/path", devID)
	if err == nil {
		t.Error("expected error for nonexistent path, got nil")
	}
}
