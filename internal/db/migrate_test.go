package db_test

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/duskvirkus/fileditto/internal/db"
)

func TestResolveMigration_SharedUsedForAllDialects(t *testing.T) {
	fsys := fstest.MapFS{
		"shared/0001_init.sql": {Data: []byte("CREATE TABLE t (id TEXT);")},
		"sqlite/placeholder":   {Data: []byte("")},
		"postgres/placeholder": {Data: []byte("")},
	}
	content, err := db.ResolveMigration(fsys, db.SQLite, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(content) != "CREATE TABLE t (id TEXT);" {
		t.Errorf("unexpected content: %q", content)
	}
}

func TestResolveMigration_DialectFileUsedWhenBothPresent(t *testing.T) {
	fsys := fstest.MapFS{
		"shared/placeholder":      {Data: []byte("")},
		"sqlite/0001_init.sql":    {Data: []byte("sqlite-specific")},
		"postgres/0001_init.sql":  {Data: []byte("postgres-specific")},
	}
	content, err := db.ResolveMigration(fsys, db.SQLite, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(content) != "sqlite-specific" {
		t.Errorf("expected sqlite-specific, got %q", content)
	}

	content, err = db.ResolveMigration(fsys, db.PostgreSQL, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(content) != "postgres-specific" {
		t.Errorf("expected postgres-specific, got %q", content)
	}
}

func TestResolveMigration_ErrorWhenOnlyOneDialectPresent(t *testing.T) {
	fsys := fstest.MapFS{
		"shared/placeholder":   {Data: []byte("")},
		"sqlite/0001_init.sql": {Data: []byte("sqlite-only")},
		"postgres/placeholder": {Data: []byte("")},
	}
	_, err := db.ResolveMigration(fsys, db.SQLite, 1)
	if err == nil {
		t.Error("expected error when only one dialect has the migration, got nil")
	}
}

func TestResolveMigration_ErrorWhenMigrationMissingEverywhere(t *testing.T) {
	fsys := fstest.MapFS{
		"shared/placeholder":   {Data: []byte("")},
		"sqlite/placeholder":   {Data: []byte("")},
		"postgres/placeholder": {Data: []byte("")},
	}
	_, err := db.ResolveMigration(fsys, db.SQLite, 1)
	if err == nil {
		t.Error("expected error when migration is missing everywhere, got nil")
	}
}

func TestListMigrationNumbers_ReturnsUnionSorted(t *testing.T) {
	fsys := fstest.MapFS{
		"shared/0001_init.sql":   {Data: []byte("")},
		"shared/0003_add.sql":    {Data: []byte("")},
		"sqlite/0002_fix.sql":    {Data: []byte("")},
		"postgres/0002_fix.sql":  {Data: []byte("")},
	}
	nums, err := db.ListMigrationNumbers(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []int{1, 2, 3}
	if len(nums) != len(want) {
		t.Fatalf("expected %v, got %v", want, nums)
	}
	for i, n := range nums {
		if n != want[i] {
			t.Errorf("index %d: expected %d, got %d", i, want[i], n)
		}
	}
}

// Ensure fstest.MapFS satisfies fs.FS.
var _ fs.FS = fstest.MapFS{}
