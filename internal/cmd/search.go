package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/justrnr500/pearls/internal/pearl"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search pearls",
	Long: `Search for pearls by keyword.

Searches across ID, name, namespace, description, and tags.

Examples:
  pearls search customer
  pearls search "user email"
  pearls search orders --type table
  pearls search analytics --json`,
	Args: cobra.ExactArgs(1),
	RunE: runSearch,
}

var (
	searchType   string
	searchStatus string
	searchTag    string
	searchJSON   bool
	searchLimit  int
)

func init() {
	rootCmd.AddCommand(searchCmd)
	searchCmd.Flags().StringVarP(&searchType, "type", "t", "", "Filter by type")
	searchCmd.Flags().StringVarP(&searchStatus, "status", "s", "", "Filter by status")
	searchCmd.Flags().StringVar(&searchTag, "tag", "", "Filter by tag")
	searchCmd.Flags().BoolVar(&searchJSON, "json", false, "Output as JSON")
	searchCmd.Flags().IntVar(&searchLimit, "limit", 50, "Maximum results")
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := args[0]

	store, _, err := getStore()
	if err != nil {
		return err
	}
	defer store.Close()

	// Perform search
	results, err := store.Search(query, searchLimit)
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}

	// Apply additional filters
	filtered := results
	if searchType != "" || searchStatus != "" || searchTag != "" {
		filtered = make([]*pearl.Pearl, 0)
		for _, p := range results {
			if searchType != "" && string(p.Type) != searchType {
				continue
			}
			if searchStatus != "" && string(p.Status) != searchStatus {
				continue
			}
			if searchTag != "" {
				hasTag := false
				for _, t := range p.Tags {
					if t == searchTag {
						hasTag = true
						break
					}
				}
				if !hasTag {
					continue
				}
			}
			filtered = append(filtered, p)
		}
	}

	if searchJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]interface{}{
			"query":   query,
			"results": filtered,
			"count":   len(filtered),
		})
	}

	if len(filtered) == 0 {
		fmt.Printf("No results for %q\n", query)
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTYPE\tSTATUS\tDESCRIPTION")
	fmt.Fprintln(w, "──\t────\t──────\t───────────")

	for _, p := range filtered {
		desc := p.Description
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.ID, p.Type, p.Status, desc)
	}
	w.Flush()

	fmt.Printf("\n%d result(s) for %q\n", len(filtered), query)
	return nil
}
