package ingestion

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jaypipes/ghw"
	"github.com/jaypipes/ghw/pkg/block"

	"github.com/duskvirkus/pxvault/internal/db"
)

// diskInfo holds detected attributes for a physical block device.
type diskInfo struct {
	serial        string // empty if undetectable
	model         string
	vendor        string
	capacityGB    float64
	interfaceType string
}

// EnsureDriveForPath returns the Media ID for the physical drive that contains
// the given path, creating Media + DriveMedia records if they don't exist.
// On each call the drive status is updated to 'connected'.
func EnsureDriveForPath(sqlDB *sql.DB, path string) (string, error) {
	info, err := detectDiskForPath(path)
	if err != nil {
		return "", fmt.Errorf("detect disk: %w", err)
	}

	mediaID, err := driveMediaID(info)
	if err != nil {
		return "", err
	}

	now := time.Now().UTC().Format(time.RFC3339)

	var existing string
	err = sqlDB.QueryRow(`SELECT id FROM Media WHERE id = ?`, mediaID).Scan(&existing)
	if err == nil {
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

	hostname, _ := os.Hostname()
	label := hostname + ":" + info.model
	if info.model == "" {
		label = hostname
	}

	tx, err := sqlDB.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.Exec(
		`INSERT INTO Media (id, media_type, label, status, created_at, updated_at)
		 VALUES (?, 'drive', ?, 'connected', ?, ?)`,
		mediaID, label, now, now,
	); err != nil {
		return "", fmt.Errorf("insert Media: %w", err)
	}

	var serialArg interface{}
	if info.serial != "" {
		serialArg = info.serial
	}
	var capacityArg interface{}
	if info.capacityGB > 0 {
		capacityArg = info.capacityGB
	}

	if _, err := tx.Exec(
		`INSERT INTO DriveMedia (id, serial_number, make, model, capacity_gb, interface_type)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		mediaID, serialArg, nullIfEmpty(info.vendor), nullIfEmpty(info.model), capacityArg, nullIfEmpty(info.interfaceType),
	); err != nil {
		return "", fmt.Errorf("insert DriveMedia: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	return mediaID, nil
}

// driveMediaID returns a stable UUID for a drive. Serial number is preferred
// as the unique key; falls back to hostname when unavailable.
func driveMediaID(info *diskInfo) (string, error) {
	if info.serial != "" {
		return db.NameUUID("drive-serial:" + info.serial).String(), nil
	}
	hostname, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("get hostname for fallback drive id: %w", err)
	}
	return db.NameUUID("drive-host:" + hostname).String(), nil
}

// detectDiskForPath finds the physical disk backing the given path by
// matching mount points from ghw against the absolute path.
func detectDiskForPath(path string) (*diskInfo, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("abs path: %w", err)
	}

	b, err := ghw.Block()
	if err != nil {
		return nil, fmt.Errorf("ghw block: %w", err)
	}

	disk, err := findDiskForPath(b.Disks, abs)
	if err != nil {
		return nil, err
	}

	vendor := disk.Vendor
	if vendor == "unknown" {
		vendor = ""
	}

	iface := interfaceString(disk.StorageController)

	return &diskInfo{
		serial:        strings.TrimSpace(disk.SerialNumber),
		model:         strings.TrimSpace(disk.Model),
		vendor:        vendor,
		capacityGB:    float64(disk.SizeBytes) / 1e9,
		interfaceType: iface,
	}, nil
}

// findDiskForPath returns the disk whose partition has the longest mount-point
// prefix match for the given absolute path.
func findDiskForPath(disks []*block.Disk, absPath string) (*block.Disk, error) {
	var bestDisk *block.Disk
	bestLen := -1

	for _, disk := range disks {
		for _, part := range disk.Partitions {
			mp := part.MountPoint
			if mp == "" {
				continue
			}
			// Ensure we match on a directory boundary.
			if mp != "/" && !strings.HasSuffix(mp, "/") {
				mp += "/"
			}
			candidate := absPath
			if !strings.HasSuffix(candidate, "/") {
				candidate += "/"
			}
			if strings.HasPrefix(candidate, mp) && len(mp) > bestLen {
				bestLen = len(mp)
				bestDisk = disk
			}
		}
	}

	if bestDisk == nil {
		return nil, fmt.Errorf("no disk found for path %s", absPath)
	}
	return bestDisk, nil
}

func interfaceString(sc block.StorageController) string {
	s := sc.String()
	if s == "unknown" {
		return ""
	}
	return s
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
