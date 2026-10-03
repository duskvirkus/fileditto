package postgres

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/duskvirkus/fileditto/internal/db"
	"github.com/duskvirkus/fileditto/internal/hardware"
)

type mediaRepository struct {
	db *sql.DB
}

func (r *mediaRepository) EnsureDriveForPath(path string, deviceID string) (string, error) {
	info, err := hardware.DetectDiskForPath(path)
	if err != nil {
		return "", fmt.Errorf("detect disk: %w", err)
	}

	mediaID := driveMediaID(info)
	now := time.Now().UTC().Format(time.RFC3339)

	var existing string
	err = r.db.QueryRow(`SELECT id FROM Media WHERE id = $1`, mediaID).Scan(&existing)
	if err == nil {
		_, err = r.db.Exec(
			`UPDATE Media SET status = 'connected', device_id = $1, updated_at = $2 WHERE id = $3`,
			deviceID, now, mediaID,
		)
		return mediaID, err
	}
	if err != sql.ErrNoRows {
		return "", fmt.Errorf("check drive: %w", err)
	}

	hostname, _ := os.Hostname()
	label := hostname + ":" + info.Model
	if info.Model == "" {
		label = hostname
	}

	tx, err := r.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.Exec(
		`INSERT INTO Media (id, media_type, label, status, device_id, created_at, updated_at)
		 VALUES ($1, 'drive', $2, 'connected', $3, $4, $5)`,
		mediaID, label, deviceID, now, now,
	); err != nil {
		return "", fmt.Errorf("insert Media: %w", err)
	}

	if _, err := tx.Exec(
		`INSERT INTO DriveMedia (id, serial_number, make, model, capacity_gb, interface_type)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		mediaID,
		nullIfEmpty(info.Serial),
		nullIfEmpty(info.Vendor),
		nullIfEmpty(info.Model),
		nullIfZero(info.CapacityGB),
		nullIfEmpty(info.InterfaceType),
	); err != nil {
		return "", fmt.Errorf("insert DriveMedia: %w", err)
	}

	return mediaID, tx.Commit()
}

func driveMediaID(info *hardware.DiskInfo) string {
	if info.Serial != "" {
		return db.NameUUID("drive-serial:" + info.Serial).String()
	}
	hostname, _ := os.Hostname()
	return db.NameUUID("drive-host:" + hostname).String()
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func nullIfZero(f float64) interface{} {
	if f == 0 {
		return nil
	}
	return f
}
