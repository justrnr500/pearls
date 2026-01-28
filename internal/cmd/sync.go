package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync database with JSONL",
	Long: `Synchronize the SQLite database with the JSONL file.

By default, syncs from JSONL to SQLite (JSONL is source of truth).
Use --to-jsonl to export the database to JSONL.
Use --refresh-hashes to update content hashes.

Examples:
  pearls sync              # Rebuild SQLite from JSONL
  pearls sync --to-jsonl   # Export SQLite to JSONL
  pearls sync --refresh-hashes  # Update content hashes`,
	RunE: runSync,
}

var (
	syncToJSONL       bool
	syncRefreshHashes bool
)

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.Flags().BoolVar(&syncToJSONL, "to-jsonl", false, "Export database to JSONL")
	syncCmd.Flags().BoolVar(&syncRefreshHashes, "refresh-hashes", false, "Update content hashes from files")
}

func runSync(cmd *cobra.Command, args []string) error {
	store, _, err := getStore()
	if err != nil {
		return err
	}
	defer store.Close()

	if syncRefreshHashes {
		fmt.Println("Refreshing content hashes...")
		if err := store.RefreshContentHashes(); err != nil {
			return fmt.Errorf("refresh hashes: %w", err)
		}
		fmt.Println("✓ Content hashes updated")
		return nil
	}

	if syncToJSONL {
		fmt.Println("Exporting database to JSONL...")
		if err := store.SyncToJSONL(); err != nil {
			return fmt.Errorf("sync to jsonl: %w", err)
		}
		fmt.Println("✓ Database exported to JSONL")
		return nil
	}

	// Default: rebuild SQLite from JSONL
	fmt.Println("Rebuilding database from JSONL...")
	if err := store.SyncFromJSONL(); err != nil {
		return fmt.Errorf("sync from jsonl: %w", err)
	}

	count, err := store.DB().Count()
	if err != nil {
		return fmt.Errorf("count pearls: %w", err)
	}

	fmt.Printf("✓ Database rebuilt (%d pearls)\n", count)
	return nil
}
