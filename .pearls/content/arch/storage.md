# Storage Architecture

Pearls uses a three-layer storage system. All three are kept in sync by the `Store` orchestrator in `internal/storage/sync.go`.

## Layers

### 1. SQLite (`internal/storage/sqlite.go`)
- Fast queries, indexes
- Tables: `pearls` (metadata)
- Key methods: `Insert`, `Update`, `Delete`, `Get`, `List`, `Search`, `FindByScope`, `FindByGlob`

### 2. JSONL (`internal/storage/jsonl.go`)
- **Source of truth** for git portability — one JSON object per line
- `SyncFromJSONL()` rebuilds SQLite from JSONL
- Writes are atomic: temp file → fsync → rename

### 3. Content files (`internal/storage/content.go`)
- Markdown files organized by namespace: `db.postgres.users` → `content/db/postgres/users.md`
- Namespace dots become directory separators, name becomes filename

## Coordination Pattern
- **Create**: content file → DB insert → JSONL append
- **Update**: content file → DB update → full JSONL rebuild
- **Delete**: get pearl → DB delete → content delete → JSONL rebuild