package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/duskvirkus/pxvault/internal/config"
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Manage the pxvault database configuration",
}

func init() {
	dbCmd.AddCommand(dbSetPathCmd)
	dbCmd.AddCommand(dbPathCmd)
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
