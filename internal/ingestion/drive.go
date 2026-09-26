package ingestion

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/duskvirkus/pxvault/internal/db"
)

// EnsureLocalDrive returns the Media ID for the local hard drive,
// creating the Media + DriveMedia records if they don't exist.
// The drive is identified by hostname and marked 'connected' on each call.
func EnsureLocalDrive(sqlDB *sql.DB) (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("get hostname: %w", err)
	}

	mediaID := db.NameUUID("local-drive:" + hostname).String()

	var existing string
	err = sqlDB.QueryRow(`SELECT id FROM Media WHERE id = ?`, mediaID).Scan(&existing)
	if err == nil {
		now := time.Now().UTC().Format(time.RFC3339)
		if _, err := sqlDB.Exec(
			`UPDATE Media SET status = 'connected', updated_at = ? WHERE id = ?`,
			now, mediaID,
		); err != nil {
			return "", fmt.Errorf("update drive status: %w", err)
		}
		return mediaID, nil
	}
	if err != sql.ErrNoRows {
		return "", fmt.Errorf("check drive: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := sqlDB.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.Exec(
		`INSERT INTO Media (id, media_type, label, status, created_at, updated_at) VALUES (?, 'drive', ?, 'connected', ?, ?)`,
		mediaID, hostname, now, now,
	); err != nil {
		return "", fmt.Errorf("insert Media: %w", err)
	}

	if _, err := tx.Exec(`INSERT INTO DriveMedia (id) VALUES (?)`, mediaID); err != nil {
		return "", fmt.Errorf("insert DriveMedia: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	return mediaID, nil
}
