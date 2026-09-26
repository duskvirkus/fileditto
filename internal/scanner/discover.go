package scanner

import (
	"database/sql"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/duskvirkus/fileditto/internal/ingestion"
)

// Discover walks root and enqueues every file it finds, without filtering.
// Directories are skipped. All file types are enqueued — classification
// happens in the ingestion worker.
func Discover(db *sql.DB, root string, deviceID int64) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk %s: %w", path, err)
		}
		if d.IsDir() {
			return nil
		}
		return ingestion.Enqueue(db, path, deviceID)
	})
}
