# Testing Conventions

## Test Setup

Create an isolated store with `t.TempDir()`:

```go
func setupTestStore(t *testing.T) (*storage.Store, string) {
    t.Helper()
    tmpDir := t.TempDir()
    store, err := storage.NewStore(
        filepath.Join(tmpDir, "pearls.db"),
        filepath.Join(tmpDir, "pearls.jsonl"),
        filepath.Join(tmpDir, "content"),
    )
    if err != nil { t.Fatalf("new store: %v", err) }
    return store, tmpDir
}
```

## Creating Test Pearls

```go
now := time.Now()
p := &pearl.Pearl{
    ID: "test.foo", Name: "foo", Namespace: "test",
    Type: pearl.TypeTable, Status: pearl.StatusActive,
    CreatedAt: now, UpdatedAt: now,
}
store.Create(p, "# Foo")
```

## Assertions

- `t.Error` / `t.Errorf` for non-fatal (test continues)
- `t.Fatal` / `t.Fatalf` for fatal (test stops)
- No assertion library — standard Go testing only

## Test File Naming

- Tests co-located: `foo.go` → `foo_test.go` in same package
- Integration tests: `integration_test.go`