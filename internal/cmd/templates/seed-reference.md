# Pearls Quick Reference

## Commands

| Command | Description |
|---------|-------------|
| `pearls create <id> --type <type>` | Create a new pearl |
| `pearls list` | List all pearls |
| `pearls show <id>` | View pearl metadata |
| `pearls cat <id>` | View pearl content |
| `pearls update <id>` | Update pearl fields |
| `pearls delete <id>` | Delete a pearl |
| `pearls context --for <path>` | Get pearls matching file globs |
| `pearls context --scope <name>` | Get pearls by scope |
| `pearls search "query"` | Keyword search |
| `pearls search "query" --semantic` | Semantic search |
| `pearls clutch` | Output all required pearls |
| `pearls doctor` | Check catalog health |
| `pearls onboard` | Set up agent instructions |

## Pearl Types

Types are free-form (lowercase alphanumeric + hyphens). Common types:
`table`, `schema`, `database`, `api`, `endpoint`, `file`, `convention`, `brainstorm`, `runbook`, `custom`

## Key Flags

- `--required` / `--priority <n>`: Mark pearl as required context (included in clutch output)
- `--globs "pattern"`: File patterns for push-based injection
- `--scopes "name"`: Scope groupings for scope-based injection
- `--tag <tag>`: Freeform tags (repeatable)
