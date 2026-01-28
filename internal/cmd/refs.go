package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var refsCmd = &cobra.Command{
	Use:   "refs <pearl-id>",
	Short: "Show pearl references",
	Long: `Show what a pearl references and what references it.

Displays bidirectional relationships:
  - Outgoing: pearls that this pearl references
  - Incoming: pearls that reference this pearl

Examples:
  pearls refs db.postgres.orders
  pearls refs db.postgres.users --json`,
	Args: cobra.ExactArgs(1),
	RunE: runRefs,
}

var refsJSON bool

func init() {
	rootCmd.AddCommand(refsCmd)
	refsCmd.Flags().BoolVar(&refsJSON, "json", false, "Output as JSON")
}

func runRefs(cmd *cobra.Command, args []string) error {
	id := args[0]

	store, _, err := getStore()
	if err != nil {
		return err
	}
	defer store.Close()

	// Get the pearl
	p, err := store.Get(id)
	if err != nil {
		return fmt.Errorf("get pearl: %w", err)
	}
	if p == nil {
		return fmt.Errorf("pearl not found: %s", id)
	}

	// Outgoing references (what this pearl references)
	outgoing := p.References
	if outgoing == nil {
		outgoing = []string{}
	}

	// Incoming references (what references this pearl)
	incoming, err := store.DB().FindReferencingPearls(id)
	if err != nil {
		return fmt.Errorf("find referencing pearls: %w", err)
	}

	if refsJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]interface{}{
			"id":            id,
			"references":    outgoing,
			"referenced_by": incoming,
		})
	}

	// Table output
	fmt.Printf("%s\n", id)
	fmt.Printf("\n")

	if len(outgoing) == 0 && len(incoming) == 0 {
		fmt.Printf("No references.\n")
		return nil
	}

	if len(outgoing) > 0 {
		fmt.Printf("References (outgoing):\n")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, ref := range outgoing {
			// Try to get the referenced pearl for more info
			refPearl, _ := store.Get(ref)
			if refPearl != nil {
				desc := refPearl.Description
				if len(desc) > 40 {
					desc = desc[:37] + "..."
				}
				fmt.Fprintf(w, "  → %s\t%s\t%s\n", ref, refPearl.Type, desc)
			} else {
				fmt.Fprintf(w, "  → %s\t(not found)\t\n", ref)
			}
		}
		w.Flush()
	}

	if len(incoming) > 0 {
		if len(outgoing) > 0 {
			fmt.Printf("\n")
		}
		fmt.Printf("Referenced by (incoming):\n")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, ref := range incoming {
			refPearl, _ := store.Get(ref)
			if refPearl != nil {
				desc := refPearl.Description
				if len(desc) > 40 {
					desc = desc[:37] + "..."
				}
				fmt.Fprintf(w, "  ← %s\t%s\t%s\n", ref, refPearl.Type, desc)
			} else {
				fmt.Fprintf(w, "  ← %s\t\t\n", ref)
			}
		}
		w.Flush()
	}

	return nil
}
