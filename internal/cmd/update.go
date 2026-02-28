package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/justrnr500/pearls/internal/pearl"
)

var updateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update pearl metadata",
	Long: `Update metadata for an existing pearl.

Examples:
  pearls update db.postgres.users --description "Updated description"
  pearls update db.postgres.users --add-tag sensitive
  pearls update db.postgres.users --remove-tag deprecated
  pearls update db.postgres.users --status deprecated
  pearls update db.postgres.users --add-ref db.postgres.organizations
  pearls update db.postgres.users --type view
  pearls update db.postgres.users --globs "src/models/**/*.go,db/migrations/*.sql"
  pearls update db.postgres.users --scopes backend,data-eng
  pearls update db.postgres.users --required --priority 10
  pearls update db.postgres.users --no-required`,
	Args: cobra.ExactArgs(1),
	RunE: runUpdate,
}

var (
	updateDescription string
	updateStatus      string
	updateType        string
	updateGlobs       string
	updateScopes      string
	updateAddTags     []string
	updateRemoveTags  []string
	updateAddRefs     []string
	updateRemoveRefs  []string
	updateJSON        bool
	updateRequired    bool
	updateNoRequired  bool
	updatePriority    int
)

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.Flags().StringVarP(&updateDescription, "description", "d", "", "Update description")
	updateCmd.Flags().StringVar(&updateStatus, "status", "", "Update status (active, deprecated, archived)")
	updateCmd.Flags().StringVarP(&updateType, "type", "t", "", "Update asset type (lowercase alphanumeric + hyphens)")
	updateCmd.Flags().StringVar(&updateGlobs, "globs", "", "Comma-separated file glob patterns for push-based context injection")
	updateCmd.Flags().StringVar(&updateScopes, "scopes", "", "Comma-separated scope names for scope-based injection")
	updateCmd.Flags().StringSliceVar(&updateAddTags, "add-tag", nil, "Add tag(s)")
	updateCmd.Flags().StringSliceVar(&updateRemoveTags, "remove-tag", nil, "Remove tag(s)")
	updateCmd.Flags().StringSliceVar(&updateAddRefs, "add-ref", nil, "Add reference(s)")
	updateCmd.Flags().StringSliceVar(&updateRemoveRefs, "remove-ref", nil, "Remove reference(s)")
	updateCmd.Flags().BoolVar(&updateJSON, "json", false, "Output as JSON")
	updateCmd.Flags().BoolVar(&updateRequired, "required", false, "Mark pearl as required context")
	updateCmd.Flags().BoolVar(&updateNoRequired, "no-required", false, "Mark pearl as not required context")
	updateCmd.Flags().IntVar(&updatePriority, "priority", 0, "Update priority ordering (higher = more important)")
}

func runUpdate(cmd *cobra.Command, args []string) error {
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

	// Track if anything changed
	changed := false

	// Update description
	if updateDescription != "" {
		p.Description = updateDescription
		changed = true
	}

	// Update status
	if updateStatus != "" {
		status := pearl.Status(updateStatus)
		if !status.IsValid() {
			return fmt.Errorf("invalid status %q: must be active, deprecated, or archived", updateStatus)
		}
		p.Status = status
		changed = true
	}

	// Update type (free-form: lowercase alphanumeric + hyphens)
	if cmd.Flags().Changed("type") {
		assetType := pearl.AssetType(updateType)
		if !assetType.IsValid() {
			return fmt.Errorf("invalid type %q: must be lowercase alphanumeric + hyphens, starting with a letter", updateType)
		}
		p.Type = assetType
		changed = true
	}

	// Update globs
	if cmd.Flags().Changed("globs") {
		var globs []string
		if updateGlobs != "" {
			globs = strings.Split(updateGlobs, ",")
		}
		if err := pearl.ValidateGlobs(globs); err != nil {
			return fmt.Errorf("invalid globs: %w", err)
		}
		p.Globs = globs
		changed = true
	}

	// Update scopes
	if cmd.Flags().Changed("scopes") {
		var scopes []string
		if updateScopes != "" {
			scopes = strings.Split(updateScopes, ",")
		}
		if err := pearl.ValidateScopes(scopes); err != nil {
			return fmt.Errorf("invalid scopes: %w", err)
		}
		p.Scopes = scopes
		changed = true
	}

	// Add tags
	if len(updateAddTags) > 0 {
		tagSet := make(map[string]bool)
		for _, t := range p.Tags {
			tagSet[t] = true
		}
		for _, t := range updateAddTags {
			if !tagSet[t] {
				p.Tags = append(p.Tags, t)
				tagSet[t] = true
			}
		}
		changed = true
	}

	// Remove tags
	if len(updateRemoveTags) > 0 {
		removeSet := make(map[string]bool)
		for _, t := range updateRemoveTags {
			removeSet[t] = true
		}
		newTags := []string{}
		for _, t := range p.Tags {
			if !removeSet[t] {
				newTags = append(newTags, t)
			}
		}
		p.Tags = newTags
		changed = true
	}

	// Add references
	if len(updateAddRefs) > 0 {
		refSet := make(map[string]bool)
		for _, r := range p.References {
			refSet[r] = true
		}
		for _, r := range updateAddRefs {
			if !refSet[r] {
				p.References = append(p.References, r)
				refSet[r] = true
			}
		}
		changed = true
	}

	// Update required
	if updateRequired && updateNoRequired {
		return fmt.Errorf("cannot use --required and --no-required together")
	}
	if updateRequired {
		p.Required = true
		changed = true
	}
	if updateNoRequired {
		p.Required = false
		changed = true
	}

	// Update priority
	if cmd.Flags().Changed("priority") {
		p.Priority = updatePriority
		changed = true
	}

	// Remove references
	if len(updateRemoveRefs) > 0 {
		removeSet := make(map[string]bool)
		for _, r := range updateRemoveRefs {
			removeSet[r] = true
		}
		newRefs := []string{}
		for _, r := range p.References {
			if !removeSet[r] {
				newRefs = append(newRefs, r)
			}
		}
		p.References = newRefs
		changed = true
	}

	if !changed {
		return fmt.Errorf("no updates specified")
	}

	p.UpdatedAt = time.Now()

	if err := store.Update(p, nil); err != nil {
		return fmt.Errorf("update pearl: %w", err)
	}

	if updateJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(p)
	}

	fmt.Printf("✓ Updated pearl: %s\n", p.ID)
	return nil
}
