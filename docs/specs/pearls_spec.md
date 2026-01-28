# Pearls: Data Asset Memory for AI Agents

## Executive Summary

**Pearls** is a CLI tool for storing and retrieving structured markdown documentation about data assets—tables, schemas, database connections, file locations, APIs, and other data sources. Inspired by [beads](https://github.com/steveyegge/beads) (a git-backed issue tracker for coding agents) and [moltbot's memory system](https://docs.clawd.bot/concepts/memory) (markdown-based memory with vector search), Pearls provides agents with a queryable knowledge base about an organization's data landscape.

### Core Philosophy

> "Beads gives agents memory about *tasks*. Pearls gives agents memory about *data*."

While beads tracks work items and dependencies, Pearls tracks data assets and their relationships—where data lives, what it contains, how it connects, and how to use it.

---

## Problem Statement

AI agents working with data frequently need to answer questions like:

- "Where is the customer data stored?"
- "What's the schema for the orders table?"
- "How do I connect to the analytics database?"
- "Which tables contain PII?"
- "What's the relationship between `users` and `accounts`?"

Without a structured knowledge base, agents either:
1. Ask the user repeatedly (poor UX)
2. Hallucinate (dangerous)
3. Fail to help (frustrating)

Pearls solves this by providing a **local-first, git-backed, agent-optimized** data catalog.

---

## Design Principles

### 1. **Markdown-Native**
Pearl content is plain markdown—human-readable, version-controllable, and LLM-friendly. No proprietary formats.

### 2. **Git as Database**
Like beads, Pearls uses git for distribution. JSONL metadata + markdown content travel with your code. No server required.

### 3. **Agent-First**
Every command supports `--json` output. Pearl IDs are short and memorable. Search is fast and semantic.

### 4. **Hierarchical Organization**
Pearls support namespacing: `db.postgres.users`, `api.stripe.customers`, `warehouse.snowflake.analytics.orders`

### 5. **Relationship Tracking**
Pearls can reference other pearls, creating a navigable graph of data assets.

---

## Data Model

### Pearl

A **pearl** is a documented data asset with metadata and markdown content.

```go
type Pearl struct {
    // Identity
    ID          string    `json:"id"`          // e.g., "prl-a3f8" or "db.postgres.users"
    Name        string    `json:"name"`        // Human-readable name
    Namespace   string    `json:"namespace"`   // Dot-separated path: "db.postgres"
    
    // Classification
    Type        AssetType `json:"type"`        // table, schema, database, api, file, bucket, etc.
    Tags        []string  `json:"tags"`        // Freeform tags: ["pii", "analytics", "deprecated"]
    
    // Content
    Description string    `json:"description"` // Brief one-liner
    ContentPath string    `json:"content_path"`// Path to markdown file
    ContentHash string    `json:"content_hash"`// SHA256 of content for change detection
    
    // Relationships
    References  []string  `json:"references"`  // IDs of related pearls
    Parent      string    `json:"parent"`      // Parent pearl ID (for hierarchical assets)
    
    // Connection (optional, for databases/APIs)
    Connection  *ConnectionInfo `json:"connection,omitempty"`
    
    // Metadata
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
    CreatedBy   string    `json:"created_by"`
    Status      Status    `json:"status"`      // active, deprecated, archived
}

type AssetType string
const (
    TypeTable     AssetType = "table"
    TypeSchema    AssetType = "schema"
    TypeDatabase  AssetType = "database"
    TypeAPI       AssetType = "api"
    TypeEndpoint  AssetType = "endpoint"
    TypeFile      AssetType = "file"
    TypeBucket    AssetType = "bucket"
    TypePipeline  AssetType = "pipeline"
    TypeDashboard AssetType = "dashboard"
    TypeQuery     AssetType = "query"
    TypeCustom    AssetType = "custom"
)

type ConnectionInfo struct {
    Type     string            `json:"type"`     // postgres, mysql, snowflake, bigquery, s3, etc.
    Host     string            `json:"host"`     // Can be env var reference: ${DB_HOST}
    Port     int               `json:"port"`
    Database string            `json:"database"`
    Schema   string            `json:"schema"`
    Extras   map[string]string `json:"extras"`   // Additional connection params
}

type Status string
const (
    StatusActive     Status = "active"
    StatusDeprecated Status = "deprecated"
    StatusArchived   Status = "archived"
)
```

### Markdown Content Structure

Each pearl has an associated markdown file with structured sections:

```markdown
# users

PostgreSQL table containing core user account information.

## Schema

| Column | Type | Nullable | Description |
|--------|------|----------|-------------|
| id | uuid | NO | Primary key |
| email | varchar(255) | NO | User email, unique |
| created_at | timestamptz | NO | Account creation time |
| org_id | uuid | YES | FK to organizations.id |

## Relationships

- **organizations** (via `org_id`): User's parent organization
- **orders** (via `orders.user_id`): User's purchase history

## Access Patterns

```sql
-- Get user by email
SELECT * FROM users WHERE email = $1;

-- Get users with recent orders
SELECT u.* FROM users u
JOIN orders o ON o.user_id = u.id
WHERE o.created_at > NOW() - INTERVAL '30 days';
```

## Notes

- PII: Contains email addresses
- Indexes: email (unique), org_id, created_at
- Row estimate: ~2.5M rows
- Partitioned: No

## History

- 2024-01: Added `org_id` column for multi-tenant support
- 2023-06: Created table
```

---

## Directory Structure

```
.pearls/
├── pearls.db           # SQLite cache (gitignored)
├── pearls.jsonl        # Metadata source of truth (git-tracked)
├── config.yaml         # Configuration
├── content/            # Markdown content (git-tracked)
│   ├── db/
│   │   └── postgres/
│   │       ├── _schema.md      # Database-level docs
│   │       ├── users.md
│   │       ├── orders.md
│   │       └── organizations.md
│   ├── api/
│   │   └── stripe/
│   │       ├── _api.md
│   │       ├── customers.md
│   │       └── payments.md
│   └── warehouse/
│       └── snowflake/
│           └── analytics/
│               ├── daily_metrics.md
│               └── user_cohorts.md
└── .gitignore
```

---

## CLI Commands

### Initialization

```bash
# Initialize pearls in current directory
pearls init

# Initialize with vector search enabled
pearls init --vector-search

# Initialize in quiet mode (for agents)
pearls init --quiet
```

### Creating Pearls

```bash
# Create a new pearl interactively
pearls create db.postgres.users --type table

# Create with inline description
pearls create db.postgres.orders --type table -d "Customer order records"

# Create from existing SQL schema
pearls create db.postgres.products --type table --from-ddl schema.sql

# Create from database introspection
pearls create db.postgres --type database --introspect "postgres://user:pass@host/db"

# Import multiple pearls from a catalog file
pearls import catalog.yaml

# JSON output for agents
pearls create api.stripe.customers --type api --json
```

### Viewing Pearls

```bash
# List all pearls
pearls list
pearls list --type table
pearls list --tag pii
pearls list --namespace db.postgres
pearls list --status active

# Show pearl details
pearls show db.postgres.users
pearls show db.postgres.users --json

# Show pearl content (markdown)
pearls cat db.postgres.users

# Show pearl with related assets
pearls show db.postgres.users --with-refs

# Tree view of namespace
pearls tree db.postgres
```

### Searching

```bash
# Keyword search
pearls search "customer email"

# Search by type
pearls search "orders" --type table

# Search with filters
pearls search "analytics" --tag dashboard --status active

# Semantic search (requires vector index)
pearls search "where is user payment information stored" --semantic

# Find pearls referencing a specific pearl
pearls refs db.postgres.users

# JSON output for agents
pearls search "customer data" --json
```

### Updating Pearls

```bash
# Edit pearl content (opens $EDITOR)
pearls edit db.postgres.users

# Update metadata
pearls update db.postgres.users --add-tag pii
pearls update db.postgres.users --status deprecated
pearls update db.postgres.users --add-ref db.postgres.organizations

# Rename/move pearl
pearls mv db.postgres.users db.postgres.accounts

# Bulk update
pearls update --namespace db.postgres --add-tag "needs-review"
```

### Deleting Pearls

```bash
# Delete pearl (moves to archived)
pearls archive db.postgres.old_table

# Permanently delete
pearls delete db.postgres.old_table --force

# Delete namespace
pearls delete db.legacy --recursive --force
```

### Syncing

```bash
# Sync local DB with JSONL (automatic, but can be manual)
pearls sync

# Export to JSONL
pearls export -o pearls.jsonl

# Import from JSONL
pearls import -i pearls.jsonl

# Rebuild vector index
pearls index --rebuild
```

### Health & Diagnostics

```bash
# Check installation health
pearls doctor

# Show database info
pearls info
pearls info --json

# Validate all pearl content
pearls validate

# Find orphaned content files
pearls gc --dry-run
```

---

## Agent-Optimized Features

### 1. Ready Query

Get the most relevant pearls for a given context:

```bash
# Get pearls most likely relevant to current work
pearls relevant "building a user registration API"

# Output (JSON for agents):
{
  "pearls": [
    {"id": "db.postgres.users", "relevance": 0.92, "reason": "User table schema"},
    {"id": "api.auth.register", "relevance": 0.88, "reason": "Registration endpoint"},
    {"id": "db.postgres.organizations", "relevance": 0.71, "reason": "Referenced by users"}
  ]
}
```

### 2. Context Injection

Generate a context block for agent prompts:

```bash
# Generate context for specific pearls
pearls context db.postgres.users db.postgres.orders

# Output: Concatenated markdown suitable for LLM context
```

### 3. Schema Extraction

Quick schema lookup optimized for code generation:

```bash
# Get just the schema (no prose)
pearls schema db.postgres.users --format sql
pearls schema db.postgres.users --format typescript
pearls schema db.postgres.users --format json-schema
```

### 4. Connection Info

Retrieve connection details (with env var resolution):

```bash
# Get connection string
pearls connect db.postgres --format url
# Output: postgres://user:****@host:5432/mydb

pearls connect db.postgres --format env
# Output:
# export PGHOST=host
# export PGPORT=5432
# export PGDATABASE=mydb
```

---

## Configuration

### config.yaml

```yaml
# .pearls/config.yaml

# Project identification
project:
  name: "my-company-data"
  description: "Data asset catalog for My Company"

# Storage settings
storage:
  content_dir: "content"  # Relative to .pearls/
  
# Vector search (optional)
vector_search:
  enabled: true
  provider: "local"  # local, openai, gemini
  model: "all-MiniLM-L6-v2"  # For local embeddings
  # For remote embeddings:
  # provider: "openai"
  # model: "text-embedding-3-small"
  # api_key: "${OPENAI_API_KEY}"

# Default metadata for new pearls
defaults:
  status: "active"
  created_by: "${USER}"

# Namespace aliases
aliases:
  pg: "db.postgres"
  sf: "warehouse.snowflake"
  
# Introspection settings (for --introspect)
introspection:
  postgres:
    exclude_schemas: ["pg_catalog", "information_schema"]
    exclude_tables: ["_migrations", "_seeds"]
```

---

## Integration Patterns

### 1. With Beads

Pearls complements beads for data engineering work:

```bash
# In your AGENTS.md:
# "Use `bd` for task tracking. Use `pearls` for data asset lookup."

# Agent workflow:
bd ready --json                           # Get next task
pearls search "customer churn" --json     # Find relevant data
pearls context warehouse.snowflake.user_cohorts  # Get full details
# ... do work ...
bd close bd-a3f8                          # Complete task
```

### 2. With Claude Code / MCP

Pearls can be exposed as an MCP server:

```bash
# Start MCP server
pearls mcp serve

# Claude Desktop config:
{
  "mcpServers": {
    "pearls": {
      "command": "pearls",
      "args": ["mcp", "serve"]
    }
  }
}
```

### 3. CI/CD Integration

```yaml
# .github/workflows/pearls.yml
name: Validate Data Catalog
on: [push]
jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install pearls
        run: go install github.com/yourorg/pearls@latest
      - name: Validate catalog
        run: pearls validate --strict
```

---

## Implementation Plan

### Phase 1: Core CLI (MVP)

1. **Storage Layer**
   - SQLite backend with JSONL sync (port from beads)
   - Markdown content file management
   - Basic CRUD operations

2. **Core Commands**
   - `init`, `create`, `show`, `list`, `edit`, `delete`
   - `search` (keyword-based)
   - `sync`, `export`, `import`

3. **Agent Support**
   - `--json` flag on all commands
   - `context` command for prompt injection

### Phase 2: Enhanced Search & Relationships

1. **Vector Search**
   - Local embeddings with sqlite-vec
   - Optional remote embeddings (OpenAI, Gemini)
   - Hybrid BM25 + vector search

2. **Relationship Graph**
   - Reference tracking
   - `refs` and `tree` commands
   - Cycle detection

### Phase 3: Integrations

1. **Database Introspection**
   - PostgreSQL, MySQL, SQLite
   - Snowflake, BigQuery, Redshift

2. **MCP Server**
   - Tool definitions for Claude
   - Resource exposure

3. **Import/Export**
   - dbt manifest import
   - DataHub/Amundsen export
   - OpenAPI spec import

---

## Technical Architecture

```
┌────────────────────────────────────────────────────────────────────┐
│                           CLI (Cobra)                              │
│  pearls create | show | list | search | edit | sync | ...          │
└────────────────────────────────┬───────────────────────────────────┘
                                 │
                                 ▼
┌────────────────────────────────────────────────────────────────────┐
│                        Core Library                                │
│  - Pearl CRUD operations                                           │
│  - Content file management                                         │
│  - Namespace resolution                                            │
│  - Reference graph                                                 │
└────────────────────────────────┬───────────────────────────────────┘
                                 │
              ┌──────────────────┼──────────────────┐
              ▼                  ▼                  ▼
┌─────────────────────┐ ┌───────────────┐ ┌───────────────────────┐
│   SQLite Storage    │ │ JSONL Sync    │ │   Vector Index        │
│   (.pearls/db)      │ │ (pearls.jsonl)│ │   (sqlite-vec)        │
│                     │ │               │ │                       │
│ - Fast queries      │ │ - Git-tracked │ │ - Semantic search     │
│ - Full-text search  │ │ - Source of   │ │ - Similarity lookup   │
│ - Relationships     │ │   truth       │ │ - Optional            │
└─────────────────────┘ └───────────────┘ └───────────────────────┘
              │                  │
              └──────────────────┘
                       │
                       ▼
┌────────────────────────────────────────────────────────────────────┐
│                     Content Files                                  │
│                   (.pearls/content/**/*.md)                        │
│                                                                    │
│  - Human-readable markdown                                         │
│  - Schema definitions, examples, notes                             │
│  - Git-tracked alongside metadata                                  │
└────────────────────────────────────────────────────────────────────┘
```

---

## Example Session

```bash
$ pearls init
✓ Created .pearls/ directory
✓ Initialized SQLite database
✓ Ready to track data assets

$ pearls create db.postgres.users --type table
Created pearl: db.postgres.users
Content file: .pearls/content/db/postgres/users.md

# Edit the generated markdown...
$ pearls edit db.postgres.users

$ pearls create db.postgres.orders --type table -d "Customer orders"
$ pearls update db.postgres.orders --add-ref db.postgres.users

$ pearls list
NAMESPACE          TYPE    STATUS  DESCRIPTION
db.postgres.users  table   active  Core user account information
db.postgres.orders table   active  Customer orders

$ pearls search "customer purchase history" --json
{
  "results": [
    {
      "id": "db.postgres.orders",
      "score": 0.87,
      "snippet": "Customer order records with purchase history..."
    }
  ]
}

$ pearls context db.postgres.users db.postgres.orders
# users

PostgreSQL table containing core user account information.

## Schema
| Column | Type | Nullable | Description |
...

---

# orders

Customer order records.

## Schema
...

$ git add .pearls/
$ git commit -m "Add initial data catalog"
```

---

## Comparison to Alternatives

| Feature | Pearls | DataHub | dbt docs | Custom wiki |
|---------|--------|---------|----------|-------------|
| Local-first | ✅ | ❌ | ✅ | ❌ |
| Git-native | ✅ | ❌ | ✅ | ❌ |
| Agent-optimized | ✅ | ❌ | ❌ | ❌ |
| Zero infrastructure | ✅ | ❌ | ✅ | ❌ |
| Semantic search | ✅ | ✅ | ❌ | ❌ |
| Relationship graph | ✅ | ✅ | ✅ | ❌ |
| Offline capable | ✅ | ❌ | ✅ | ❌ |

---

## Design Decisions

1. **ID Format**: Namespace paths (`db.postgres.users`) as primary identifiers
   - Human-readable and self-documenting
   - Natural hierarchy matches data organization
   - No collision concerns since paths are explicit

2. **Content Storage**: Separate markdown files in `content/` directory
   - Enables meaningful diffs and PR reviews
   - Works with existing editors and tooling
   - Content changes are visible in git history

3. **Vector Index**: sqlite-vec extension in main SQLite database
   - Single database file for simplicity
   - No additional dependencies to manage
   - Proven approach from moltbot

4. **Schema Format**: Suggested sections via templates, freeform content allowed
   - Templates provide structure for common patterns
   - Markdown flexibility beats rigid TOML/YAML for documentation
   - Agents can parse suggested sections; humans can add whatever they need

---

## Next Steps

1. Review and refine this spec
2. Set up Go module structure
3. Implement Phase 1 MVP
4. Dogfood with a real project
5. Iterate based on agent usage patterns

---

## References

- [beads](https://github.com/steveyegge/beads) - Git-backed issue tracker for coding agents
- [moltbot memory](https://docs.clawd.bot/concepts/memory) - Markdown + vector memory system
- [dbt](https://docs.getdbt.com/) - Data transformation with docs
- [DataHub](https://datahubproject.io/) - Enterprise data catalog
