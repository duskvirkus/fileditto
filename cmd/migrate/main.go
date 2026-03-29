// migrate opens or creates a pxvault database at the given path and runs all
// pending migrations. Used by the Python integration tests to produce a
// fully-migrated database file.
package main

import (
	"fmt"
	"os"

	"github.com/duskvirkus/pxvault/internal/db"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: migrate <db-path>\n")
		os.Exit(1)
	}
	path := os.Args[1]

	conn, err := db.OpenDB(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open db: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	if err := db.RunMigrations(conn); err != nil {
		fmt.Fprintf(os.Stderr, "run migrations: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("database migrated: %s\n", path)
}
