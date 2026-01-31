# Adding CLI Commands

All commands live in `internal/cmd/` and use the Cobra framework.

## Command Template

```go
var fooCmd = &cobra.Command{
    Use:   "foo <id>",
    Short: "One-line description",
    Long:  `Detailed help with examples.`,
    Args:  cobra.ExactArgs(1),
    RunE:  runFoo,
}

func init() {
    rootCmd.AddCommand(fooCmd)
    fooCmd.Flags().StringVar(&fooFlag, "flag", "default", "description")
}

func runFoo(cmd *cobra.Command, args []string) error {
    store, _, err := getStore()
    if err != nil { return err }
    defer store.Close()
    // ...
}
```

## Key Patterns

- Use `RunE` (not `Run`) — returns error for consistent error handling
- Access store via `getStore()` from `store.go` — finds `.pearls/` root, opens all three layers
- Always `defer store.Close()`
- For JSON output: `--json` flag with `json.NewEncoder(os.Stdout).SetIndent("", "  ")`
- Flags: `StringVarP` for short flags (`-t`), `StringSliceVar` for repeatable (`--tag`)
- Templates: use `go:embed` with files in `internal/cmd/templates/` — keeps text out of Go code