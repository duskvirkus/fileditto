package ingestion_test

import (
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

func testDeviceID(t *testing.T, d db.DB) string {
	t.Helper()
	id, err := d.Devices().EnsureLocal()
	if err != nil {
		t.Fatalf("EnsureLocal: %v", err)
	}
	return id
}

func TestEnqueue_CreatesPendingEntry(t *testing.T) {
	d := openTestDB(t)
	devID := testDeviceID(t, d)
	if err := d.Queue().Enqueue("/some/file.jpg", devID); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	rc := d.(db.RawConner).RawConn()
	var status string
	rc.QueryRow(`SELECT status FROM IngestionQueue WHERE file_path = ?`, "/some/file.jpg").Scan(&status)
	if status != "pending" {
		t.Errorf("expected pending, got %q", status)
	}
}

func TestDequeueNext_ReturnsNilWhenEmpty(t *testing.T) {
	d := openTestDB(t)
	entry, err := d.Queue().DequeueNext()
	if err != nil {
		t.Fatalf("DequeueNext: %v", err)
	}
	if entry != nil {
		t.Errorf("expected nil, got %+v", entry)
	}
}

func TestDequeueNext_MarksEntryAsProcessing(t *testing.T) {
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

func TestSetStatus_Done(t *testing.T) {
	d := openTestDB(t)
	devID := testDeviceID(t, d)
	d.Queue().Enqueue("/some/file.jpg", devID)
	entry, _ := d.Queue().DequeueNext()
	if err := d.Queue().SetStatus(entry.ID, db.QueueStatusDone, nil); err != nil {
		t.Fatalf("SetStatus: %v", err)
	}
	rc := d.(db.RawConner).RawConn()
	var status string
	rc.QueryRow(`SELECT status FROM IngestionQueue WHERE id = ?`, entry.ID).Scan(&status)
	if status != "done" {
		t.Errorf("expected done, got %q", status)
	}
}

func TestSetStatus_FailedIncrementsAttemptCount(t *testing.T) {
	d := openTestDB(t)
	devID := testDeviceID(t, d)
	d.Queue().Enqueue("/bad/file.jpg", devID)
	entry, _ := d.Queue().DequeueNext()
	errMsg := "something went wrong"
	d.Queue().SetStatus(entry.ID, db.QueueStatusFailed, &errMsg)
	rc := d.(db.RawConner).RawConn()
	var attempts *int
	rc.QueryRow(`SELECT attempt_count FROM IngestionQueue WHERE id = ?`, entry.ID).Scan(&attempts)
	if attempts == nil || *attempts != 1 {
		t.Errorf("expected attempt_count=1, got %v", attempts)
	}
}

func TestDequeueNext_RetriesFailedEntryBelowMaxAttempts(t *testing.T) {
	d := openTestDB(t)
	devID := testDeviceID(t, d)
	d.Queue().Enqueue("/some/file.jpg", devID)
	entry, _ := d.Queue().DequeueNext()
	errMsg := "transient error"
	d.Queue().SetStatus(entry.ID, db.QueueStatusFailed, &errMsg)
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

func TestDequeueNext_DoesNotRetryExhaustedEntry(t *testing.T) {
	d := openTestDB(t)
	devID := testDeviceID(t, d)
	d.Queue().Enqueue("/some/file.jpg", devID)
	errMsg := "persistent error"
	for range db.MaxAttempts {
		entry, _ := d.Queue().DequeueNext()
		if entry == nil {
			t.Fatal("expected entry during retry loop, got nil")
		}
		d.Queue().SetStatus(entry.ID, db.QueueStatusFailed, &errMsg)
	}
	entry, err := d.Queue().DequeueNext()
	if err != nil {
		t.Fatalf("DequeueNext after exhaustion: %v", err)
	}
	if entry != nil {
		t.Errorf("expected nil after max attempts, got %+v", entry)
	}
}

func TestPendingCount(t *testing.T) {
	d := openTestDB(t)
	devID := testDeviceID(t, d)
	d.Queue().Enqueue("/file1.jpg", devID)
	d.Queue().Enqueue("/file2.jpg", devID)
	d.Queue().Enqueue("/file3.jpg", devID)
	count, err := d.Queue().PendingCount()
	if err != nil {
		t.Fatalf("PendingCount: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3, got %d", count)
	}
}
