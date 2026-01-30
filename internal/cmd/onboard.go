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

Use --hooks to install a Claude Code hook script that automatically injects
relevant pearl context based on your current git changes.

Examples:
  pearls onboard                    # Update CLAUDE.md (default)
  pearls onboard --target agents    # Update agents.md
  pearls onboard --target all       # Update both
  pearls onboard --force            # Overwrite existing pearls section
  pearls onboard --hooks            # Set up Claude Code context hook`,
	RunE: runOnboard,
}

var (
	onboardTarget string
	onboardForce  bool
	onboardHooks  bool
)

func init() {
	rootCmd.AddCommand(onboardCmd)
	onboardCmd.Flags().StringVar(&onboardTarget, "target", "claude", "Which file to update: claude, agents, or all")
	onboardCmd.Flags().BoolVar(&onboardForce, "force", false, "Overwrite existing pearls section")
	onboardCmd.Flags().BoolVar(&onboardHooks, "hooks", false, "Set up Claude Code hooks for automatic context injection")
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

	if onboardHooks {
		if err := setupClaudeHooks(cwd); err != nil {
			return fmt.Errorf("setup hooks: %w", err)
		}
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
- ` + "`pl create <id> --type brainstorm --content \"# Design\\n\\nKey decisions...\"`" + ` — Inline content (no editor needed)
- ` + "`echo \"...\" | pl create <id> --type brainstorm --content -`" + ` — Content from stdin
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

func setupClaudeHooks(projectRoot string) error {
	hooksDir := filepath.Join(projectRoot, ".claude", "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("create hooks directory: %w", err)
	}

	scriptPath := filepath.Join(hooksDir, "context-inject.sh")
	if err := os.WriteFile(scriptPath, []byte(hookScript()), 0755); err != nil {
		return fmt.Errorf("write hook script: %w", err)
	}

	fmt.Printf("✓ Created hook script: %s\n", scriptPath)
	fmt.Println()
	fmt.Println("To register this hook, add the following to your Claude Code settings")
	fmt.Println("(.claude/settings.json or equivalent):")
	fmt.Println()
	fmt.Printf("  Hook command: %s\n", scriptPath)
	fmt.Println("  Event: PreToolUse (or your preferred trigger)")
	fmt.Println()

	return nil
}

func hookScript() string {
	return "#!/usr/bin/env bash\n" +
		"set -euo pipefail\n" +
		"\n" +
		"# Pearls context injection hook for Claude Code.\n" +
		"# Automatically injects relevant pearl context based on current git changes.\n" +
		"# All errors are suppressed so this hook never breaks the agent.\n" +
		"\n" +
		"{\n" +
		"  REPO_ROOT=\"$(git rev-parse --show-toplevel 2>/dev/null)\" || exit 0\n" +
		"\n" +
		"  # Gather changed files (unstaged + staged), deduplicated.\n" +
		"  FILES=\"$(\n" +
		"    { git diff --name-only HEAD 2>/dev/null; git diff --name-only --cached 2>/dev/null; } \\\n" +
		"      | sort -u \\\n" +
		"      | head -20\n" +
		"  )\"\n" +
		"\n" +
		"  # Nothing changed — nothing to inject.\n" +
		"  if [ -z \"$FILES\" ]; then\n" +
		"    exit 0\n" +
		"  fi\n" +
		"\n" +
		"  OUTPUT=\"\"\n" +
		"  while IFS= read -r FILE; do\n" +
		"    RESULT=\"$(pearls context --for \"$FILE\" 2>/dev/null)\" || true\n" +
		"    if [ -n \"$RESULT\" ]; then\n" +
		"      OUTPUT=\"${OUTPUT}${RESULT}\"$'\\n'\n" +
		"    fi\n" +
		"  done <<< \"$FILES\"\n" +
		"\n" +
		"  if [ -n \"$OUTPUT\" ]; then\n" +
		"    echo \"<!-- pearls: auto-injected context -->\"\n" +
		"    echo \"$OUTPUT\"\n" +
		"    echo \"<!-- /pearls -->\"\n" +
		"  fi\n" +
		"} 2>/dev/null\n"
}
