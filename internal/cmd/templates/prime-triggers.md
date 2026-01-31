## When to Create Pearls

- **After figuring out a non-obvious system** (auth flow, data pipeline, deployment) → `pl create <id> --type architecture`
- **After discovering an undocumented convention** (naming, error handling, test patterns) → `pl create <id> --type convention --globs "<paths>"`
- **After a debugging session with a surprising root cause** → `pl create <id> --type runbook`
- **After a design discussion or brainstorm** → `pl create <id> --type brainstorm --content -`
- **After encountering undocumented API behavior** → `pl create <id> --type api`
- **After running `pl introspect`** → create/update pearls for new or changed schemas
