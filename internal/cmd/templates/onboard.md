<!-- pearls:start -->
## Pearls - Semantic Context Injection

This project uses Pearls to store and inject knowledge into your sessions — data schemas, API docs, codebase conventions, architectural decisions, brainstorms, and more.

### Context Retrieval

**Push (automatic context based on what you're working on):**
- `pl context --for <path>` — Get context matching a file path (uses pearl glob patterns)
- `pl context --scope <scope>` — Get context for a domain/scope
- `pl context --for <path> --scope <scope>` — Combine both (union)

**Pull (search for what you need):**
- `pl search "query" --semantic` — Natural language search
- `pl search "query"` — Keyword search
- `pl context <ids...>` — Get specific pearls by ID

### Managing Knowledge
- `pl create <id> --type <type>` — Create a pearl (type is free-form: table, api, convention, brainstorm, runbook, etc.)
- `pl create <id> --type convention --globs "src/**/*.ts" --scopes "error-handling"` — With push triggers
- `pl create <id> --type brainstorm --content "# Design\n\nKey decisions..."` — Inline content (no editor needed)
- `echo "..." | pl create <id> --type brainstorm --content -` — Content from stdin
- `pl update <id> --globs "src/payments/**" --scopes "payments"` — Add globs/scopes to existing pearl
- `pl list` — List all pearls
- `pl list --scope payments` — List by scope
- `pl show <id>` — View pearl details
- `pl cat <id>` — View full markdown content
- `pl refs <id>` — See relationships
- `pl introspect <db> --prefix <ns>` — Auto-discover from database
- `pl doctor` — Check catalog health

### When to Use Pearls
- Before working on a feature, run `pl context --for <file>` to get relevant context
- Before querying a database, run `pl search` for schema documentation
- After a brainstorm or design session, save it with `pl create` so it persists across sessions
- When documenting conventions, attach globs so agents automatically get them in the right directories
- When setting up a new database connection, run `pl introspect` to bootstrap docs
<!-- pearls:end -->