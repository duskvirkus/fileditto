// Migration logic is in this file; see db.go for types and interfaces.
package db

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"
)

// RunMigrations applies pending migrations to conn using the embedded MigrationsFS.
func RunMigrations(conn *sql.DB, dialect Dialect) error {
	// Strip the "migrations/" prefix so ResolveMigration sees "shared/", "sqlite/", "postgres/".
	sub, err := fs.Sub(MigrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("migrations sub-fs: %w", err)
	}

	numbers, err := ListMigrationNumbers(sub)
	if err != nil {
		return err
	}

	current, err := currentSchemaVersion(conn, dialect)
	if err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}

	for _, n := range numbers {
		content, err := ResolveMigration(sub, dialect, n)
		if err != nil {
			return fmt.Errorf("resolve migration %04d: %w", n, err)
		}

		sum := checksumOf(content)

		if n <= current {
			if err := verifyChecksum(conn, dialect, n, sum); err != nil {
				return err
			}
			continue
		}

		if err := applyMigration(conn, dialect, n, content, sum); err != nil {
			return fmt.Errorf("apply migration %04d: %w", n, err)
		}
	}

	return verifyNamespaceUUID(conn, dialect)
}

// ResolveMigration returns the SQL content for migration number n using dialect d.
// Shared takes priority. If not in shared, both dialect dirs must have it.
func ResolveMigration(fsys fs.FS, dialect Dialect, n int) ([]byte, error) {
	if content, ok := readFirst(fsys, "shared", n); ok {
		return content, nil
	}

	sqliteContent, hasSQLite := readFirst(fsys, "sqlite", n)
	postgresContent, hasPostgres := readFirst(fsys, "postgres", n)

	if hasSQLite && hasPostgres {
		switch dialect {
		case SQLite:
			return sqliteContent, nil
		case PostgreSQL:
			return postgresContent, nil
		}
	}

	if hasSQLite != hasPostgres {
		return nil, fmt.Errorf(
			"migration %04d is dialect-specific but only one dialect directory has it (both are required)",
			n,
		)
	}

	return nil, fmt.Errorf("migration %04d not found in shared, sqlite, or postgres directories", n)
}

// ListMigrationNumbers returns the sorted union of migration numbers across all directories.
func ListMigrationNumbers(fsys fs.FS) ([]int, error) {
	seen := map[int]bool{}
	for _, dir := range []string{"shared", "sqlite", "postgres"} {
		entries, err := fs.ReadDir(fsys, dir)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", dir, err)
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
				continue
			}
			var n int
			if _, err := fmt.Sscanf(e.Name(), "%04d", &n); err != nil {
				return nil, fmt.Errorf("parse migration number from %s: %w", e.Name(), err)
			}
			seen[n] = true
		}
	}

	nums := make([]int, 0, len(seen))
	for n := range seen {
		nums = append(nums, n)
	}
	sort.Ints(nums)
	return nums, nil
}

// readFirst looks for any file named NNNN_*.sql in dir and returns its content.
func readFirst(fsys fs.FS, dir string, n int) ([]byte, bool) {
	prefix := fmt.Sprintf("%04d", n)
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, false
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), prefix) && strings.HasSuffix(e.Name(), ".sql") {
			content, err := fs.ReadFile(fsys, dir+"/"+e.Name())
			if err != nil {
				return nil, false
			}
			return content, true
		}
	}
	return nil, false
}

func checksumOf(content []byte) string {
	sum := sha256.Sum256(content)
	return fmt.Sprintf("%x", sum)
}

func currentSchemaVersion(conn *sql.DB, dialect Dialect) (int, error) {
	exists, err := tableExists(conn, dialect, "schema_version")
	if err != nil || !exists {
		return 0, err
	}
	var version int
	if err := conn.QueryRow("SELECT version FROM schema_version").Scan(&version); err != nil {
		return 0, err
	}
	return version, nil
}

func tableExists(conn *sql.DB, dialect Dialect, name string) (bool, error) {
	var query string
	var arg interface{}
	switch dialect {
	case SQLite:
		query = "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?"
		arg = name
	case PostgreSQL:
		query = "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='public' AND table_name=$1"
		arg = name
	default:
		return false, fmt.Errorf("unknown dialect %q", dialect)
	}
	var count int
	if err := conn.QueryRow(query, arg).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

// ph returns the placeholder for argument position n (1-based) for the dialect.
func ph(dialect Dialect, n int) string {
	if dialect == PostgreSQL {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}

func verifyChecksum(conn *sql.DB, dialect Dialect, n int, computed string) error {
	var stored string
	q := fmt.Sprintf(
		"SELECT checksum FROM schema_migrations WHERE migration_number = %s",
		ph(dialect, 1),
	)
	err := conn.QueryRow(q, n).Scan(&stored)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read checksum for migration %04d: %w", n, err)
	}
	if stored != computed {
		return fmt.Errorf(
			"checksum mismatch for migration %04d: stored %s, computed %s",
			n, stored, computed,
		)
	}
	return nil
}

func applyMigration(conn *sql.DB, dialect Dialect, n int, content []byte, sum string) error {
	tx, err := conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.Exec(string(content)); err != nil {
		return fmt.Errorf("execute SQL: %w", err)
	}

	updateVersion := fmt.Sprintf("UPDATE schema_version SET version = %s", ph(dialect, 1))
	if _, err := tx.Exec(updateVersion, n); err != nil {
		return fmt.Errorf("update schema_version: %w", err)
	}

	insertMig := fmt.Sprintf(
		"INSERT INTO schema_migrations (migration_number, applied_at, checksum) VALUES (%s, %s, %s)",
		ph(dialect, 1), ph(dialect, 2), ph(dialect, 3),
	)
	if _, err := tx.Exec(insertMig, n, time.Now().UTC().Format(time.RFC3339), sum); err != nil {
		return fmt.Errorf("record migration: %w", err)
	}

	return tx.Commit()
}

func verifyNamespaceUUID(conn *sql.DB, dialect Dialect) error {
	q := fmt.Sprintf(
		"SELECT value FROM schema_metadata WHERE key = %s",
		ph(dialect, 1),
	)
	var stored string
	if err := conn.QueryRow(q, "uuid_namespace").Scan(&stored); err != nil {
		return fmt.Errorf("read uuid_namespace: %w", err)
	}
	if stored != ProjectNamespace {
		return fmt.Errorf("namespace UUID mismatch: stored %q, compiled-in %q", stored, ProjectNamespace)
	}
	return nil
}
