# Storage Architecture

Pearls uses a three-layer storage system. All three are kept in sync by the `Store` orchestrator in `internal/storage/sync.go`.

## Layers

### 1. SQLite (`internal/storage/sqlite.go`)
- Fast queries, indexes, vector search (384-dim via sqlite-vec)
- Tables: `pearls` (metadata), `pearl_embeddings` (vectors)
- Key methods: `Insert`, `Update`, `Delete`, `Get`, `List`, `Search`, `FindByScope`, `FindByGlob`

### 2. JSONL (`internal/storage/jsonl.go`)
- **Source of truth** for git portability — one JSON object per line
- `SyncFromJSONL()` rebuilds SQLite from JSONL
- Writes are atomic: temp file → fsync → rename

### 3. Content files (`internal/storage/content.go`)
- Markdown files organized by namespace: `db.postgres.users` → `content/db/postgres/users.md`
- Namespace dots become directory separators, name becomes filename

## Coordination Pattern
- **Create**: content file → DB insert → embedding → JSONL append
- **Update**: content file → DB update → embedding update → full JSONL rebuild
- **Delete**: get pearl → delete embedding → DB delete → content delete → JSONL rebuild

## Embeddings (`internal/storage/vector.go`)
- Optional — Store works without embedder
- Uses all-MiniLM-L6-v2 (384 dimensions, L2 distance)
- sqlite-vec k-NN search via CTE for JOIN compatibility
- Auto-generated on Create/Update if embedder is configured