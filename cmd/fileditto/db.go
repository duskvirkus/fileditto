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
	dbCmd.AddCommand(dbPathCmd)
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

var dbPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show the configured database path or DSN",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		switch cfg.DBType {
		case "sqlite":
			fmt.Println(cfg.DBPath)
		case "postgres":
			fmt.Println(cfg.DBDSN)
		default:
			fmt.Printf("type=%s (unconfigured)\n", cfg.DBType)
		}
		return nil
	},
}
