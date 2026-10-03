package main

import (
	"fmt"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/duskvirkus/fileditto/internal/config"
)

var dbConfigureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Interactively configure the database connection",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		typePrompt := promptui.Select{
			Label: "Select database type",
			Items: []string{"SQLite", "PostgreSQL"},
		}
		_, dbType, err := typePrompt.Run()
		if err != nil {
			return fmt.Errorf("prompt: %w", err)
		}

		cfg := &config.Config{}

		switch dbType {
		case "SQLite":
			cfg.DBType = "sqlite"
			pathPrompt := promptui.Prompt{
				Label:   "Database file path",
				Default: "~/.local/share/fileditto/fileditto.db",
			}
			path, err := pathPrompt.Run()
			if err != nil {
				return fmt.Errorf("prompt: %w", err)
			}
			cfg.DBPath = path

		case "PostgreSQL":
			cfg.DBType = "postgres"
			dsnPrompt := promptui.Prompt{
				Label:   "Connection string (DSN)",
				Default: "postgres://user:pass@localhost:5432/fileditto",
				Mask:    0,
			}
			dsn, err := dsnPrompt.Run()
			if err != nil {
				return fmt.Errorf("prompt: %w", err)
			}
			cfg.DBDSN = dsn
		}

		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Printf("database configured: type=%s\n", cfg.DBType)
		return nil
	},
}
