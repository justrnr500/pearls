#!/usr/bin/env bash
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
