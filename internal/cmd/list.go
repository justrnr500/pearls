package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/justrnr500/pearls/internal/storage"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List pearls",
	Long: `List all pearls with optional filtering.

Examples:
  pearls list
  pearls list --type table
  pearls list --namespace db.postgres
  pearls list --tag pii
  pearls list --status active
  pearls list --scope backend
  pearls list --json`,
	Aliases: []string{"ls"},
	RunE:    runList,
}

var (
	listNamespace string
	listType      string
	listStatus    string
	listTag       string
	listScope     string
	listJSON      bool
	listLimit     int
)

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringVarP(&listNamespace, "namespace", "n", "", "Filter by namespace")
	listCmd.Flags().StringVarP(&listType, "type", "t", "", "Filter by type")
	listCmd.Flags().StringVarP(&listStatus, "status", "s", "", "Filter by status")
	listCmd.Flags().StringVar(&listTag, "tag", "", "Filter by tag")
	listCmd.Flags().StringVar(&listScope, "scope", "", "Filter by scope")
	listCmd.Flags().BoolVar(&listJSON, "json", false, "Output as JSON")
	listCmd.Flags().IntVar(&listLimit, "limit", 0, "Limit number of results")
}

func runList(cmd *cobra.Command, args []string) error {
	store, _, err := getStore()
	if err != nil {
		return err
	}
	defer store.Close()

	opts := storage.ListOptions{
		Namespace: listNamespace,
		Type:      listType,
		Status:    listStatus,
		Tag:       listTag,
		Scope:     listScope,
		Limit:     listLimit,
	}

	pearls, err := store.List(opts)
	if err != nil {
		return fmt.Errorf("list pearls: %w", err)
	}

	if listJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]interface{}{
			"pearls": pearls,
			"count":  len(pearls),
		})
	}

	if len(pearls) == 0 {
		fmt.Println("No pearls found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTYPE\tSTATUS\tDESCRIPTION")
	fmt.Fprintln(w, "──\t────\t──────\t───────────")

	for _, p := range pearls {
		desc := p.Description
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.ID, p.Type, p.Status, desc)
	}
	w.Flush()

	fmt.Printf("\n%d pearl(s)\n", len(pearls))
	return nil
}

// truncate truncates a string to max length with ellipsis
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// formatTags formats tags for display
func formatTags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	return strings.Join(tags, ", ")
}
