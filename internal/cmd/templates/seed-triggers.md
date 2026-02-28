# Pearls Context Triggers

This pearl documents how pearls are automatically injected into agent sessions.

## Push-Based Injection (Globs)

Pearls with `globs` patterns are automatically surfaced when you work on matching files.
The `UserPromptSubmit` hook runs `pearls context --for <changed-files>` to find relevant pearls.

Example: A pearl with `globs: ["src/payments/**"]` will be injected when editing payment files.

## Session Start Injection (Clutch)

The `SessionStart` hook runs `pearls clutch` which outputs all **required** pearls
sorted by priority (highest first). These provide baseline context every session.

Mark pearls as required: `pearls update <id> --required --priority <n>`

## Scope-Based Injection

Use `pearls context --scope <name>` to pull pearls tagged with a specific scope.
Scopes group related pearls across different namespaces.
