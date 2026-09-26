package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/duskvirkus/fileditto/internal/config"
	"github.com/duskvirkus/fileditto/internal/db"
)

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
