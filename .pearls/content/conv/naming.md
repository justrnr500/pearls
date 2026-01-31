# Pearl ID Conventions

## Format

IDs are dot-separated namespace paths: \`<category>.<subcategory>.<name>\`

## Naming Patterns

| Category | Example | Use for |
|----------|---------|---------|
| \`db.<engine>.<table>\` | \`db.postgres.users\` | Database tables/schemas |
| \`api.<service>.<resource>\` | \`api.stripe.customers\` | External APIs |
| \`arch.<topic>\` | \`arch.storage\` | Architecture documentation |
| \`conv.<topic>\` | \`conv.testing\` | Codebase conventions |
| \`runbook.<topic>\` | \`runbook.deploy\` | Operational runbooks |

## Rules

- Lowercase alphanumeric only (no hyphens in namespace segments)
- Dots separate hierarchy levels
- Last segment is the name, everything before is namespace
- Namespace maps to filesystem: \`db.postgres.users\` → \`content/db/postgres/users.md\`
- Types are free-form strings (not restricted to the built-in list)