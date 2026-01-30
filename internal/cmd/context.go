package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var contextCmd = &cobra.Command{
	Use:   "context [id...]",
	Short: "Generate context for AI agents",
	Long: `Generate concatenated markdown context from one or more pearls.

This command outputs the content of multiple pearls in a format
suitable for injection into AI agent prompts.

Examples:
  pearls context db.postgres.users
  pearls context db.postgres.users db.postgres.orders
  pearls context db.postgres.users --with-refs
  pearls context --for src/api/handler.go
  pearls context --scope backend
  pearls context --for src/api/handler.go --scope backend`,
	RunE: runContext,
}

var (
	contextWithRefs bool
	contextBrief    bool
	contextFor      string
	contextScope    string
)

func init() {
	rootCmd.AddCommand(contextCmd)
	contextCmd.Flags().BoolVar(&contextWithRefs, "with-refs", false, "Include referenced pearls")
	contextCmd.Flags().BoolVar(&contextBrief, "brief", false, "Only include metadata, not full content")
	contextCmd.Flags().StringVar(&contextFor, "for", "", "File path (relative to repo root) to match pearls by glob")
	contextCmd.Flags().StringVar(&contextScope, "scope", "", "Scope name to match pearls")
}

func runContext(cmd *cobra.Command, args []string) error {
	if len(args) == 0 && contextFor == "" && contextScope == "" {
		return fmt.Errorf("at least one pearl ID, --for, or --scope must be provided")
	}

	store, _, err := getStore()
	if err != nil {
		return err
	}
	defer store.Close()

	// Collect all IDs to include
	ids := make([]string, 0, len(args))
	seen := make(map[string]bool)

	// Add requested IDs
	for _, id := range args {
		if !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}

	// Add pearls matched by --for flag
	if contextFor != "" {
		matched, err := store.FindByGlob(contextFor)
		if err != nil {
			return fmt.Errorf("find by glob: %w", err)
		}
		for _, p := range matched {
			if !seen[p.ID] {
				ids = append(ids, p.ID)
				seen[p.ID] = true
			}
		}
	}

	// Add pearls matched by --scope flag
	if contextScope != "" {
		matched, err := store.FindByScope(contextScope)
		if err != nil {
			return fmt.Errorf("find by scope: %w", err)
		}
		for _, p := range matched {
			if !seen[p.ID] {
				ids = append(ids, p.ID)
				seen[p.ID] = true
			}
		}
	}

	// Add referenced IDs if requested
	if contextWithRefs {
		for _, id := range args {
			p, err := store.Get(id)
			if err != nil || p == nil {
				continue
			}
			for _, refID := range p.References {
				if !seen[refID] {
					ids = append(ids, refID)
					seen[refID] = true
				}
			}
		}
	}

	// Generate output
	var output strings.Builder

	for i, id := range ids {
		p, err := store.Get(id)
		if err != nil {
			return fmt.Errorf("get pearl %s: %w", id, err)
		}
		if p == nil {
			fmt.Fprintf(os.Stderr, "Warning: pearl not found: %s\n", id)
			continue
		}

		if i > 0 {
			output.WriteString("\n---\n\n")
		}

		if contextBrief {
			// Brief mode: just metadata
			output.WriteString(fmt.Sprintf("## %s\n\n", p.ID))
			output.WriteString(fmt.Sprintf("- **Type:** %s\n", p.Type))
			output.WriteString(fmt.Sprintf("- **Status:** %s\n", p.Status))
			if p.Description != "" {
				output.WriteString(fmt.Sprintf("- **Description:** %s\n", p.Description))
			}
			if len(p.Tags) > 0 {
				output.WriteString(fmt.Sprintf("- **Tags:** %s\n", strings.Join(p.Tags, ", ")))
			}
			if p.Connection != nil {
				output.WriteString(fmt.Sprintf("- **Connection:** %s", p.Connection.Type))
				if p.Connection.Host != "" {
					output.WriteString(fmt.Sprintf(" @ %s", p.Connection.Host))
				}
				if p.Connection.Database != "" {
					output.WriteString(fmt.Sprintf("/%s", p.Connection.Database))
				}
				output.WriteString("\n")
			}
			output.WriteString("\n")
		} else {
			// Full mode: include content
			content, err := store.GetContent(p)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not read content for %s: %v\n", id, err)
				// Fall back to brief
				output.WriteString(fmt.Sprintf("## %s\n\n", p.ID))
				output.WriteString(fmt.Sprintf("%s\n\n", p.Description))
			} else {
				output.WriteString(content)
				if !strings.HasSuffix(content, "\n") {
					output.WriteString("\n")
				}
			}
		}
	}

	fmt.Print(output.String())
	return nil
}
