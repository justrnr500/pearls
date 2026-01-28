package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/justrnr500/pearls/internal/pearl"
	"github.com/justrnr500/pearls/internal/storage"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a pearl",
	Long: `Delete a pearl from the catalog.

By default, this archives the pearl (sets status to archived).
Use --force to permanently delete the pearl and its content file.

Examples:
  pearls delete db.postgres.old_table
  pearls delete db.postgres.old_table --force
  pearls delete db.legacy --recursive --force`,
	Args: cobra.ExactArgs(1),
	RunE: runDelete,
}

var archiveCmd = &cobra.Command{
	Use:   "archive <id>",
	Short: "Archive a pearl",
	Long: `Archive a pearl (set status to archived).

This is a soft delete - the pearl remains in the catalog but is marked as archived.

Examples:
  pearls archive db.postgres.old_table`,
	Args: cobra.ExactArgs(1),
	RunE: runArchive,
}

var (
	deleteForce     bool
	deleteRecursive bool
	deleteYes       bool
)

func init() {
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(archiveCmd)

	deleteCmd.Flags().BoolVarP(&deleteForce, "force", "f", false, "Permanently delete (not just archive)")
	deleteCmd.Flags().BoolVarP(&deleteRecursive, "recursive", "r", false, "Delete all pearls in namespace")
	deleteCmd.Flags().BoolVarP(&deleteYes, "yes", "y", false, "Skip confirmation prompt")
}

func runArchive(cmd *cobra.Command, args []string) error {
	id := args[0]

	store, _, err := getStore()
	if err != nil {
		return err
	}
	defer store.Close()

	p, err := store.Get(id)
	if err != nil {
		return fmt.Errorf("get pearl: %w", err)
	}
	if p == nil {
		return fmt.Errorf("pearl not found: %s", id)
	}

	p.Status = pearl.StatusArchived
	p.UpdatedAt = time.Now()

	if err := store.Update(p, nil); err != nil {
		return fmt.Errorf("archive pearl: %w", err)
	}

	fmt.Printf("✓ Archived pearl: %s\n", p.ID)
	return nil
}

func runDelete(cmd *cobra.Command, args []string) error {
	id := args[0]

	store, _, err := getStore()
	if err != nil {
		return err
	}
	defer store.Close()

	// If not force, just archive
	if !deleteForce {
		p, err := store.Get(id)
		if err != nil {
			return fmt.Errorf("get pearl: %w", err)
		}
		if p == nil {
			return fmt.Errorf("pearl not found: %s", id)
		}

		p.Status = pearl.StatusArchived
		p.UpdatedAt = time.Now()

		if err := store.Update(p, nil); err != nil {
			return fmt.Errorf("archive pearl: %w", err)
		}

		fmt.Printf("✓ Archived pearl: %s (use --force to permanently delete)\n", p.ID)
		return nil
	}

	// Force delete
	if deleteRecursive {
		// Find all pearls in namespace
		pearls, err := store.List(storage.ListOptions{Namespace: id})
		if err != nil {
			return fmt.Errorf("list pearls: %w", err)
		}

		// Also get the pearl itself if it exists
		p, _ := store.Get(id)
		if p != nil {
			pearls = append(pearls, p)
		}

		if len(pearls) == 0 {
			return fmt.Errorf("no pearls found in namespace: %s", id)
		}

		// Confirm
		if !deleteYes {
			fmt.Printf("This will permanently delete %d pearl(s):\n", len(pearls))
			for _, p := range pearls {
				fmt.Printf("  - %s\n", p.ID)
			}
			fmt.Print("\nContinue? [y/N] ")

			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				fmt.Println("Aborted.")
				return nil
			}
		}

		// Delete all
		for _, p := range pearls {
			if err := store.Delete(p.ID); err != nil {
				return fmt.Errorf("delete pearl %s: %w", p.ID, err)
			}
			fmt.Printf("✓ Deleted: %s\n", p.ID)
		}

		return nil
	}

	// Single delete
	p, err := store.Get(id)
	if err != nil {
		return fmt.Errorf("get pearl: %w", err)
	}
	if p == nil {
		return fmt.Errorf("pearl not found: %s", id)
	}

	// Confirm
	if !deleteYes {
		fmt.Printf("Permanently delete %s? [y/N] ", id)

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	if err := store.Delete(id); err != nil {
		return fmt.Errorf("delete pearl: %w", err)
	}

	fmt.Printf("✓ Deleted pearl: %s\n", id)
	return nil
}
