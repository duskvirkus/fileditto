// Package scanner walks directory trees and enqueues files for ingestion.
package scanner

import (
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/duskvirkus/fileditto/internal/db"
)

// Discover walks root and enqueues every file it finds.
// Directories are skipped; classification happens in the ingestion worker.
func Discover(d db.DB, root string, deviceID string) error {
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk %s: %w", path, err)
		}
		if entry.IsDir() {
			return nil
		}
		return d.Queue().Enqueue(path, deviceID)
	})
}
