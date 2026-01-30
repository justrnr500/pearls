package cmd

import (
	"encoding/json"
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
	// 1. Write the hook script
	hooksDir := filepath.Join(projectRoot, ".claude", "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("create hooks directory: %w", err)
	}

	scriptPath := filepath.Join(hooksDir, "pearls-context.sh")
	if err := os.WriteFile(scriptPath, []byte(hookScript()), 0755); err != nil {
		return fmt.Errorf("write hook script: %w", err)
	}
	fmt.Printf("✓ Created hook script: %s\n", scriptPath)

	// 2. Register the hook in .claude/settings.json
	settingsPath := filepath.Join(projectRoot, ".claude", "settings.json")
	if err := registerHook(settingsPath, scriptPath); err != nil {
		return fmt.Errorf("register hook: %w", err)
	}
	fmt.Printf("✓ Registered UserPromptSubmit hook in %s\n", settingsPath)

	return nil
}

// registerHook adds the pearls hook to .claude/settings.json, merging with
// any existing configuration.
func registerHook(settingsPath, scriptPath string) error {
	// Read existing settings or start fresh
	settings := make(map[string]interface{})
	if data, err := os.ReadFile(settingsPath); err == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("parse existing settings: %w", err)
		}
	}

	// Build the hook entry
	hookEntry := map[string]interface{}{
		"type":    "command",
		"command": scriptPath,
		"timeout": 10,
	}

	hookGroup := map[string]interface{}{
		"hooks": []interface{}{hookEntry},
	}

	// Get or create the hooks map
	hooks, _ := settings["hooks"].(map[string]interface{})
	if hooks == nil {
		hooks = make(map[string]interface{})
	}

	// Get or create the UserPromptSubmit array
	existing, _ := hooks["UserPromptSubmit"].([]interface{})

	// Check if pearls hook is already registered (avoid duplicates)
	alreadyRegistered := false
	for _, entry := range existing {
		group, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		hookList, _ := group["hooks"].([]interface{})
		for _, h := range hookList {
			hook, ok := h.(map[string]interface{})
			if !ok {
				continue
			}
			cmd, _ := hook["command"].(string)
			if strings.Contains(cmd, "pearls-context") {
				alreadyRegistered = true
				break
			}
		}
	}

	if !alreadyRegistered {
		existing = append(existing, hookGroup)
	}

	hooks["UserPromptSubmit"] = existing
	settings["hooks"] = hooks

	// Write back
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}

	return os.WriteFile(settingsPath, append(data, '\n'), 0644)
}

func hookScript() string {
	return `#!/usr/bin/env bash
# Pearls context injection hook for Claude Code.
# Runs on UserPromptSubmit — reads hook input from stdin,
# gathers context for recently changed files, outputs JSON
# with additionalContext for injection into the agent's session.
#
# Exit 0 always — hooks must never block the agent.

{
  # Read stdin (hook input JSON) — we don't use it yet but must consume it
  cat > /dev/null

  REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null)" || exit 0
  cd "$REPO_ROOT"

  # Gather changed files (unstaged + staged), deduplicated, max 20
  FILES="$(
    { git diff --name-only HEAD 2>/dev/null; git diff --name-only --cached 2>/dev/null; } \
      | sort -u \
      | head -20
  )"

  # Nothing changed — nothing to inject
  [ -z "$FILES" ] && exit 0

  CONTEXT=""
  while IFS= read -r FILE; do
    RESULT="$(pearls context --for "$FILE" 2>/dev/null)" || true
    if [ -n "$RESULT" ]; then
      CONTEXT="${CONTEXT}${RESULT}"$'\n'
    fi
  done <<< "$FILES"

  # No matching pearls — exit cleanly
  [ -z "$CONTEXT" ] && exit 0

  # Output structured JSON for Claude Code context injection
  python3 -c "
import json, sys
ctx = sys.stdin.read()
print(json.dumps({
  'hookSpecificOutput': {
    'hookEventName': 'UserPromptSubmit',
    'additionalContext': ctx.strip()
  }
}))
" <<< "$CONTEXT"

} 2>/dev/null

exit 0
`
}
