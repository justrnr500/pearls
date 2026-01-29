# Pearls

Data asset memory for AI agents.

Pearls is a CLI tool for storing and retrieving structured markdown documentation about data assets -- tables, schemas, database connections, file locations, APIs, and other data sources. It gives AI agents a queryable knowledge base about an organization's data landscape.

> Beads gives agents memory about *tasks*. Pearls gives agents memory about *data*.

## Features

- **Markdown-native** -- Pearl content is plain markdown. Human-readable, version-controllable, LLM-friendly.
- **Git as database** -- JSONL metadata + markdown content travel with your code. No server required.
- **Agent-first** -- Every command supports `--json` output. Search is fast and semantic.
- **Hierarchical** -- Dot-separated namespaces: `db.postgres.users`, `api.stripe.customers`
- **Semantic search** -- Natural language queries powered by local embeddings (all-MiniLM-L6-v2)
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

# Edit the generated markdown
$EDITOR .pearls/content/db/postgres/users.md

# Create more pearls and link them
pearls create db.postgres.orders --type table -d "Customer orders"
pearls update db.postgres.orders --add-ref db.postgres.users

# Or auto-generate from a live database
pearls introspect postgres --prefix db.postgres

# Search
pearls search "customer data"
pearls search "where is payment info stored" --semantic

# View relationships
pearls refs db.postgres.users

# Generate context for an agent prompt
pearls context db.postgres.users db.postgres.orders

# Check catalog health
pearls doctor

# Commit to git
git add .pearls/
git commit -m "Add data catalog"
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

Create a new pearl to document a data asset.

```bash
pearls create db.postgres.users --type table
pearls create api.stripe.customers --type api -d "Stripe customer records"
pearls create db.postgres.orders --type table --tag pii --tag core
pearls create warehouse.snowflake.metrics --type query --json
```

**Flags:**
- `--type, -t` -- Asset type: `table`, `schema`, `database`, `api`, `endpoint`, `file`, `bucket`, `pipeline`, `dashboard`, `query`, `custom` (default: `table`)
- `--description, -d` -- Brief description
- `--tag` -- Tags (repeatable)
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
pearls list --namespace db.postgres
pearls list --tag pii
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
pearls update db.postgres.users --remove-ref db.postgres.old_table
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

Generate concatenated markdown for AI agent prompts.

```bash
pearls context db.postgres.users db.postgres.orders
pearls context db.postgres.users --with-refs   # Include referenced pearls
pearls context db.postgres.users --brief       # Metadata only, no content
```

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

Inject agent instructions into project config files.

```bash
pearls onboard                    # Update CLAUDE.md (default)
pearls onboard --target agents    # Update agents.md
pearls onboard --target all       # Update both
pearls onboard --force            # Overwrite existing pearls section
```

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

### JSON Output

All commands support `--json` for structured output:

```bash
pearls search "customer data" --json
```

```json
{
  "query": "customer data",
  "results": [
    {
      "id": "db.postgres.customers",
      "type": "table",
      "status": "active",
      "description": "Customer account records"
    }
  ],
  "count": 1
}
```

### Context Injection

Generate markdown context for LLM prompts:

```bash
# Get full documentation for relevant assets
pearls context db.postgres.users db.postgres.orders

# Include all referenced assets
pearls context db.postgres.orders --with-refs
```

### Workflow with Beads

Pearls complements [beads](https://github.com/steveyegge/beads) for data work:

```bash
bd ready                                        # Get next task
pearls search "customer churn" --semantic --json # Find relevant data
pearls context warehouse.snowflake.user_cohorts  # Get full details
# ... do work ...
bd close bd-a3f8                                 # Complete task
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
