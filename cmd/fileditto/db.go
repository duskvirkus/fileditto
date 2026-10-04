package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/duskvirkus/fileditto/internal/config"
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Manage the fileditto database configuration",
}

func init() {
	dbCmd.AddCommand(dbSetPathCmd)
	dbCmd.AddCommand(dbShowCmd)
	dbCmd.AddCommand(dbConfigureCmd)
}

var dbSetPathCmd = &cobra.Command{
	Use:   "set-path <path>",
	Short: "Set the SQLite database file path",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := &config.Config{DBType: "sqlite", DBPath: args[0]}
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Printf("db path set to %s\n", args[0])
		return nil
	},
}

var dbShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show the current database configuration",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		fmt.Printf("type: %s\n", cfg.DBType)
		switch cfg.DBType {
		case "sqlite":
			fmt.Printf("path: %s\n", cfg.DBPath)
		case "postgres":
			fmt.Printf("dsn:  %s\n", cfg.DBDSN)
		}
		return nil
	},
}
