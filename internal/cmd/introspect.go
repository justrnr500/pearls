package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"

	"github.com/justrnr500/pearls/internal/config"
	"github.com/justrnr500/pearls/internal/introspect"
)

var introspectCmd = &cobra.Command{
	Use:   "introspect <type>",
	Short: "Auto-generate pearls from database schemas",
	Long: `Connect to a live database and generate pearls for discovered schemas and tables.

Supported types: postgres, mysql, sqlite

Credentials are read from .env in the repo root.
Default env vars: PEARLS_POSTGRES_URL, PEARLS_MYSQL_URL, PEARLS_SQLITE_PATH

Examples:
  pearls introspect postgres --prefix db.postgres
  pearls introspect mysql --prefix db.mysql --schema mydb
  pearls introspect sqlite --prefix db.local
  pearls introspect postgres --env DATABASE_URL --prefix db.main
  pearls introspect postgres --prefix db.pg --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runIntrospect,
}

var (
	introspectPrefix       string
	introspectEnv          string
	introspectSchema       string
	introspectDryRun       bool
	introspectSkipExisting bool
)

func init() {
	rootCmd.AddCommand(introspectCmd)
	introspectCmd.Flags().StringVar(&introspectPrefix, "prefix", "", "Namespace prefix for generated pearls (required)")
	introspectCmd.Flags().StringVar(&introspectEnv, "env", "", "Override env var name for connection string")
	introspectCmd.Flags().StringVar(&introspectSchema, "schema", "", "Limit to a specific schema")
	introspectCmd.Flags().BoolVar(&introspectDryRun, "dry-run", false, "Print what would be created without writing")
	introspectCmd.Flags().BoolVar(&introspectSkipExisting, "skip-existing", false, "Don't overwrite pearls that already exist")
	introspectCmd.MarkFlagRequired("prefix")
}

func runIntrospect(cmd *cobra.Command, args []string) error {
	dbType := args[0]

	// Validate type
	switch dbType {
	case "postgres", "mysql", "sqlite":
		// ok
	default:
		return fmt.Errorf("unsupported database type %q: must be postgres, mysql, or sqlite", dbType)
	}

	// Load .env from repo root
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}
	root, err := config.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("find pearls root: %w", err)
	}

	envPath := filepath.Join(root, ".env")
	godotenv.Load(envPath) // Best effort — .env may not exist

	// Determine env var
	envVar := introspectEnv
	if envVar == "" {
		envVar = introspect.DefaultEnvVar(dbType)
	}
	if envVar == "" {
		return fmt.Errorf("could not determine env var for type %q", dbType)
	}

	connStr := os.Getenv(envVar)
	if connStr == "" {
		return fmt.Errorf("connection string not found: set %s in .env or environment", envVar)
	}

	// Create introspector
	var intro introspect.Introspector
	switch dbType {
	case "postgres":
		intro = &introspect.PostgresIntrospector{}
	case "mysql":
		intro = &introspect.MySQLIntrospector{}
	case "sqlite":
		intro = &introspect.SQLiteIntrospector{}
	}

	fmt.Printf("Connecting to %s...\n", dbType)
	if err := intro.Connect(connStr); err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer intro.Close()

	// Discover schemas
	schemas, err := intro.Schemas()
	if err != nil {
		return fmt.Errorf("discover schemas: %w", err)
	}

	if introspectSchema != "" {
		// Filter to specific schema
		found := false
		for _, s := range schemas {
			if s == introspectSchema {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("schema %q not found (available: %v)", introspectSchema, schemas)
		}
		schemas = []string{introspectSchema}
	}

	fmt.Printf("Found %d schema(s): %v\n", len(schemas), schemas)

	// Discover tables per schema
	allTables := make(map[string][]introspect.Table)
	totalTables := 0
	for _, schema := range schemas {
		tables, err := intro.Tables(schema)
		if err != nil {
			return fmt.Errorf("discover tables in %s: %w", schema, err)
		}
		allTables[schema] = tables
		totalTables += len(tables)
		fmt.Printf("  %s: %d table(s)\n", schema, len(tables))
	}

	// Generate pearls
	generated := introspect.GeneratePearls(introspectPrefix, allTables, envVar)

	if introspectDryRun {
		fmt.Printf("\nDry run — would create %d pearl(s):\n", len(generated))
		for _, gp := range generated {
			fmt.Printf("  %s (%s)\n", gp.Pearl.ID, gp.Pearl.Type)
		}
		return nil
	}

	// Get store
	store, _, err := getStore()
	if err != nil {
		return err
	}
	defer store.Close()

	// Create pearls
	created := 0
	skipped := 0
	for _, gp := range generated {
		// Check if pearl already exists
		existing, _ := store.Get(gp.Pearl.ID)
		if existing != nil {
			if introspectSkipExisting {
				skipped++
				continue
			}
			// Delete existing to recreate
			store.Delete(gp.Pearl.ID)
		}

		p := gp.Pearl
		content := gp.GeneratedContent
		if content == "" {
			content = "# " + p.Name + "\n"
		}
		if err := store.Create(&p, content); err != nil {
			return fmt.Errorf("create pearl %s: %w", gp.Pearl.ID, err)
		}
		created++
	}

	fmt.Printf("\n✓ Created %d pearl(s)", created)
	if skipped > 0 {
		fmt.Printf(" (%d skipped)", skipped)
	}
	fmt.Println()

	return nil
}
