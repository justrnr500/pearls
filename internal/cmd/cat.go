package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var catCmd = &cobra.Command{
	Use:   "cat <id>",
	Short: "Display pearl content",
	Long: `Display the markdown content of a pearl.

Examples:
  pearls cat db.postgres.users
  pearls cat api.stripe.customers`,
	Args: cobra.ExactArgs(1),
	RunE: runCat,
}

func init() {
	rootCmd.AddCommand(catCmd)
}

func runCat(cmd *cobra.Command, args []string) error {
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

	if p.ContentPath == "" {
		return fmt.Errorf("pearl has no content file")
	}

	content, err := store.GetContent(p)
	if err != nil {
		return fmt.Errorf("read content: %w", err)
	}

	fmt.Fprint(os.Stdout, content)
	return nil
}
