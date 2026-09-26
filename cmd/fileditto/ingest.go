package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/duskvirkus/fileditto/internal/config"
	"github.com/duskvirkus/fileditto/internal/db"
	"github.com/duskvirkus/fileditto/internal/ingestion"
	"github.com/duskvirkus/fileditto/internal/scanner"
)

var ingestCmd = &cobra.Command{
	Use:   "ingest <directory>",
	Short: "Scan a directory and ingest all media files",
	Args:  cobra.ExactArgs(1),
	RunE:  runIngest,
}

func init() {
	ingestCmd.Flags().Int("max-failures", 3, "abort after this many processing errors")
}

func runIngest(cmd *cobra.Command, args []string) error {
	root := args[0]

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	conn, err := db.OpenDB(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer conn.Close()

	if err := db.RunMigrations(conn); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	recovered, err := ingestion.ResetStuck(conn)
	if err != nil {
		return fmt.Errorf("reset stuck entries: %w", err)
	}
	if recovered > 0 {
		fmt.Printf("recovered %d stuck queue entries\n", recovered)
	}

	deviceID, err := ingestion.EnsureLocalDevice(conn)
	if err != nil {
		return fmt.Errorf("ensure local device: %w", err)
	}

	mediaID, err := ingestion.EnsureDriveForPath(conn, root, deviceID)
	if err != nil {
		return fmt.Errorf("ensure local drive: %w", err)
	}

	fmt.Printf("scanning %s\n", root)
	if err := scanner.Discover(conn, root, deviceID); err != nil {
		return fmt.Errorf("scan: %w", err)
	}

	pending, err := ingestion.PendingCount(conn)
	if err != nil {
		return fmt.Errorf("pending count: %w", err)
	}
	fmt.Printf("processing %d files\n", pending)

	maxFailures, _ := cmd.Flags().GetInt("max-failures")
	var processed, failed int
	for {
		ok, err := ingestion.ProcessNext(conn, mediaID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			failed++
			if failed >= maxFailures {
				return fmt.Errorf("aborting after %d failures", failed)
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
	return nil
}
