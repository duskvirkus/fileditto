package ingestion

import (
	"database/sql"
	"fmt"
	"os"
	"time"
)

// EnsureLocalDevice returns the Devices row ID for the local machine,
// creating it with the hostname as the name if it doesn't already exist.
func EnsureLocalDevice(sqlDB *sql.DB) (int64, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return 0, fmt.Errorf("get hostname: %w", err)
	}

	var id int64
	err = sqlDB.QueryRow(`SELECT id FROM Devices WHERE name = ?`, hostname).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("check device: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := sqlDB.Exec(
		`INSERT INTO Devices (name, created_at, updated_at) VALUES (?, ?, ?)`,
		hostname, now, now,
	)
	if err != nil {
		return 0, fmt.Errorf("insert Device: %w", err)
	}
	return res.LastInsertId()
}
