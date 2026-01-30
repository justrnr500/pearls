# Pearls

Semantic context injection for AI agents.

Pearls is a CLI tool for storing, searching, and injecting knowledge into AI agent sessions. It catalogs anything an agent might need -- data schemas, API docs, codebase conventions, architectural decisions, brainstorms, scripts, runbooks -- as searchable markdown. Agents retrieve context via semantic search (pull) or receive it automatically based on file paths and scopes (push).

> Beads gives agents memory about *tasks*. Pearls gives agents memory about *everything else*.

## Features

- **Markdown-native** -- Pearl content is plain markdown. Human-readable, version-controllable, LLM-friendly.
- **Git as database** -- JSONL metadata + markdown content travel with your code. No server required.
- **Agent-first** -- Every command supports `--json` output. Search is fast and semantic.
- **Push-based context** -- Attach glob patterns and scopes to pearls. Agents get relevant context injected based on what files they're touching or what domain they're working in. Replaces scattered `agent.md` files.
- **Semantic search** -- Natural language queries powered by local embeddings (all-MiniLM-L6-v2)
- **Free-form types** -- Not just data assets. Store conventions, brainstorms, API docs, runbooks, decisions -- any knowledge worth preserving.
- **Hierarchical** -- Dot-separated namespaces: `db.postgres.users`, `api.stripe.customers`
- **Relationship tracking** -- Pearls reference other pearls, creating a navigable graph
- **Database introspection** -- Auto-generate pearls from live Postgres, MySQL, or SQLite databases
- **Health checks** -- `pearls doctor` validates catalog integrity (sync, references, orphans)

## Install

```bash
go install github.com/justrnr500/pearls/cmd/pearls@latest
```

A shorter alias is also available:

```bash
go install github.com/justrnr500/pearls/cmd/pl@latest
```

## Quick Start

```bash
# Initialize in your project
pearls init

# Create a pearl for a table
pearls create db.postgres.users --type table -d "Core user accounts"

# Create a convention pearl with globs (push-based context)
pearls create conventions.error-handling --type convention \
  -d "How we handle errors in this codebase" \
  --globs "src/**/*.ts" --scopes "conventions"

# Create an API pearl scoped to payment code
pearls create api.stripe.customers --type api \
  -d "Stripe customer API integration" \
  --globs "src/payments/**,src/billing/**" --scopes "payments,stripe"

# Edit the generated markdown
$EDITOR .pearls/content/conventions/error-handling.md

# Or auto-generate from a live database
pearls introspect postgres --prefix db.postgres

# Pull: semantic search
pearls search "where is payment info stored" --semantic

# Push: get context for a file path (matches globs)
pearls context --for src/payments/checkout.ts

# Push: get context for a scope
pearls context --scope payments

# Check catalog health
pearls doctor

# Commit to git
git add .pearls/
git commit -m "Add knowledge catalog"
```

## Commands

### `pearls init`

Initialize a new data catalog in the current directory.

```bash
pearls init
pearls init --name my-project
pearls init --quiet            # Suppress output (for agents)
```

Creates `.pearls/` with config, SQLite cache, JSONL metadata, and content directory.

### `pearls create`

Create a new pearl.

```bash
pearls create db.postgres.users --type table
pearls create api.stripe.customers --type api -d "Stripe customer records"
pearls create conventions.error-handling --type convention \
  --globs "src/**/*.ts" --scopes "conventions"
pearls create decisions.auth-redesign --type brainstorm \
  --scopes "auth,architecture" -d "Auth system redesign discussion"

# Inline content (no template, no editor)
pearls create conventions.logging --type convention \
  --content "# Logging\n\nUse structured logging with context fields."

# Content from stdin (pipe from another command)
cat design-notes.md | pearls create decisions.auth --type brainstorm --content -
```

**Flags:**
- `--type, -t` -- Pearl type. Free-form string (lowercase alphanumeric + hyphens). Common types: `table`, `schema`, `api`, `convention`, `brainstorm`, `runbook`, `decision`, `script` (default: `table`)
- `--description, -d` -- Brief description
- `--tag` -- Tags (repeatable)
- `--globs` -- Comma-separated file glob patterns for push-based context injection (e.g., `"src/payments/**,src/billing/**"`)
- `--scopes` -- Comma-separated scope names for scope-based injection (e.g., `"payments,stripe"`)
- `--content` -- Inline content string. Supports `\n` for newlines. Use `--content -` to read from stdin. Skips template generation.
- `--json` -- JSON output

### `pearls show`

Display detailed information about a pearl.

