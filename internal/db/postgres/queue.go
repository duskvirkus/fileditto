package postgres

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/duskvirkus/fileditto/internal/db"
)

type queueRepository struct {
	db *sql.DB
}

func (r *queueRepository) Enqueue(filePath string, deviceID string) error {
	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.Exec(
		`INSERT INTO IngestionQueue (id, file_path, device_id, status, created_at, updated_at)
		 VALUES ($1, $2, $3, 'pending', $4, $5)
		 ON CONFLICT (file_path, device_id) DO NOTHING`,
		id, filePath, deviceID, now, now,
	)
	if err != nil {
		return fmt.Errorf("enqueue %s: %w", filePath, err)
	}
	return nil
}

func (r *queueRepository) DequeueNext() (*db.QueueEntry, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck

	var entry db.QueueEntry
	err = tx.QueryRow(
		`SELECT id, file_path, device_id, status, attempt_count, error, created_at, updated_at
		 FROM IngestionQueue
		 WHERE status = 'pending'
		    OR (status = 'failed' AND COALESCE(attempt_count, 0) < $1)
		 ORDER BY created_at LIMIT 1`,
		db.MaxAttempts,
	).Scan(
		&entry.ID, &entry.FilePath, &entry.DeviceID, &entry.Status,
		&entry.AttemptCount, &entry.Error, &entry.CreatedAt, &entry.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("dequeue: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := tx.Exec(
		`UPDATE IngestionQueue SET status = 'processing', updated_at = $1 WHERE id = $2`,
		now, entry.ID,
	); err != nil {
		return nil, fmt.Errorf("mark processing: %w", err)
	}

	entry.Status = db.QueueStatusProcessing
	return &entry, tx.Commit()
}

func (r *queueRepository) SetStatus(id string, status db.QueueStatus, errMsg *string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	var err error
	if status == db.QueueStatusFailed {
		_, err = r.db.Exec(
			`UPDATE IngestionQueue
			 SET status = $1, error = $2, updated_at = $3,
			     attempt_count = COALESCE(attempt_count, 0) + 1
			 WHERE id = $4`,
			status, errMsg, now, id,
		)
	} else {
		_, err = r.db.Exec(
			`UPDATE IngestionQueue SET status = $1, error = $2, updated_at = $3 WHERE id = $4`,
			status, errMsg, now, id,
		)
	}
	if err != nil {
		return fmt.Errorf("set status %s for %s: %w", status, id, err)
	}
	return nil
}

func (r *queueRepository) ResetStuck() (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := r.db.Exec(
		`UPDATE IngestionQueue SET status = 'pending', updated_at = $1 WHERE status = 'processing'`,
		now,
	)
	if err != nil {
		return 0, fmt.Errorf("reset stuck: %w", err)
	}
	return res.RowsAffected()
}

func (r *queueRepository) PendingCount() (int64, error) {
	var count int64
	err := r.db.QueryRow(`SELECT COUNT(*) FROM IngestionQueue WHERE status = 'pending'`).Scan(&count)
	return count, err
}
