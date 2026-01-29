package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/justrnr500/pearls/internal/config"
	"github.com/justrnr500/pearls/internal/storage"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new pearls data catalog",
	Long: `Initialize a new pearls data catalog in the current directory.

This creates a .pearls/ directory with:
  - config.yaml   Configuration file
  - pearls.jsonl  Metadata (git-tracked)
  - pearls.db     SQLite cache (gitignored)
  - content/      Markdown content files
  - .gitignore    Ignores the database file`,
	RunE: runInit,
}

var (
	initQuiet bool
	initName  string
)

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVarP(&initQuiet, "quiet", "q", false, "Suppress output (for agents)")
	initCmd.Flags().StringVarP(&initName, "name", "n", "", "Project name")
}

func runInit(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	// Check if already initialized
	if config.Exists(cwd) {
		if !initQuiet {
			fmt.Println("Already initialized in", filepath.Join(cwd, config.DirName))
		}
		return nil
	}

	paths := config.ResolvePaths(cwd)

	// Create directory structure
	dirs := []string{
		paths.Root,
		paths.Content,
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	if !initQuiet {
		fmt.Println("✓ Created", paths.Root)
	}

	// Create config.yaml
	cfg := config.Default()
	if initName != "" {
		cfg.Project.Name = initName
	} else {
		// Use directory name as default project name
		cfg.Project.Name = filepath.Base(cwd)
	}

	if err := cfg.Save(paths.Config); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	if !initQuiet {
		fmt.Println("✓ Created", config.ConfigFile)
	}

	// Initialize SQLite database
	db, err := storage.OpenDB(paths.DB)
	if err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}
	db.Close()

	if !initQuiet {
		fmt.Println("✓ Initialized SQLite database")
	}

	// Create empty JSONL file
	if err := os.WriteFile(paths.JSONL, []byte{}, 0644); err != nil {
		return fmt.Errorf("create jsonl file: %w", err)
	}

	if !initQuiet {
		fmt.Println("✓ Created", config.JSONLFile)
	}

	// Create .gitignore
	gitignore := `# Pearls - SQLite database (local cache, rebuilt from jsonl)
pearls.db
pearls.db-shm
pearls.db-wal
`
	gitignorePath := filepath.Join(paths.Root, config.GitIgnoreFile)
	if err := os.WriteFile(gitignorePath, []byte(gitignore), 0644); err != nil {
		return fmt.Errorf("create gitignore: %w", err)
	}

	if !initQuiet {
		fmt.Println("✓ Created .gitignore")
	}

	// Ensure .env is in repo root .gitignore
	rootGitignore := filepath.Join(cwd, ".gitignore")
	ensureGitignoreEntry(rootGitignore, ".env")

	if !initQuiet {
		fmt.Println("\nReady to track data assets. Try:")
		fmt.Println("  pearls create db.postgres.users --type table")
	}

	return nil
}

// ensureGitignoreEntry ensures that the given entry exists in the gitignore file
// at path. If the file does not exist, it is created. If the entry already
// exists (compared after trimming whitespace), no changes are made.
func ensureGitignoreEntry(path, entry string) {
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == entry {
			return
		}
	}

	// Append entry on a new line
	content := string(data)
	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += entry + "\n"

	os.WriteFile(path, []byte(content), 0644)
}
