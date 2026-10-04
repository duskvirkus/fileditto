// Package db defines the database interface, domain types, and shared migration
// infrastructure used by both the SQLite and PostgreSQL implementations.
package db

import (
	"database/sql"
	"embed"
)

// Dialect identifies the database backend.
type Dialect string

const (
	// SQLite selects the SQLite backend.
	SQLite Dialect = "sqlite"
	// PostgreSQL selects the PostgreSQL backend.
	PostgreSQL Dialect = "postgres"
)

// MigrationsFS embeds the SQL migration files for all dialects.
//
//go:embed migrations/shared all:migrations/sqlite all:migrations/postgres
var MigrationsFS embed.FS

// MaxAttempts is the number of processing attempts before a queue entry is
// left permanently failed.
const MaxAttempts = 3

// DB is the top-level database handle. It provides access to each domain
// repository and manages migrations and lifecycle.
type DB interface {
	Queue()   QueueRepository
	Files()   FileRepository
	Devices() DeviceRepository
	Media()   MediaRepository
	Migrate() error
	Close()   error
}

// QueueStatus is the lifecycle state of an IngestionQueue entry.
type QueueStatus string

const (
	QueueStatusPending     QueueStatus = "pending"
	QueueStatusProcessing  QueueStatus = "processing"
	QueueStatusDone        QueueStatus = "done"
	QueueStatusFailed      QueueStatus = "failed"
	QueueStatusUnsupported QueueStatus = "unsupported"
)

// QueueEntry is a row from IngestionQueue.
type QueueEntry struct {
	ID           string
	FilePath     string
	DeviceID     string
	Status       QueueStatus
	AttemptCount *int
	Error        *string
	CreatedAt    string
	UpdatedAt    string
}

// File is a row from the Files table.
type File struct {
	ID        string
	SHA256    string
	SizeBytes int64
	FileType  string
	CreatedAt string
	UpdatedAt string
}

// Photo is a row from the Photo table.
type Photo struct {
	WidthPx      int
	HeightPx     int
	ColorProfile string
	MetadataJSON []byte
}

// Video is a row from the Video table.
type Video struct {
	DurationSeconds float64
	WidthPx         int
	HeightPx        int
	FrameRate       float64
	Codec           string
	MetadataJSON    []byte
}

// QueueRepository manages the IngestionQueue table.
type QueueRepository interface {
	// Enqueue adds filePath to the queue for deviceID. A duplicate path+device
	// pair is silently ignored.
	Enqueue(filePath string, deviceID string) error
	// DequeueNext atomically claims the next pending (or retryable failed) entry
	// and marks it processing. Returns nil when the queue is empty.
	DequeueNext() (*QueueEntry, error)
	// SetStatus updates the status of entry id. When status is Failed,
	// attempt_count is incremented and errMsg is stored.
	SetStatus(id string, status QueueStatus, errMsg *string) error
	// ResetStuck resets all processing entries back to pending and returns the
	// count. Used at startup to recover from a previous crash.
	ResetStuck() (int64, error)
	// PendingCount returns the number of entries with status pending.
	PendingCount() (int64, error)
}

// FileRepository manages Files, Photo, Video, and Locations tables.
type FileRepository interface {
	// WriteFile atomically inserts a file with its type metadata and a Location
	// record. If a file with the same SHA256 already exists, returns
	// isNew=false and makes no changes. photo and video may be nil when not
	// applicable to the file type.
	WriteFile(f *File, photo *Photo, video *Video, mediaID, pathOnMedia string) (id string, isNew bool, err error)
}

// DeviceRepository manages the Devices table.
type DeviceRepository interface {
	// EnsureLocal upserts a row for the current hostname and returns its ID.
	EnsureLocal() (string, error)
}

// MediaRepository manages Media and DriveMedia tables.
type MediaRepository interface {
	// EnsureDriveForPath detects the physical disk backing path, upserts a
	// Media+DriveMedia row for it, and returns the media ID.
	EnsureDriveForPath(path string, deviceID string) (string, error)
}

// RawConner is implemented by concrete DB types to expose the underlying
// *sql.DB for use in tests only.
type RawConner interface {
	RawConn() *sql.DB
}
