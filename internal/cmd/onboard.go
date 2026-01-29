package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var onboardCmd = &cobra.Command{
	Use:   "onboard",
	Short: "Inject agent instructions into project config files",
	Long: `Generate agent-facing instructions for using pearls and append them
to CLAUDE.md, agents.md, or both.

Examples:
  pearls onboard                    # Update CLAUDE.md (default)
  pearls onboard --target agents    # Update agents.md
  pearls onboard --target all       # Update both
  pearls onboard --force            # Overwrite existing pearls section`,
	RunE: runOnboard,
}

var (
	onboardTarget string
	onboardForce  bool
)

func init() {
	rootCmd.AddCommand(onboardCmd)
	onboardCmd.Flags().StringVar(&onboardTarget, "target", "claude", "Which file to update: claude, agents, or all")
	onboardCmd.Flags().BoolVar(&onboardForce, "force", false, "Overwrite existing pearls section")
}

func runOnboard(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	targets := map[string]string{
		"claude": "CLAUDE.md",
		"agents": "agents.md",
	}

	var files []string
	switch onboardTarget {
	case "claude":
		files = []string{targets["claude"]}
	case "agents":
		files = []string{targets["agents"]}
	case "all":
		files = []string{targets["claude"], targets["agents"]}
	default:
		return fmt.Errorf("invalid target %q: must be claude, agents, or all", onboardTarget)
	}

	for _, name := range files {
		path := filepath.Join(cwd, name)
		if err := onboardToFile(path, onboardForce); err != nil {
			return fmt.Errorf("onboard %s: %w", name, err)
		}
		fmt.Printf("✓ Updated %s\n", name)
	}

	return nil
}

const (
	markerStart = "<!-- pearls:start -->"
	markerEnd   = "<!-- pearls:end -->"
)

func onboardTemplate() string {
	return `<!-- pearls:start -->
## Pearls - Data Asset Memory

This project uses Pearls to document data assets (tables, schemas, APIs, pipelines, etc.).

### Quick Reference
- ` + "`pl list`" + ` — List all documented data assets
- ` + "`pl search \"query\"`" + ` — Keyword search
- ` + "`pl search \"query\" --semantic`" + ` — Natural language search
- ` + "`pl show <id>`" + ` — View asset metadata
- ` + "`pl cat <id>`" + ` — View full markdown documentation
- ` + "`pl context <ids...>`" + ` — Get concatenated docs for your context window
- ` + "`pl create <id> --type <type>`" + ` — Document a new asset
- ` + "`pl refs <id>`" + ` — See relationships
- ` + "`pl introspect <type> --prefix <ns>`" + ` — Auto-discover from database
- ` + "`pl doctor`" + ` — Check catalog health

### When to Use Pearls
- Before querying a database, check ` + "`pl search`" + ` for schema documentation
- When encountering unfamiliar data assets, check ` + "`pl show`" + `
- After discovering new data sources, create a pearl with ` + "`pl create`" + `
- When setting up a new database connection, run ` + "`pl introspect`" + ` to bootstrap docs
<!-- pearls:end -->`
}

func onboardToFile(path string, force bool) error {
	existing := ""
	if data, err := os.ReadFile(path); err == nil {
		existing = string(data)
	}

	startIdx := strings.Index(existing, markerStart)
	endIdx := strings.Index(existing, markerEnd)
	hasMarkers := startIdx >= 0 && endIdx >= 0 && endIdx > startIdx

	template := onboardTemplate()

	if hasMarkers && !force {
		return nil
	}

	var result string
	if hasMarkers && force {
		before := existing[:startIdx]
		after := existing[endIdx+len(markerEnd):]
		result = before + template + after
	} else {
		if existing != "" && !strings.HasSuffix(existing, "\n") {
			existing += "\n"
		}
		if existing != "" {
			existing += "\n"
		}
		result = existing + template + "\n"
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	return os.WriteFile(path, []byte(result), 0644)
}
