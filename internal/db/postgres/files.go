package postgres

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/duskvirkus/fileditto/internal/db"
)

type fileRepository struct {
	db *sql.DB
}

func (r *fileRepository) WriteFile(
	f *db.File,
	photo *db.Photo,
	video *db.Video,
	mediaID, pathOnMedia string,
) (string, bool, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return "", false, err
	}
	defer tx.Rollback() //nolint:errcheck

	now := time.Now().UTC().Format(time.RFC3339)

	result, err := tx.Exec(
		`INSERT INTO Files (id, sha256, size_bytes, file_type, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (sha256) DO NOTHING`,
		f.ID, f.SHA256, f.SizeBytes, f.FileType, now, now,
	)
	if err != nil {
		return "", false, fmt.Errorf("insert File: %w", err)
	}

	n, _ := result.RowsAffected()
	if n == 0 {
		return f.ID, false, tx.Commit()
	}

	switch {
	case photo != nil:
		if _, err := tx.Exec(
			`INSERT INTO Photo (id, width_px, height_px, color_profile, metadata_blob)
			 VALUES ($1, $2, $3, $4, $5)`,
			f.ID, photo.WidthPx, photo.HeightPx,
			nullIfEmpty(photo.ColorProfile), string(photo.MetadataJSON),
		); err != nil {
			return "", false, fmt.Errorf("insert Photo: %w", err)
		}
	case video != nil:
		if _, err := tx.Exec(
			`INSERT INTO Video (id, duration_seconds, width_px, height_px, frame_rate, codec, metadata_blob)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			f.ID, video.DurationSeconds, video.WidthPx, video.HeightPx,
			nullIfZero(video.FrameRate), nullIfEmpty(video.Codec), string(video.MetadataJSON),
		); err != nil {
			return "", false, fmt.Errorf("insert Video: %w", err)
		}
	}

	locID := uuid.New().String()
	if _, err := tx.Exec(
		`INSERT INTO Locations (id, file_id, media_id, path_on_media, skip_for_counting, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, 1, 'healthy', $5, $6)`,
		locID, f.ID, mediaID, pathOnMedia, now, now,
	); err != nil {
		return "", false, fmt.Errorf("insert Location: %w", err)
	}

	return f.ID, true, tx.Commit()
}
