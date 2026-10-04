// Package app provides application-level wiring such as the database factory.
package app

import (
	"fmt"

	"github.com/duskvirkus/fileditto/internal/config"
	"github.com/duskvirkus/fileditto/internal/db"
	"github.com/duskvirkus/fileditto/internal/db/postgres"
	"github.com/duskvirkus/fileditto/internal/db/sqlite"
)

// OpenDB opens the database specified by cfg and returns a db.DB ready for use.
// It does not run migrations — call d.Migrate() separately.
func OpenDB(cfg *config.Config) (db.DB, error) {
	switch db.Dialect(cfg.DBType) {
	case db.SQLite:
		return sqlite.Open(cfg.DBPath)
	case db.PostgreSQL:
		return postgres.Open(cfg.DBDSN)
	default:
		return nil, fmt.Errorf("unknown db type %q — run 'fileditto db configure'", cfg.DBType)
	}
}
