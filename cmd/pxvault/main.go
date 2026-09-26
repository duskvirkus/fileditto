package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/duskvirkus/pxvault/internal/config"
	"github.com/duskvirkus/pxvault/internal/db"
	"github.com/duskvirkus/pxvault/internal/ingestion"
	"github.com/duskvirkus/pxvault/internal/scanner"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "pxvault",
	Short: "Photo and video archive manager",
}

func init() {
	rootCmd.AddCommand(ingestCmd)
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(dbCmd)
	dbCmd.AddCommand(dbSetPathCmd)
	dbCmd.AddCommand(dbPathCmd)
	ingestCmd.Flags().Int("max-failures", 3, "abort after this many processing errors")
}

// --- ingest ---

var ingestCmd = &cobra.Command{
	Use:   "ingest <directory>",
	Short: "Scan a directory and ingest all media files",
	Args:  cobra.ExactArgs(1),
	RunE:  runIngest,
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

	fmt.Printf("scanning %s\n", root)
	if err := scanner.Discover(conn, root); err != nil {
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
		ok, err := ingestion.ProcessNext(conn)
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

// --- migrate ---

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run any pending database migrations",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
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

		fmt.Printf("database migrated: %s\n", cfg.DBPath)
		return nil
	},
}

// --- db ---

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Manage the pxvault database configuration",
}

var dbSetPathCmd = &cobra.Command{
	Use:   "set-path <path>",
	Short: "Set the database file path",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := &config.Config{DBPath: args[0]}
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Printf("db path set to %s\n", args[0])
		return nil
	},
}

var dbPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show the configured database file path",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		fmt.Println(cfg.DBPath)
		return nil
	},
}
