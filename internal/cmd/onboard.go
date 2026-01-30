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
## Pearls - Semantic Context Injection

This project uses Pearls to store and inject knowledge into your sessions — data schemas, API docs, codebase conventions, architectural decisions, brainstorms, and more.

### Context Retrieval

**Push (automatic context based on what you're working on):**
- ` + "`pl context --for <path>`" + ` — Get context matching a file path (uses pearl glob patterns)
- ` + "`pl context --scope <scope>`" + ` — Get context for a domain/scope
- ` + "`pl context --for <path> --scope <scope>`" + ` — Combine both (union)

**Pull (search for what you need):**
- ` + "`pl search \"query\" --semantic`" + ` — Natural language search
- ` + "`pl search \"query\"`" + ` — Keyword search
- ` + "`pl context <ids...>`" + ` — Get specific pearls by ID

### Managing Knowledge
- ` + "`pl create <id> --type <type>`" + ` — Create a pearl (type is free-form: table, api, convention, brainstorm, runbook, etc.)
- ` + "`pl create <id> --type convention --globs \"src/**/*.ts\" --scopes \"error-handling\"`" + ` — With push triggers
- ` + "`pl update <id> --globs \"src/payments/**\" --scopes \"payments\"`" + ` — Add globs/scopes to existing pearl
- ` + "`pl list`" + ` — List all pearls
- ` + "`pl list --scope payments`" + ` — List by scope
- ` + "`pl show <id>`" + ` — View pearl details
- ` + "`pl cat <id>`" + ` — View full markdown content
- ` + "`pl refs <id>`" + ` — See relationships
- ` + "`pl introspect <db> --prefix <ns>`" + ` — Auto-discover from database
- ` + "`pl doctor`" + ` — Check catalog health

### When to Use Pearls
- Before working on a feature, run ` + "`pl context --for <file>`" + ` to get relevant context
- Before querying a database, run ` + "`pl search`" + ` for schema documentation
- After a brainstorm or design session, save it with ` + "`pl create`" + ` so it persists across sessions
- When documenting conventions, attach globs so agents automatically get them in the right directories
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