```bash
pearls show db.postgres.users
pearls show db.postgres.users --with-refs
pearls show db.postgres.users --json
```

### `pearls list`

List pearls with optional filtering.

```bash
pearls list
pearls list --type table
pearls list --type convention
pearls list --namespace db.postgres
pearls list --tag pii
pearls list --scope payments
pearls list --status active
pearls list --json
```

**Aliases:** `pearls ls`

### `pearls cat`

Display the raw markdown content of a pearl.

```bash
pearls cat db.postgres.users
```

### `pearls search`

Search by keyword or semantic similarity.

```bash
# Keyword search (default)
pearls search customer
pearls search "user email" --type table
pearls search orders --tag analytics --json

# Semantic search (natural language)
pearls search "where is payment data stored" --semantic
pearls search "tables with PII" --semantic
```

**Flags:**
- `--semantic` -- Use vector similarity instead of keyword matching
- `--type, -t` -- Filter by type
- `--status, -s` -- Filter by status
- `--tag` -- Filter by tag
- `--limit` -- Maximum results (default: 50)
- `--json` -- JSON output

Semantic search uses all-MiniLM-L6-v2 embeddings (downloaded automatically on first use, ~90MB, cached at `~/.pearls/models/`).

### `pearls update`

Update pearl metadata.

```bash
pearls update db.postgres.users -d "Updated description"
pearls update db.postgres.users --add-tag sensitive
pearls update db.postgres.users --remove-tag deprecated
pearls update db.postgres.users --status deprecated
pearls update db.postgres.users --add-ref db.postgres.organizations
pearls update db.postgres.users --type convention
pearls update db.postgres.users --globs "src/models/user/**"
pearls update db.postgres.users --scopes "users,auth"
```

### `pearls delete`

Delete or archive a pearl.

```bash
# Archive (soft delete, default)
pearls delete db.postgres.old_table
pearls archive db.postgres.old_table

# Permanently delete
pearls delete db.postgres.old_table --force

# Delete entire namespace
pearls delete db.legacy --recursive --force
```

### `pearls refs`

Show bidirectional relationships for a pearl.

```bash
pearls refs db.postgres.orders
pearls refs db.postgres.orders --json
```

Output:
```
db.postgres.orders

References (outgoing):
  → db.postgres.users        table  User accounts
  → db.postgres.products     table  Product catalog

Referenced by (incoming):
  ← db.postgres.order_items  table  Line items per order
```

### `pearls context`

Generate concatenated markdown for AI agent prompts. Supports both pull (by ID) and push (by file path or scope) retrieval.

```bash
# Pull: request specific pearls by ID
pearls context db.postgres.users db.postgres.orders
pearls context db.postgres.users --with-refs   # Include referenced pearls
pearls context db.postgres.users --brief       # Metadata only, no content

# Push: get context for a file path (matches pearl globs)
pearls context --for src/payments/checkout.ts

# Push: get context for a scope
pearls context --scope payments

# Push: combine path and scope (union of results)
pearls context --for src/payments/checkout.ts --scope auth
```

**Flags:**
- `--for` -- File path (relative to repo root) to match against pearl glob patterns
- `--scope` -- Scope name to match against pearl scopes
- `--with-refs` -- Include referenced pearls
- `--brief` -- Metadata only, no markdown content

### `pearls sync`

Synchronize SQLite cache with JSONL source of truth.

```bash
pearls sync
```

Run after pulling from git to rebuild the local database from the JSONL file.

### `pearls index`

Manage the vector search index.

```bash
# Show index status
pearls index

# Rebuild all embeddings
pearls index --rebuild
```

Use `--rebuild` after initial setup, model upgrades, or to index existing pearls that don't have embeddings yet.

### `pearls introspect`

Auto-generate pearls from a live database.

```bash
pearls introspect postgres --prefix db.postgres
pearls introspect mysql --prefix db.mysql --schema mydb
pearls introspect sqlite --prefix db.local
pearls introspect postgres --env DATABASE_URL --prefix db.main
pearls introspect postgres --prefix db.pg --dry-run
```

**Flags:**
- `--prefix` -- Namespace prefix for generated pearls (required)
- `--schema` -- Limit to a specific schema
- `--env` -- Override env var name for connection string
- `--dry-run` -- Print what would be created without writing
- `--skip-existing` -- Don't overwrite pearls that already exist

Supported databases: **PostgreSQL**, **MySQL**, **SQLite**.

Credentials are read from `.env` in the repo root. Default env vars: `PEARLS_POSTGRES_URL`, `PEARLS_MYSQL_URL`, `PEARLS_SQLITE_PATH`.

