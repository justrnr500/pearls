package cmd

import (
	"encoding/json"
	"fmt"
	"os"
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
  pearls create db.postgres.orders --type table --tag pii --tag core`,
	Args: cobra.ExactArgs(1),
	RunE: runCreate,
}

var (
	createType        string
	createDescription string
	createTags        []string
	createJSON        bool
)

func init() {
	rootCmd.AddCommand(createCmd)
	createCmd.Flags().StringVarP(&createType, "type", "t", "table", "Asset type (table, schema, database, api, endpoint, file, bucket, pipeline, dashboard, query, custom)")
	createCmd.Flags().StringVarP(&createDescription, "description", "d", "", "Brief description")
	createCmd.Flags().StringSliceVar(&createTags, "tag", nil, "Tags (can be repeated)")
	createCmd.Flags().BoolVar(&createJSON, "json", false, "Output as JSON")
}

func runCreate(cmd *cobra.Command, args []string) error {
	id := args[0]

	// Validate ID as namespace
	if err := pearl.ValidateNamespace(id); err != nil {
		return fmt.Errorf("invalid ID %q: %w", id, err)
	}

	// Validate type
	assetType := pearl.AssetType(createType)
	if !assetType.IsValid() {
		return fmt.Errorf("invalid type %q: must be one of: table, schema, database, api, endpoint, file, bucket, pipeline, dashboard, query, custom", createType)
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
		Description: createDescription,
		Status:      pearl.StatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
		CreatedBy:   createdBy,
	}

	// Generate template content
	content := store.Content().Template(p)

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
