// Package cmd provides the CLI commands for pearls.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version information set via ldflags
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "pearls",
	Short: "Data asset memory for AI agents",
	Long: `Pearls is a CLI tool for storing and retrieving structured markdown
documentation about data assets—tables, schemas, database connections,
file locations, APIs, and other data sources.

Beads gives agents memory about tasks. Pearls gives agents memory about data.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Version = Version
	rootCmd.SetVersionTemplate(`{{printf "pearls %s\ncommit: %s\nbuilt: %s\n" .Version "` + Commit + `" "` + BuildDate + `"}}`)
}
