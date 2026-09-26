package ingestion

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/duskvirkus/pxvault/internal/db"
	"github.com/gabriel-vasile/mimetype"
	"github.com/rwcarlsen/goexif/exif"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// ProcessNext dequeues and processes one pending entry.
// Returns (true, nil) if an entry was processed, (false, nil) if the queue is empty.
func ProcessNext(sqlDB *sql.DB) (bool, error) {
	entry, err := DequeueNext(sqlDB)
	if err != nil {
		return false, err
	}
	if entry == nil {
		return false, nil
	}

	if err := processEntry(sqlDB, entry); err != nil {
		errStr := err.Error()
		if statusErr := SetStatus(sqlDB, entry.ID, StatusFailed, &errStr); statusErr != nil {
			return true, fmt.Errorf("process failed (%w); also failed to record failure: %v", err, statusErr)
		}
		return true, fmt.Errorf("process %s: %w", entry.FilePath, err)
	}
	return true, nil
}

func processEntry(sqlDB *sql.DB, entry *QueueEntry) error {
	mime, err := mimetype.DetectFile(entry.FilePath)
	if err != nil {
		return fmt.Errorf("detect mime: %w", err)
	}

	fileType := classifyMIME(mime.String())
	if fileType == "unsupported" {
		return SetStatus(sqlDB, entry.ID, StatusUnsupported, nil)
	}

	sha, size, err := hashFile(entry.FilePath)
	if err != nil {
		return err
	}

	fileUUID, err := db.FileUUID(sha)
	if err != nil {
		return fmt.Errorf("derive uuid: %w", err)
	}

	var existing string
	err = sqlDB.QueryRow(`SELECT id FROM Files WHERE sha256 = ?`, sha).Scan(&existing)
	if err == nil {
		return SetStatus(sqlDB, entry.ID, StatusDone, nil)
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("dedup check: %w", err)
	}

	meta, err := extractMetadata(entry.FilePath, fileType)
	if err != nil {
		return fmt.Errorf("extract metadata: %w", err)
	}

	if err := writeFile(sqlDB, fileUUID.String(), sha, size, fileType, meta); err != nil {
		return err
	}

	return SetStatus(sqlDB, entry.ID, StatusDone, nil)
}

func classifyMIME(mime string) string {
	switch {
	case strings.HasPrefix(mime, "image/"):
		return "photo"
	case strings.HasPrefix(mime, "video/"):
		return "video"
	default:
		return "unsupported"
	}
}

func hashFile(path string) (sha256hex string, size int64, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, fmt.Errorf("open for hash: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	size, err = io.Copy(h, f)
	if err != nil {
		return "", 0, fmt.Errorf("hash: %w", err)
	}
	return fmt.Sprintf("%x", h.Sum(nil)), size, nil
}

func extractMetadata(path, fileType string) (map[string]interface{}, error) {
	switch fileType {
	case "photo":
		return extractImageMetadata(path)
	case "video":
		return extractVideoMetadata(path)
	default:
		return map[string]interface{}{}, nil
	}
}

func extractImageMetadata(path string) (map[string]interface{}, error) {
	meta := map[string]interface{}{}

	f, err := os.Open(path)
	if err != nil {
		return meta, err
	}
	defer f.Close()

	cfg, _, err := image.DecodeConfig(f)
	if err == nil {
		meta["width"] = cfg.Width
		meta["height"] = cfg.Height
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return meta, nil
	}
	x, err := exif.Decode(f)
	if err == nil {
		if lat, long, err := x.LatLong(); err == nil {
			meta["latitude"] = lat
			meta["longitude"] = long
		}
		if tm, err := x.DateTime(); err == nil {
			meta["taken_at"] = tm.UTC().Format(time.RFC3339)
		}
	}

	return meta, nil
}

type ffprobeOut struct {
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
	Streams []struct {
		CodecType string `json:"codec_type"`
		CodecName string `json:"codec_name"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
	} `json:"streams"`
}

func extractVideoMetadata(path string) (map[string]interface{}, error) {
	out, err := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration:stream=codec_name,width,height,codec_type",
		"-of", "json",
		path,
	).Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe: %w", err)
	}

	var fp ffprobeOut
	if err := json.Unmarshal(out, &fp); err != nil {
		return nil, err
	}

	meta := map[string]interface{}{}
	if d, err := strconv.ParseFloat(fp.Format.Duration, 64); err == nil {
		meta["duration_seconds"] = d
	}
	for _, s := range fp.Streams {
		if s.CodecType == "video" {
			meta["codec"] = s.CodecName
			meta["width"] = s.Width
			meta["height"] = s.Height
			break
		}
	}
	return meta, nil
}

func writeFile(sqlDB *sql.DB, fileID, sha string, size int64, fileType string, meta map[string]interface{}) error {
	tx, err := sqlDB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := tx.Exec(
		`INSERT INTO Files (id, sha256, size_bytes, file_type, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		fileID, sha, size, fileType, now, now,
	); err != nil {
		return fmt.Errorf("insert Files: %w", err)
	}

	metaJSON, _ := json.Marshal(meta)

	switch fileType {
	case "photo":
		width, _ := meta["width"].(int)
		height, _ := meta["height"].(int)
		if _, err := tx.Exec(
			`INSERT INTO Photo (id, width_px, height_px, metadata_blob) VALUES (?, ?, ?, ?)`,
			fileID, width, height, metaJSON,
		); err != nil {
			return fmt.Errorf("insert Photo: %w", err)
		}
	case "video":
		width, _ := meta["width"].(int)
		height, _ := meta["height"].(int)
		codec, _ := meta["codec"].(string)
		durSec, _ := meta["duration_seconds"].(float64)
		if _, err := tx.Exec(
			`INSERT INTO Video (id, duration_seconds, width_px, height_px, codec, metadata_blob)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			fileID, durSec, width, height, codec, metaJSON,
		); err != nil {
			return fmt.Errorf("insert Video: %w", err)
		}
	}

	return tx.Commit()
}
