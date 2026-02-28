package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/justrnr500/pearls/internal/pearl"
)

var createCmd = &cobra.Command{
	Use:   "create <id>",
	Short: "Create a new pearl",
	Long: `Create a new pearl to document a data asset.

The ID should be a dot-separated namespace path like:
  db.postgres.users
  api.stripe.customers
  warehouse.snowflake.analytics.orders

Examples:
  pearls create db.postgres.users --type table
  pearls create api.stripe.customers --type api -d "Stripe customer records"
  pearls create db.postgres.orders --type table --tag pii --tag core
  pearls create db.postgres.users --type table --required --priority 10`,
	Args: cobra.ExactArgs(1),
	RunE: runCreate,
}

var (
	createType        string
	createDescription string
	createTags        []string
	createGlobs       string
	createScopes      string
	createContent     string
	createJSON        bool
	createRequired    bool
	createPriority    int
)

func init() {
	rootCmd.AddCommand(createCmd)
	createCmd.Flags().StringVarP(&createType, "type", "t", "table", "Asset type (table, schema, database, api, endpoint, file, bucket, pipeline, dashboard, query, custom)")
	createCmd.Flags().StringVarP(&createDescription, "description", "d", "", "Brief description")
	createCmd.Flags().StringSliceVar(&createTags, "tag", nil, "Tags (can be repeated)")
	createCmd.Flags().StringVar(&createGlobs, "globs", "", "Comma-separated file glob patterns for push-based context injection")
	createCmd.Flags().StringVar(&createScopes, "scopes", "", "Comma-separated scope names for scope-based injection")
	createCmd.Flags().StringVar(&createContent, "content", "", `Inline content (use "-" to read from stdin)`)
	createCmd.Flags().BoolVar(&createJSON, "json", false, "Output as JSON")
	createCmd.Flags().BoolVar(&createRequired, "required", false, "Mark pearl as required context")
	createCmd.Flags().IntVar(&createPriority, "priority", 0, "Priority ordering (higher = more important)")
}

func runCreate(cmd *cobra.Command, args []string) error {
	id := args[0]

	// Validate ID as namespace
	if err := pearl.ValidateNamespace(id); err != nil {
		return fmt.Errorf("invalid ID %q: %w", id, err)
	}

	// Validate type (free-form: lowercase alphanumeric + hyphens)
	assetType := pearl.AssetType(createType)
	if !assetType.IsValid() {
		return fmt.Errorf("invalid type %q: must be lowercase alphanumeric + hyphens, starting with a letter", createType)
	}

	store, _, err := getStore()
	if err != nil {
		return err
	}
	defer store.Close()

	// Check if pearl already exists
	existing, err := store.Get(id)
	if err != nil {
		return fmt.Errorf("check existing: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("pearl %q already exists", id)
	}

	// Parse globs and scopes from comma-separated strings
	var globs []string
	if createGlobs != "" {
		globs = strings.Split(createGlobs, ",")
	}
	var scopes []string
	if createScopes != "" {
		scopes = strings.Split(createScopes, ",")
	}

	// Validate globs and scopes
	if err := pearl.ValidateGlobs(globs); err != nil {
		return fmt.Errorf("invalid --globs: %w", err)
	}
	if err := pearl.ValidateScopes(scopes); err != nil {
		return fmt.Errorf("invalid --scopes: %w", err)
	}

	// Parse namespace and name from ID
	namespace := pearl.ParentNamespace(id)
	name := pearl.LastSegment(id)

	now := time.Now()
	createdBy := os.Getenv("USER")
	if createdBy == "" {
		createdBy = "unknown"
	}

	p := &pearl.Pearl{
		ID:          id,
		Name:        name,
		Namespace:   namespace,
		Type:        assetType,
		Tags:        createTags,
		Globs:       globs,
		Scopes:      scopes,
		Description: createDescription,
		Required:    createRequired,
		Priority:    createPriority,
		Status:      pearl.StatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
		CreatedBy:   createdBy,
	}

	// Determine content: inline flag, stdin, or template
	var content string
	switch {
	case createContent == "-":
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}
		content = string(data)
	case createContent != "":
		content = expandEscapes(createContent)
	default:
		content = store.Content().Template(p)
	}

	// Create the pearl
	if err := store.Create(p, content); err != nil {
		return fmt.Errorf("create pearl: %w", err)
	}

	if createJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(p)
	}

	fmt.Printf("✓ Created pearl: %s\n", p.ID)
	fmt.Printf("  Content: %s\n", p.ContentPath)
	return nil
}

// expandEscapes replaces literal \n with real newlines and \t with real tabs.
// Agents pass single-line strings with escaped newlines.
func expandEscapes(s string) string {
	s = strings.ReplaceAll(s, `\n`, "\n")
	s = strings.ReplaceAll(s, `\t`, "\t")
	return s
}
