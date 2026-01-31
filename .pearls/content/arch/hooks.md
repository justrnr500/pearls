# Claude Code Hook System

Pearls integrates with Claude Code via two hooks, installed by \`pearls onboard --hooks\`.

## Hooks

### SessionStart — \`pearls prime\`
- Fires at session start and after context compaction
- Outputs catalog summary, discovery triggers, and quick reference
- Adaptive: empty catalog gets creation guidance, large catalog gets search reminders
- Override: place \`.pearls/PRIME.md\` to customize output entirely

### UserPromptSubmit — \`pearls-context.sh\`
- Fires on each user prompt
- Reads git diff to find changed files
- Runs \`pearls context --for <file>\` for each changed file
- Outputs matching pearl content as additionalContext JSON

## Hook Registration

Both hooks are registered in \`.claude/settings.json\` under the \`hooks\` key. The \`registerHook()\` and \`registerSessionStartHook()\` functions in \`onboard.go\` handle deduplication.

## Templates

All hook content lives in \`internal/cmd/templates/\` as \`go:embed\` files:
- \`onboard.md\` — CLAUDE.md injection template
- \`hook-context.sh\` — UserPromptSubmit bash script
- \`prime-triggers.md\` — discovery trigger rules
- \`prime-reference.md\` — quick command reference

Edit the \`.md\`/\`.sh\` files directly; Go code just loads and outputs them.