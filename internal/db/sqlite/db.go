// Package sqlite implements the db.DB interface using a local SQLite file.
package sqlite

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"

	"github.com/duskvirkus/fileditto/internal/db"
)

// DB is the SQLite implementation of db.DB.
type DB struct {
	conn    *sql.DB
	queue   *queueRepository
	files   *fileRepository
	devices *deviceRepository
	media   *mediaRepository
}

// Open opens or creates the SQLite database at path.
func Open(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	var journalMode string
	if err := conn.QueryRow("PRAGMA journal_mode=WAL").Scan(&journalMode); err != nil {
		conn.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}
	if journalMode != "wal" {
		conn.Close()
		return nil, fmt.Errorf("expected WAL journal mode, got %q", journalMode)
	}

	if _, err := conn.Exec("PRAGMA foreign_keys=ON"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	d := &DB{conn: conn}
	d.queue = &queueRepository{conn}
	d.files = &fileRepository{conn}
	d.devices = &deviceRepository{conn}
	d.media = &mediaRepository{conn}
	return d, nil
}

func (d *DB) Queue() db.QueueRepository    { return d.queue }
func (d *DB) Files() db.FileRepository     { return d.files }
func (d *DB) Devices() db.DeviceRepository { return d.devices }
func (d *DB) Media() db.MediaRepository    { return d.media }
func (d *DB) Close() error                 { return d.conn.Close() }

func (d *DB) Migrate() error {
	return db.RunMigrations(d.conn, db.SQLite)
}

// RawConn exposes the underlying *sql.DB for use in tests only.
func (d *DB) RawConn() *sql.DB { return d.conn }
