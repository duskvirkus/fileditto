package ingestion_test

import (
	"bytes"
	"database/sql"
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

func testMediaID(t *testing.T, conn *sql.DB) string {
	t.Helper()
	// Use a temp dir that exists on the local filesystem so ghw can detect the disk.
	mediaID, err := ingestion.EnsureDriveForPath(conn, t.TempDir())
	if err != nil {
		t.Fatalf("EnsureDriveForPath: %v", err)
	}
	return mediaID
}

func TestProcessNext_ReturnsFalseWhenQueueEmpty(t *testing.T) {
	conn := openTestDB(t)
	mediaID := testMediaID(t, conn)
	processed, err := ingestion.ProcessNext(conn, mediaID)
	if err != nil {
		t.Fatalf("ProcessNext: %v", err)
	}
	if processed {
		t.Error("expected false (empty queue), got true")
	}
}

func TestProcessNext_PhotoFileWrittenToFilesAndPhoto(t *testing.T) {
	conn := openTestDB(t)
	devID := testDeviceID(t, conn)
	mediaID := testMediaID(t, conn)

	imgPath := filepath.Join(t.TempDir(), "test.png")
	writePNG(t, imgPath)

	ingestion.Enqueue(conn, imgPath, devID)

	processed, err := ingestion.ProcessNext(conn, mediaID)
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

	var locCount int
	conn.QueryRow(`SELECT COUNT(*) FROM Locations WHERE media_id = ? AND skip_for_counting = 1`, mediaID).Scan(&locCount)
	if locCount != 1 {
		t.Errorf("expected 1 Location row, got %d", locCount)
	}

	var status string
	conn.QueryRow(`SELECT status FROM IngestionQueue WHERE file_path = ?`, imgPath).Scan(&status)
	if status != "done" {
		t.Errorf("expected queue status=done, got %q", status)
	}
}

func TestProcessNext_UnsupportedFileSetToUnsupported(t *testing.T) {
	conn := openTestDB(t)
	devID := testDeviceID(t, conn)
	mediaID := testMediaID(t, conn)

	txtPath := filepath.Join(t.TempDir(), "notes.txt")
	os.WriteFile(txtPath, []byte("hello"), 0644)

	ingestion.Enqueue(conn, txtPath, devID)
	ingestion.ProcessNext(conn, mediaID)

	var status string
	conn.QueryRow(`SELECT status FROM IngestionQueue WHERE file_path = ?`, txtPath).Scan(&status)
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
	devID := testDeviceID(t, conn)
	mediaID := testMediaID(t, conn)

	imgPath := filepath.Join(t.TempDir(), "test.png")
	writePNG(t, imgPath)

	ingestion.Enqueue(conn, imgPath, devID)
	ingestion.ProcessNext(conn, mediaID)

	// Second enqueue of the same path is a no-op due to UNIQUE(file_path, device_id).
	ingestion.Enqueue(conn, imgPath, devID)
	ingestion.ProcessNext(conn, mediaID)

	var fileCount int
	conn.QueryRow(`SELECT COUNT(*) FROM Files`).Scan(&fileCount)
	if fileCount != 1 {
		t.Errorf("expected 1 Files row (dedup), got %d", fileCount)
	}

	var queueCount int
	conn.QueryRow(`SELECT COUNT(*) FROM IngestionQueue`).Scan(&queueCount)
	if queueCount != 1 {
		t.Errorf("expected 1 queue entry (duplicate path ignored), got %d", queueCount)
	}

	var doneCount int
	conn.QueryRow(
		`SELECT COUNT(*) FROM IngestionQueue WHERE status = 'done'`,
	).Scan(&doneCount)
	if doneCount != 1 {
		t.Errorf("expected 1 done queue entry, got %d", doneCount)
	}
}
