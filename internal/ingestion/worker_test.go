package ingestion_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/duskvirkus/fileditto/internal/db"
	"github.com/duskvirkus/fileditto/internal/ingestion"
)

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

func testMediaID(t *testing.T, d db.DB) string {
	t.Helper()
	devID := testDeviceID(t, d)
	mediaID, err := d.Media().EnsureDriveForPath(t.TempDir(), devID)
	if err != nil {
		t.Fatalf("EnsureDriveForPath: %v", err)
	}
	return mediaID
}

func TestProcessNext_ReturnsFalseWhenQueueEmpty(t *testing.T) {
	d := openTestDB(t)
	mediaID := testMediaID(t, d)
	processed, err := ingestion.ProcessNext(d, mediaID)
	if err != nil {
		t.Fatalf("ProcessNext: %v", err)
	}
	if processed {
		t.Error("expected false (empty queue), got true")
	}
}

func TestProcessNext_PhotoFileWrittenToFilesAndPhoto(t *testing.T) {
	d := openTestDB(t)
	devID := testDeviceID(t, d)
	mediaID := testMediaID(t, d)

	imgPath := filepath.Join(t.TempDir(), "test.png")
	writePNG(t, imgPath)
	d.Queue().Enqueue(imgPath, devID)

	processed, err := ingestion.ProcessNext(d, mediaID)
	if err != nil {
		t.Fatalf("ProcessNext: %v", err)
	}
	if !processed {
		t.Fatal("expected processed=true")
	}

	rc := d.(db.RawConner).RawConn()
	var fileCount int
	rc.QueryRow(`SELECT COUNT(*) FROM Files WHERE file_type = 'photo'`).Scan(&fileCount)
	if fileCount != 1 {
		t.Errorf("expected 1 Files row, got %d", fileCount)
	}

	var photoCount int
	rc.QueryRow(`SELECT COUNT(*) FROM Photo`).Scan(&photoCount)
	if photoCount != 1 {
		t.Errorf("expected 1 Photo row, got %d", photoCount)
	}

	var locCount int
	rc.QueryRow(`SELECT COUNT(*) FROM Locations WHERE media_id = ?`, mediaID).Scan(&locCount)
	if locCount != 1 {
		t.Errorf("expected 1 Location row, got %d", locCount)
	}

	var status string
	rc.QueryRow(`SELECT status FROM IngestionQueue WHERE file_path = ?`, imgPath).Scan(&status)
	if status != "done" {
		t.Errorf("expected queue status=done, got %q", status)
	}
}

func TestProcessNext_UnsupportedFileSetToUnsupported(t *testing.T) {
	d := openTestDB(t)
	devID := testDeviceID(t, d)
	mediaID := testMediaID(t, d)

	txtPath := filepath.Join(t.TempDir(), "notes.txt")
	os.WriteFile(txtPath, []byte("hello"), 0644)
	d.Queue().Enqueue(txtPath, devID)
	ingestion.ProcessNext(d, mediaID)

	rc := d.(db.RawConner).RawConn()
	var status string
	rc.QueryRow(`SELECT status FROM IngestionQueue WHERE file_path = ?`, txtPath).Scan(&status)
	if status != "unsupported" {
		t.Errorf("expected unsupported, got %q", status)
	}
}

func TestProcessNext_DuplicateFileNotReinserted(t *testing.T) {
	d := openTestDB(t)
	devID := testDeviceID(t, d)
	mediaID := testMediaID(t, d)

	imgPath := filepath.Join(t.TempDir(), "test.png")
	writePNG(t, imgPath)
	d.Queue().Enqueue(imgPath, devID)
	ingestion.ProcessNext(d, mediaID)

	d.Queue().Enqueue(imgPath, devID)
	ingestion.ProcessNext(d, mediaID)

	rc := d.(db.RawConner).RawConn()
	var fileCount int
	rc.QueryRow(`SELECT COUNT(*) FROM Files`).Scan(&fileCount)
	if fileCount != 1 {
		t.Errorf("expected 1 Files row (dedup), got %d", fileCount)
	}
}
