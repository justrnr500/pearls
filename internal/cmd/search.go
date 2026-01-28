package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/justrnr500/pearls/internal/pearl"
	"github.com/justrnr500/pearls/internal/storage"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search pearls",
	Long: `Search for pearls by keyword or semantic similarity.

By default, searches across ID, name, namespace, description, and tags.
Use --semantic for natural language queries using vector similarity.

Examples:
  pearls search customer
  pearls search "user email"
  pearls search orders --type table
  pearls search "where is payment data stored" --semantic
  pearls search analytics --json`,
	Args: cobra.ExactArgs(1),
	RunE: runSearch,
}

var (
	searchType     string
	searchStatus   string
	searchTag      string
	searchJSON     bool
	searchLimit    int
	searchSemantic bool
)

func init() {
	rootCmd.AddCommand(searchCmd)
	searchCmd.Flags().StringVarP(&searchType, "type", "t", "", "Filter by type")
	searchCmd.Flags().StringVarP(&searchStatus, "status", "s", "", "Filter by status")
	searchCmd.Flags().StringVar(&searchTag, "tag", "", "Filter by tag")
	searchCmd.Flags().BoolVar(&searchJSON, "json", false, "Output as JSON")
	searchCmd.Flags().IntVar(&searchLimit, "limit", 50, "Maximum results")
	searchCmd.Flags().BoolVar(&searchSemantic, "semantic", false, "Use semantic search (natural language)")
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := args[0]

	store, _, err := getStore()
	if err != nil {
		return err
	}
	defer store.Close()

	if searchSemantic {
		return runSemanticSearch(store, query)
	}
	return runKeywordSearch(store, query)
}

func runKeywordSearch(store interface {
	Search(string, int) ([]*pearl.Pearl, error)
}, query string) error {
	// Perform keyword search
	results, err := store.Search(query, searchLimit)
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}

	// Apply additional filters
	filtered := filterPearls(results)

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

// searchResult holds a pearl with optional similarity score
type searchResult struct {
	Pearl      *pearl.Pearl `json:"pearl"`
	Similarity float32      `json:"similarity,omitempty"`
}

func runSemanticSearch(store interface {
	SearchSemantic(string, int) ([]storage.SemanticResult, error)
	Get(string) (*pearl.Pearl, error)
	HasEmbedder() bool
}, query string) error {
	if !store.HasEmbedder() {
		return fmt.Errorf("semantic search requires vector search to be enabled\nRun 'pearls index --rebuild' to generate embeddings")
	}

	// Perform semantic search
	semanticResults, err := store.SearchSemantic(query, searchLimit)
	if err != nil {
		return fmt.Errorf("semantic search: %w", err)
	}

	// Fetch full pearl data and build results with similarity scores
	var results []searchResult
	for _, sr := range semanticResults {
		p, err := store.Get(sr.ID)
		if err != nil || p == nil {
			continue
		}

		// Convert distance to similarity (1 / (1 + distance))
		similarity := 1.0 / (1.0 + sr.Distance)

		// Apply filters
		if !matchesFilters(p) {
			continue
		}

		results = append(results, searchResult{
			Pearl:      p,
			Similarity: float32(similarity),
		})
	}

	if searchJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]interface{}{
			"query":    query,
			"semantic": true,
			"results":  results,
			"count":    len(results),
		})
	}

	if len(results) == 0 {
		fmt.Printf("No semantic results for %q\n", query)
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SCORE\tID\tTYPE\tDESCRIPTION")
	fmt.Fprintln(w, "─────\t──\t────\t───────────")

	for _, r := range results {
		desc := r.Pearl.Description
		if len(desc) > 45 {
			desc = desc[:42] + "..."
		}
		fmt.Fprintf(w, "%.2f\t%s\t%s\t%s\n", r.Similarity, r.Pearl.ID, r.Pearl.Type, desc)
	}
	w.Flush()

	fmt.Printf("\n%d semantic result(s) for %q\n", len(results), query)
	return nil
}

func filterPearls(pearls []*pearl.Pearl) []*pearl.Pearl {
	if searchType == "" && searchStatus == "" && searchTag == "" {
		return pearls
	}

	filtered := make([]*pearl.Pearl, 0)
	for _, p := range pearls {
		if matchesFilters(p) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

func matchesFilters(p *pearl.Pearl) bool {
	if searchType != "" && string(p.Type) != searchType {
		return false
	}
	if searchStatus != "" && string(p.Status) != searchStatus {
		return false
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
			return false
		}
	}
	return true
}
