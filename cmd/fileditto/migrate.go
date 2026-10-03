package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/duskvirkus/fileditto/internal/app"
	"github.com/duskvirkus/fileditto/internal/config"
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
		d, err := app.OpenDB(cfg)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}
		defer d.Close()
		if err := d.Migrate(); err != nil {
			return fmt.Errorf("run migrations: %w", err)
		}
		fmt.Println("migrations complete")
		return nil
	},
}
