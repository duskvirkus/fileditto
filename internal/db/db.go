package db

import (
	"crypto/sha256"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// OpenDB opens or creates the SQLite database at path, enabling WAL mode.
func OpenDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode=WAL").Scan(&journalMode); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}
	if journalMode != "wal" {
		db.Close()
		return nil, fmt.Errorf("expected WAL journal mode, got %q", journalMode)
	}

	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	return db, nil
}

// RunMigrations applies all pending embedded migrations in order and verifies
// checksums of previously-applied migrations against the stored records.
// After all migrations, it verifies the project namespace UUID stored in
// schema_metadata against the compiled-in constant.
func RunMigrations(db *sql.DB) error {
	files, err := loadMigrationFiles()
	if err != nil {
		return err
	}

	// Determine current schema version (0 if schema_version table doesn't exist yet).
	currentVersion, err := currentSchemaVersion(db)
	if err != nil {
		return fmt.Errorf("read schema_version: %w", err)
	}

	for _, mf := range files {
		if mf.number <= currentVersion {
			// Already applied — verify checksum to detect tampering.
			if err := verifyChecksum(db, mf); err != nil {
				return err
			}
			continue
		}

		// Apply this migration in a transaction.
		if err := applyMigration(db, mf); err != nil {
			return fmt.Errorf("apply migration %04d: %w", mf.number, err)
		}
	}

	// Verify namespace UUID matches compiled-in constant.
	if err := verifyNamespaceUUID(db); err != nil {
		return err
	}

	return nil
}

// migrationFile holds a parsed migration.
type migrationFile struct {
	number   int
	filename string
	content  []byte
	checksum string // hex SHA-256 of content
}

func loadMigrationFiles() ([]migrationFile, error) {
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}

	var files []migrationFile
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}

		content, err := migrationsFS.ReadFile("migrations/" + e.Name())
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", e.Name(), err)
		}

		var number int
		if _, err := fmt.Sscanf(e.Name(), "%04d", &number); err != nil {
			return nil, fmt.Errorf("parse migration number from %s: %w", e.Name(), err)
		}

		sum := sha256.Sum256(content)
		files = append(files, migrationFile{
			number:   number,
			filename: e.Name(),
			content:  content,
			checksum: fmt.Sprintf("%x", sum),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].number < files[j].number
	})
	return files, nil
}

func currentSchemaVersion(db *sql.DB) (int, error) {
	// Check if schema_version table exists.
	var name string
	err := db.QueryRow(
		"SELECT name FROM sqlite_master WHERE type='table' AND name='schema_version'",
	).Scan(&name)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	var version int
	if err := db.QueryRow("SELECT version FROM schema_version").Scan(&version); err != nil {
		return 0, err
	}
	return version, nil
}

func verifyChecksum(db *sql.DB, mf migrationFile) error {
	var stored string
	err := db.QueryRow(
		"SELECT checksum FROM schema_migrations WHERE migration_number = ?",
		mf.number,
	).Scan(&stored)
	if err == sql.ErrNoRows {
		// Not recorded — skip verification (migration may have been applied outside runner).
		return nil
	}
	if err != nil {
		return fmt.Errorf("read checksum for migration %04d: %w", mf.number, err)
	}
	if stored != mf.checksum {
		return fmt.Errorf(
			"checksum mismatch for migration %04d: stored %s, computed %s",
			mf.number, stored, mf.checksum,
		)
	}
	return nil
}

func applyMigration(db *sql.DB, mf migrationFile) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.Exec(string(mf.content)); err != nil {
		return fmt.Errorf("execute SQL: %w", err)
	}

	// Update schema_version.
	if _, err := tx.Exec("UPDATE schema_version SET version = ?", mf.number); err != nil {
		return fmt.Errorf("update schema_version: %w", err)
	}

	// Record in schema_migrations.
	appliedAt := time.Now().UTC().Format(time.RFC3339)
	if _, err := tx.Exec(
		"INSERT INTO schema_migrations (migration_number, applied_at, checksum) VALUES (?, ?, ?)",
		mf.number, appliedAt, mf.checksum,
	); err != nil {
		return fmt.Errorf("record schema_migrations: %w", err)
	}

	return tx.Commit()
}

func verifyNamespaceUUID(db *sql.DB) error {
	var stored string
	err := db.QueryRow(
		"SELECT value FROM schema_metadata WHERE key = 'uuid_namespace'",
	).Scan(&stored)
	if err != nil {
		return fmt.Errorf("read uuid_namespace from schema_metadata: %w", err)
	}
	if stored != ProjectNamespace {
		return fmt.Errorf(
			"namespace UUID mismatch: stored %q, compiled-in %q",
			stored, ProjectNamespace,
		)
	}
	return nil
}
