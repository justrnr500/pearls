package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show pearl details",
	Long: `Display detailed information about a pearl.

Examples:
  pearls show db.postgres.users
  pearls show db.postgres.users --json
  pearls show db.postgres.users --with-refs`,
	Args: cobra.ExactArgs(1),
	RunE: runShow,
}

var (
	showJSON     bool
	showWithRefs bool
)

func init() {
	rootCmd.AddCommand(showCmd)
	showCmd.Flags().BoolVar(&showJSON, "json", false, "Output as JSON")
	showCmd.Flags().BoolVar(&showWithRefs, "with-refs", false, "Include referenced pearls")
}

func runShow(cmd *cobra.Command, args []string) error {
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

	if showJSON {
		output := map[string]interface{}{
			"pearl": p,
		}

		if showWithRefs && len(p.References) > 0 {
			refs := []interface{}{}
			for _, refID := range p.References {
				ref, err := store.Get(refID)
				if err == nil && ref != nil {
					refs = append(refs, ref)
				}
			}
			output["references"] = refs
		}

		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(output)
	}

	// Human-readable output
	fmt.Printf("● %s\n", p.ID)
	fmt.Printf("  Name:        %s\n", p.Name)
	if p.Namespace != "" {
		fmt.Printf("  Namespace:   %s\n", p.Namespace)
	}
	fmt.Printf("  Type:        %s\n", p.Type)
	fmt.Printf("  Status:      %s\n", p.Status)

	if p.Description != "" {
		fmt.Printf("  Description: %s\n", p.Description)
	}

	if len(p.Tags) > 0 {
		fmt.Printf("  Tags:        %s\n", strings.Join(p.Tags, ", "))
	}

	if p.ContentPath != "" {
		fmt.Printf("  Content:     %s\n", p.ContentPath)
	}

	if p.Connection != nil {
		fmt.Printf("  Connection:\n")
		fmt.Printf("    Type:      %s\n", p.Connection.Type)
		if p.Connection.Host != "" {
			fmt.Printf("    Host:      %s\n", p.Connection.Host)
		}
		if p.Connection.Port > 0 {
			fmt.Printf("    Port:      %d\n", p.Connection.Port)
		}
		if p.Connection.Database != "" {
			fmt.Printf("    Database:  %s\n", p.Connection.Database)
		}
	}

	if len(p.References) > 0 {
		fmt.Printf("  References:\n")
		for _, ref := range p.References {
			fmt.Printf("    → %s\n", ref)
		}

		if showWithRefs {
			fmt.Printf("\n  Referenced Pearls:\n")
			for _, refID := range p.References {
				ref, err := store.Get(refID)
				if err != nil || ref == nil {
					fmt.Printf("    %s (not found)\n", refID)
					continue
				}
				fmt.Printf("    ● %s [%s] %s\n", ref.ID, ref.Type, ref.Description)
			}
		}
	}

	if p.Parent != "" {
		fmt.Printf("  Parent:      %s\n", p.Parent)
	}

	fmt.Printf("  Created:     %s by %s\n", p.CreatedAt.Format("2006-01-02 15:04"), p.CreatedBy)
	fmt.Printf("  Updated:     %s\n", p.UpdatedAt.Format("2006-01-02 15:04"))

	return nil
}
