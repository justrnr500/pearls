# Config System

## Directory Structure

```
.pearls/
├── config.yaml     # Project settings (vector search, project name)
├── pearls.db       # SQLite database (gitignored)
├── pearls.jsonl    # Git-tracked source of truth
├── content/        # Markdown files organized by namespace
│   ├── db/postgres/users.md
│   └── api/stripe/customers.md
└── .gitignore      # Ignores pearls.db (SQLite is cache, JSONL is truth)
```

## Key Functions (`internal/config/config.go`)

- `FindRoot(cwd)` — walks up directory tree looking for `.pearls/`
- `ResolvePaths(root)` — returns `Paths` struct with absolute paths for DB, JSONL, content, config
- `Load(path)` / `Save(path)` — YAML config read/write

## Paths Struct

```go
type Paths struct {
    Root    string  // .pearls directory
    Config  string  // config.yaml
    DB      string  // pearls.db
    JSONL   string  // pearls.jsonl
    Content string  // content/
}
```

## Config YAML

```yaml
project:
  name: my-project
vector_search:
  enabled: true
  model_path: ""  # empty = default (~/.pearls/models)
```