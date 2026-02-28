package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/justrnr500/pearls/internal/storage"
)

var clutchCmd = &cobra.Command{
	Use:   "clutch",
	Short: "Output required pearls sorted by priority",
	Long: `Output concatenated content of all required pearls sorted by priority.

This command outputs pearls that have been marked as required, ordered
by priority (highest first). The output format matches the context command.

Examples:
  pearls clutch
  pearls clutch --brief
  pearls clutch --json`,
	RunE: runClutch,
}

var (
	clutchBrief bool
	clutchJSON  bool
)

func init() {
	rootCmd.AddCommand(clutchCmd)
	clutchCmd.Flags().BoolVar(&clutchBrief, "brief", false, "Only include metadata, not full content")
	clutchCmd.Flags().BoolVar(&clutchJSON, "json", false, "Output as JSON")
}

func runClutch(cmd *cobra.Command, args []string) error {
	store, _, err := getStore()
	if err != nil {
		return err
	}
	defer store.Close()

	reqTrue := true
	pearls, err := store.List(storage.ListOptions{Required: &reqTrue})
	if err != nil {
		return fmt.Errorf("list required pearls: %w", err)
	}

	if clutchJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]interface{}{
			"pearls": pearls,
			"count":  len(pearls),
		})
	}

	if len(pearls) == 0 {
		return nil
	}

	var output strings.Builder

	for i, p := range pearls {
		if i > 0 {
			output.WriteString("\n---\n\n")
		}

		if clutchBrief {
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
			content, err := store.GetContent(p)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not read content for %s: %v\n", p.ID, err)
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
