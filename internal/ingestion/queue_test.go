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
