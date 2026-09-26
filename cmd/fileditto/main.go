package main

import (
	"os"

	"github.com/spf13/cobra"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "fileditto",
	Short: "File archive and backup manager",
}

func init() {
	rootCmd.AddCommand(ingestCmd)
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(dbCmd)
}
