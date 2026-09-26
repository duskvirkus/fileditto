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