Introspection discovers schemas, tables, columns, foreign keys, and indexes. Foreign keys are automatically converted to pearl references.

### `pearls doctor`

Run health checks on the catalog.

```bash
pearls doctor
pearls doctor --json
```

Checks:
- JSONL/SQLite sync (pearl counts and IDs match)
- Orphaned content (markdown files with no pearl)
- Missing content (pearls with content_path that doesn't exist)
- Broken references (pearls referencing IDs that don't exist)
- Config validity (config.yaml parses without errors)

### `pearls onboard`

Inject agent instructions into project config files, and optionally set up automatic context injection hooks.

```bash
pearls onboard                    # Update CLAUDE.md (default)
pearls onboard --target agents    # Update agents.md
pearls onboard --target all       # Update both
pearls onboard --force            # Overwrite existing pearls section
pearls onboard --hooks            # Set up Claude Code context injection hook
```

The `--hooks` flag creates `.claude/hooks/context-inject.sh`, a shell script that automatically injects relevant pearl context based on your current git changes. Register it as a Claude Code hook for push-based context delivery.

## Directory Structure

```
.pearls/
├── config.yaml         # Configuration
├── pearls.jsonl        # Metadata source of truth (git-tracked)
├── pearls.db           # SQLite cache (gitignored)
├── content/            # Markdown content (git-tracked)
│   ├── db/
│   │   └── postgres/
│   │       ├── users.md
│   │       └── orders.md
│   └── api/
│       └── stripe/
│           └── customers.md
└── .gitignore
```

- **pearls.jsonl** -- Git-tracked source of truth for metadata
- **pearls.db** -- SQLite cache for fast queries (rebuilt from JSONL)
- **content/** -- Markdown files mirroring namespace hierarchy

## Configuration

`.pearls/config.yaml`:

```yaml
project:
  name: my-data-catalog
  description: Data asset catalog
storage:
  content_dir: content
defaults:
  status: active
  created_by: ${USER}
vector_search:
  enabled: true
  model_path: ~/.pearls/models    # Shared across projects
aliases: {}
```

### Vector Search

Vector search is enabled by default. The embedding model (`all-MiniLM-L6-v2`) downloads automatically on first use to `~/.pearls/models/` (shared across all projects, ~90MB).

To disable:

```yaml
vector_search:
  enabled: false
```

## Agent Integration

### Three Retrieval Layers

| Layer | Mechanism | Use case |
|-------|-----------|----------|
| Pull  | Semantic search | Agent asks "what do I need to know about X?" |
| Push  | Glob matching | Agent working in a file path gets relevant context |
| Push  | Scope matching | Agent declares what domain it's working in |

### Push: Replace Scattered agent.md Files

Instead of maintaining `CLAUDE.md` or `agent.md` files in every directory, attach glob patterns to pearls. When an agent is working in `src/payments/`, it can pull all relevant context with one command:

```bash
pearls context --for src/payments/checkout.ts
```

This returns every pearl whose globs match that path -- API docs, conventions, schema info -- without needing a file in that directory.

### Pull: Semantic Search

```bash
pearls search "how do we handle payment errors" --semantic --json
```

### JSON Output

All commands support `--json` for structured output:

```bash
pearls context --scope payments --json
```

### Workflow with Beads

Pearls complements [beads](https://github.com/steveyegge/beads) for agent work:

```bash
bd ready                                             # Get next task
pearls context --for src/payments/checkout.ts        # Get relevant context
pearls search "customer churn" --semantic             # Find more knowledge
# ... do work ...
bd close bd-a3f8                                      # Complete task
```

## Content Templates

When you create a pearl, a markdown template is generated based on the asset type. For a `table`:

```markdown
# users

Brief description here.

## Schema

| Column | Type | Nullable | Description |
|--------|------|----------|-------------|
| id     |      |          |             |

## Relationships

## Notes
```

Edit the generated file to document your data asset.

## Development

```bash
# Clone
git clone https://github.com/justrnr500/pearls.git
cd pearls

# Build
go build -o pearls ./cmd/pearls

# Test
go test ./...

# Install locally
go install ./cmd/pearls
```

### Project Structure

```
cmd/
├── pearls/           # Main entry point
└── pl/               # Short alias
internal/
├── cmd/              # CLI commands (Cobra)
├── config/           # Configuration management
├── embedding/        # Vector embeddings (hugot + all-MiniLM-L6-v2)
├── introspect/       # Database introspection (Postgres, MySQL, SQLite)
├── pearl/            # Core types and validation
└── storage/          # SQLite, JSONL, content files, vector index
```

## License

MIT
