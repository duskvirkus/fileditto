package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/duskvirkus/fileditto/internal/db"
)

type deviceRepository struct {
	db *sql.DB
}

func (r *deviceRepository) EnsureLocal() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("get hostname: %w", err)
	}

	var id string
	err = r.db.QueryRow(`SELECT id FROM Devices WHERE name = ?`, hostname).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return "", fmt.Errorf("check device: %w", err)
	}

	id = db.NameUUID("device:" + hostname).String()
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := r.db.Exec(
		`INSERT INTO Devices (id, name, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		id, hostname, now, now,
	); err != nil {
		return "", fmt.Errorf("insert device: %w", err)
	}
	return id, nil
}
