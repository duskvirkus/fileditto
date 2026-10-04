// Package postgres implements the db.DB interface using a PostgreSQL database.
package postgres

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/duskvirkus/fileditto/internal/db"
)

type DB struct {
	conn    *sql.DB
	queue   *queueRepository
	files   *fileRepository
	devices *deviceRepository
	media   *mediaRepository
}

// Open opens a connection to the PostgreSQL database at dsn and verifies it
// with a ping. The caller should call Migrate before first use.
func Open(dsn string) (*DB, error) {
	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
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
	return db.RunMigrations(d.conn, db.PostgreSQL)
}

func (d *DB) RawConn() *sql.DB { return d.conn }
