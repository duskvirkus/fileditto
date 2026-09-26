package main

import (
	"fmt"
	"os"

	"github.com/duskvirkus/pxvault/internal/db"
	"github.com/duskvirkus/pxvault/internal/ingestion"
	"github.com/duskvirkus/pxvault/internal/scanner"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "usage: ingest <db-path> <directory>\n")
		os.Exit(1)
	}
	dbPath := os.Args[1]
	root := os.Args[2]

	conn, err := db.OpenDB(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open db: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	if err := db.RunMigrations(conn); err != nil {
		fmt.Fprintf(os.Stderr, "run migrations: %v\n", err)
		os.Exit(1)
	}

	recovered, err := ingestion.ResetStuck(conn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reset stuck entries: %v\n", err)
		os.Exit(1)
	}
	if recovered > 0 {
		fmt.Printf("recovered %d stuck queue entries\n", recovered)
	}

	fmt.Printf("scanning %s\n", root)
	if err := scanner.Discover(conn, root); err != nil {
		fmt.Fprintf(os.Stderr, "scan: %v\n", err)
		os.Exit(1)
	}

	pending, err := ingestion.PendingCount(conn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pending count: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("processing %d files\n", pending)

	const maxFailures = 3
	var processed, failed int
	for {
		ok, err := ingestion.ProcessNext(conn)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			failed++
			if failed >= maxFailures {
				fmt.Fprintf(os.Stderr, "aborting after %d failures\n", failed)
				os.Exit(1)
			}
		}
		if !ok {
			break
		}
		processed++
		if processed%100 == 0 {
			fmt.Printf("  %d done\n", processed)
		}
	}

	fmt.Printf("done: %d processed, %d errors\n", processed, failed)
}
