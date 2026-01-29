# Phase 3: Agent Integration & Database Introspection — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `pl` shortcut binary, `pearls onboard` for agent config injection, `pearls doctor` for catalog health checks, and `pearls introspect` for auto-generating pearls from live databases.

**Architecture:** Four independent features layered on top of the existing Cobra command structure. Each feature is a new command file in `internal/cmd/` following the established pattern (package-level var + `init()` registration + `RunE` handler). The introspect feature additionally introduces a new `internal/introspect/` package with a driver interface and per-database implementations.

**Tech Stack:** Go 1.24, Cobra CLI, SQLite (existing), `database/sql` + `github.com/lib/pq` (Postgres), `github.com/go-sql-driver/mysql` (MySQL), `github.com/mattn/go-sqlite3` (existing), `github.com/joho/godotenv` (.env loading)

---

## Task 1: `pl` Shortcut Binary

Creates a second entry point binary that is fully equivalent to `pearls`.

**Files:**
- Create: `cmd/pl/main.go`

**Step 1: Create the `pl` entry point**

Create `cmd/pl/main.go`:

```go
// Package main is the entry point for the pl CLI (alias for pearls).
package main

import (
	"github.com/justrnr500/pearls/internal/cmd"
)

func main() {
	cmd.Execute()
}
```

**Step 2: Verify both binaries build**

Run:
```bash
go build ./cmd/pearls && go build ./cmd/pl
```
Expected: Both compile without errors.

**Step 3: Test the `pl` binary**

Run:
```bash
go run ./cmd/pl --version
go run ./cmd/pl --help
```
Expected: Same output as `pearls --version` and `pearls --help`.

**Step 4: Commit**

```bash
git add cmd/pl/main.go
git commit -m "feat: add pl shortcut binary"
```

---

## Task 2: `pearls onboard` Command

Generates agent-facing instructions and appends them to project config files (CLAUDE.md, agents.md).

**Files:**
- Create: `internal/cmd/onboard.go`
- Test: `internal/cmd/onboard_test.go`

**Step 1: Write the failing test**

Create `internal/cmd/onboard_test.go`:

```go
package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOnboardTemplate(t *testing.T) {
	content := onboardTemplate()
	if !strings.Contains(content, "<!-- pearls:start -->") {
		t.Error("template missing start marker")
	}
	if !strings.Contains(content, "<!-- pearls:end -->") {
		t.Error("template missing end marker")
	}
	if !strings.Contains(content, "pl list") {
		t.Error("template missing pl list command")
	}
}

func TestOnboardNewFile(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "CLAUDE.md")

	err := onboardToFile(target, false)
	if err != nil {
		t.Fatalf("onboard to new file: %v", err)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "<!-- pearls:start -->") {
		t.Error("file missing start marker")
	}
	if !strings.Contains(content, "<!-- pearls:end -->") {
		t.Error("file missing end marker")
	}
}

func TestOnboardAppendToExisting(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "CLAUDE.md")

	// Write pre-existing content
	os.WriteFile(target, []byte("# My Project\n\nExisting content.\n"), 0644)

	err := onboardToFile(target, false)
	if err != nil {
		t.Fatalf("onboard: %v", err)
	}

	data, _ := os.ReadFile(target)
	content := string(data)

	if !strings.HasPrefix(content, "# My Project") {
		t.Error("existing content should be preserved at top")
	}
	if !strings.Contains(content, "<!-- pearls:start -->") {
		t.Error("pearls block should be appended")
	}
}

func TestOnboardSkipIfAlreadyOnboarded(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "CLAUDE.md")

	// First onboard
	onboardToFile(target, false)
	data1, _ := os.ReadFile(target)

	// Second onboard without force — should not change
	err := onboardToFile(target, false)
	if err != nil {
		t.Fatalf("second onboard: %v", err)
	}

	data2, _ := os.ReadFile(target)
	if string(data1) != string(data2) {
		t.Error("file should not change on second onboard without --force")
	}
}

func TestOnboardForceReplaces(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "CLAUDE.md")

	// Pre-populate with old onboard content
	old := "# Existing\n\n<!-- pearls:start -->\nOLD CONTENT\n<!-- pearls:end -->\n\nAfter.\n"
	os.WriteFile(target, []byte(old), 0644)

	err := onboardToFile(target, true)
	if err != nil {
		t.Fatalf("force onboard: %v", err)
	}

	data, _ := os.ReadFile(target)
	content := string(data)

	if strings.Contains(content, "OLD CONTENT") {
		t.Error("old content should be replaced")
	}
	if !strings.Contains(content, "pl list") {
		t.Error("new template should be injected")
	}
	if !strings.Contains(content, "# Existing") {
		t.Error("content before markers should be preserved")
	}
	if !strings.Contains(content, "After.") {
		t.Error("content after markers should be preserved")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/cmd/ -run TestOnboard -v`
Expected: FAIL — functions not defined.

**Step 3: Implement onboard command**

Create `internal/cmd/onboard.go`:

```go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var onboardCmd = &cobra.Command{
	Use:   "onboard",
	Short: "Inject agent instructions into project config files",
	Long: `Generate agent-facing instructions for using pearls and append them
to CLAUDE.md, agents.md, or both.

Examples:
  pearls onboard                    # Update CLAUDE.md (default)
  pearls onboard --target agents    # Update agents.md
  pearls onboard --target all       # Update both
  pearls onboard --force            # Overwrite existing pearls section`,
	RunE: runOnboard,
}

var (
	onboardTarget string
	onboardForce  bool
)

func init() {
	rootCmd.AddCommand(onboardCmd)
	onboardCmd.Flags().StringVar(&onboardTarget, "target", "claude", "Which file to update: claude, agents, or all")
	onboardCmd.Flags().BoolVar(&onboardForce, "force", false, "Overwrite existing pearls section")
}

func runOnboard(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	targets := map[string]string{
		"claude": "CLAUDE.md",
		"agents": "agents.md",
	}

	var files []string
	switch onboardTarget {
	case "claude":
		files = []string{targets["claude"]}
	case "agents":
		files = []string{targets["agents"]}
	case "all":
		files = []string{targets["claude"], targets["agents"]}
	default:
		return fmt.Errorf("invalid target %q: must be claude, agents, or all", onboardTarget)
	}

	for _, name := range files {
		path := filepath.Join(cwd, name)
		if err := onboardToFile(path, onboardForce); err != nil {
			return fmt.Errorf("onboard %s: %w", name, err)
		}
		fmt.Printf("✓ Updated %s\n", name)
	}

	return nil
}

const (
	markerStart = "<!-- pearls:start -->"
	markerEnd   = "<!-- pearls:end -->"
)

func onboardTemplate() string {
	return `<!-- pearls:start -->
## Pearls - Data Asset Memory

This project uses Pearls to document data assets (tables, schemas, APIs, pipelines, etc.).

### Quick Reference
- ` + "`pl list`" + ` — List all documented data assets
- ` + "`pl search \"query\"`" + ` — Keyword search
- ` + "`pl search \"query\" --semantic`" + ` — Natural language search
- ` + "`pl show <id>`" + ` — View asset metadata
- ` + "`pl cat <id>`" + ` — View full markdown documentation
- ` + "`pl context <ids...>`" + ` — Get concatenated docs for your context window
- ` + "`pl create <id> --type <type>`" + ` — Document a new asset
- ` + "`pl refs <id>`" + ` — See relationships
- ` + "`pl introspect <type> --prefix <ns>`" + ` — Auto-discover from database
- ` + "`pl doctor`" + ` — Check catalog health

### When to Use Pearls
- Before querying a database, check ` + "`pl search`" + ` for schema documentation
- When encountering unfamiliar data assets, check ` + "`pl show`" + `
- After discovering new data sources, create a pearl with ` + "`pl create`" + `
- When setting up a new database connection, run ` + "`pl introspect`" + ` to bootstrap docs
<!-- pearls:end -->`
}

func onboardToFile(path string, force bool) error {
	// Read existing content (may not exist)
	existing := ""
	if data, err := os.ReadFile(path); err == nil {
		existing = string(data)
	}

	startIdx := strings.Index(existing, markerStart)
	endIdx := strings.Index(existing, markerEnd)
	hasMarkers := startIdx >= 0 && endIdx >= 0 && endIdx > startIdx

	template := onboardTemplate()

	if hasMarkers && !force {
		// Already onboarded, skip
		return nil
	}

	var result string
	if hasMarkers && force {
		// Replace between markers (inclusive)
		before := existing[:startIdx]
		after := existing[endIdx+len(markerEnd):]
		result = before + template + after
	} else {
		// Append to end
		if existing != "" && !strings.HasSuffix(existing, "\n") {
			existing += "\n"
		}
		if existing != "" {
			existing += "\n"
		}
		result = existing + template + "\n"
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	return os.WriteFile(path, []byte(result), 0644)
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/cmd/ -run TestOnboard -v`
Expected: All 5 tests PASS.

**Step 5: Commit**

```bash
git add internal/cmd/onboard.go internal/cmd/onboard_test.go
git commit -m "feat: add pearls onboard command"
```

---

## Task 3: `pearls doctor` Command

Health checks to diagnose common catalog issues.

**Files:**
- Create: `internal/cmd/doctor.go`
- Test: `internal/cmd/doctor_test.go`

**Step 1: Write the failing test**

Create `internal/cmd/doctor_test.go`:

```go
package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/justrnr500/pearls/internal/pearl"
	"github.com/justrnr500/pearls/internal/storage"
)

func setupDoctorTestStore(t *testing.T) (*storage.Store, string) {
	t.Helper()
	tmpDir := t.TempDir()

	store, err := storage.NewStore(
		filepath.Join(tmpDir, "pearls.db"),
		filepath.Join(tmpDir, "pearls.jsonl"),
		filepath.Join(tmpDir, "content"),
	)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	return store, tmpDir
}

func TestCheckJSONLSync_AllGood(t *testing.T) {
	store, _ := setupDoctorTestStore(t)
	defer store.Close()

	now := time.Now()
	p := &pearl.Pearl{
		ID: "test.a", Name: "a", Namespace: "test",
		Type: pearl.TypeTable, Status: pearl.StatusActive,
		CreatedAt: now, UpdatedAt: now,
	}
	store.Create(p, "# A")

	result := checkJSONLSync(store)
	if !result.Passed {
		t.Errorf("expected pass, got issues: %v", result.Issues)
	}
}

func TestCheckOrphanedContent(t *testing.T) {
	store, tmpDir := setupDoctorTestStore(t)
	defer store.Close()

	// Create an orphan markdown file
	contentDir := filepath.Join(tmpDir, "content", "orphan")
	os.MkdirAll(contentDir, 0755)
	os.WriteFile(filepath.Join(contentDir, "stale.md"), []byte("# Orphan"), 0644)

	result := checkOrphanedContent(store)
	if result.Passed {
		t.Error("expected fail due to orphaned content")
	}
	if len(result.Issues) != 1 {
		t.Errorf("expected 1 orphan, got %d", len(result.Issues))
	}
}

func TestCheckMissingContent(t *testing.T) {
	store, _ := setupDoctorTestStore(t)
	defer store.Close()

	now := time.Now()
	p := &pearl.Pearl{
		ID: "test.missing", Name: "missing", Namespace: "test",
		Type: pearl.TypeTable, Status: pearl.StatusActive,
		ContentPath: "test/missing.md",
		CreatedAt: now, UpdatedAt: now,
	}
	// Insert directly to DB to bypass content file creation
	store.DB().Insert(p)
	store.JSONL().Append(p)

	result := checkMissingContent(store)
	if result.Passed {
		t.Error("expected fail due to missing content")
	}
}

func TestCheckBrokenReferences(t *testing.T) {
	store, _ := setupDoctorTestStore(t)
	defer store.Close()

	now := time.Now()
	p := &pearl.Pearl{
		ID: "test.ref", Name: "ref", Namespace: "test",
		Type: pearl.TypeTable, Status: pearl.StatusActive,
		References: []string{"nonexistent.pearl"},
		CreatedAt: now, UpdatedAt: now,
	}
	store.Create(p, "# Ref")

	result := checkBrokenReferences(store)
	if result.Passed {
		t.Error("expected fail due to broken reference")
	}
}

func TestCheckConfigValidity(t *testing.T) {
	tmpDir := t.TempDir()

	// Write a valid config
	configPath := filepath.Join(tmpDir, "config.yaml")
	os.WriteFile(configPath, []byte("project:\n  name: test\n"), 0644)

	result := checkConfigValidity(configPath)
	if !result.Passed {
		t.Errorf("expected pass, got issues: %v", result.Issues)
	}
}

func TestCheckConfigValidity_BadYAML(t *testing.T) {
	tmpDir := t.TempDir()

	configPath := filepath.Join(tmpDir, "config.yaml")
	os.WriteFile(configPath, []byte("{{invalid yaml"), 0644)

	result := checkConfigValidity(configPath)
	if result.Passed {
		t.Error("expected fail for invalid YAML")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/cmd/ -run TestCheck -v`
Expected: FAIL — functions not defined.

**Step 3: Implement doctor command**

Create `internal/cmd/doctor.go`:

```go
package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/justrnr500/pearls/internal/config"
	"github.com/justrnr500/pearls/internal/storage"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check catalog health",
	Long: `Run health checks on the pearls catalog to diagnose common issues.

Checks:
  - JSONL/SQLite sync (pearl counts and IDs match)
  - Orphaned content (markdown files with no pearl)
  - Missing content (pearls with content_path that doesn't exist)
  - Broken references (pearls referencing IDs that don't exist)
  - Config validity (config.yaml parses without errors)`,
	RunE: runDoctor,
}

var doctorJSON bool

func init() {
	rootCmd.AddCommand(doctorCmd)
	doctorCmd.Flags().BoolVar(&doctorJSON, "json", false, "Output as JSON")
}

// CheckResult represents the result of a single health check.
type CheckResult struct {
	Name   string   `json:"name"`
	Passed bool     `json:"passed"`
	Issues []string `json:"issues,omitempty"`
}

func runDoctor(cmd *cobra.Command, args []string) error {
	store, paths, err := getStore()
	if err != nil {
		return err
	}
	defer store.Close()

	checks := []CheckResult{
		checkJSONLSync(store),
		checkOrphanedContent(store),
		checkMissingContent(store),
		checkBrokenReferences(store),
		checkConfigValidity(paths.Config),
	}

	if doctorJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(checks)
	}

	allPassed := true
	for _, c := range checks {
		if c.Passed {
			fmt.Printf("✓ %s\n", c.Name)
		} else {
			allPassed = false
			fmt.Printf("✗ %s\n", c.Name)
			for _, issue := range c.Issues {
				fmt.Printf("    %s\n", issue)
			}
		}
	}

	if !allPassed {
		return fmt.Errorf("some checks failed")
	}

	return nil
}

func checkJSONLSync(store *storage.Store) CheckResult {
	name := "JSONL/SQLite in sync"

	jsonlPearls, err := store.JSONL().ReadAll()
	if err != nil {
		return CheckResult{Name: name, Passed: false, Issues: []string{fmt.Sprintf("read JSONL: %v", err)}}
	}

	dbCount, err := store.DB().Count()
	if err != nil {
		return CheckResult{Name: name, Passed: false, Issues: []string{fmt.Sprintf("count DB: %v", err)}}
	}

	if len(jsonlPearls) != dbCount {
		return CheckResult{
			Name:   name,
			Passed: false,
			Issues: []string{fmt.Sprintf("count mismatch: JSONL=%d, SQLite=%d", len(jsonlPearls), dbCount)},
		}
	}

	// Check IDs match
	jsonlIDs := make(map[string]bool)
	for _, p := range jsonlPearls {
		jsonlIDs[p.ID] = true
	}

	dbPearls, err := store.DB().All()
	if err != nil {
		return CheckResult{Name: name, Passed: false, Issues: []string{fmt.Sprintf("list DB: %v", err)}}
	}

	var missingInJSONL []string
	for _, p := range dbPearls {
		if !jsonlIDs[p.ID] {
			missingInJSONL = append(missingInJSONL, p.ID)
		}
	}

	if len(missingInJSONL) > 0 {
		return CheckResult{
			Name:   name,
			Passed: false,
			Issues: []string{fmt.Sprintf("in SQLite but not JSONL: %v", missingInJSONL)},
		}
	}

	return CheckResult{Name: fmt.Sprintf("JSONL/SQLite in sync (%d pearls)", dbCount), Passed: true}
}

func checkOrphanedContent(store *storage.Store) CheckResult {
	name := "No orphaned content files"

	files, err := store.Content().ListFiles()
	if err != nil {
		return CheckResult{Name: name, Passed: false, Issues: []string{fmt.Sprintf("list files: %v", err)}}
	}

	pearls, err := store.DB().All()
	if err != nil {
		return CheckResult{Name: name, Passed: false, Issues: []string{fmt.Sprintf("list pearls: %v", err)}}
	}

	pearlPaths := make(map[string]bool)
	for _, p := range pearls {
		if p.ContentPath != "" {
			pearlPaths[p.ContentPath] = true
		}
	}

	var orphans []string
	for _, f := range files {
		if !pearlPaths[f] {
			orphans = append(orphans, f)
		}
	}

	if len(orphans) > 0 {
		issues := make([]string, len(orphans))
		for i, o := range orphans {
			issues[i] = o
		}
		return CheckResult{Name: name, Passed: false, Issues: issues}
	}

	return CheckResult{Name: name, Passed: true}
}

func checkMissingContent(store *storage.Store) CheckResult {
	name := "No missing content files"

	pearls, err := store.DB().All()
	if err != nil {
		return CheckResult{Name: name, Passed: false, Issues: []string{fmt.Sprintf("list pearls: %v", err)}}
	}

	var missing []string
	for _, p := range pearls {
		if p.ContentPath != "" && !store.Content().Exists(p.ContentPath) {
			missing = append(missing, p.ID)
		}
	}

	if len(missing) > 0 {
		return CheckResult{
			Name:   name,
			Passed: false,
			Issues: []string{fmt.Sprintf("%d pearls missing content: %v", len(missing), missing)},
		}
	}

	return CheckResult{Name: name, Passed: true}
}

func checkBrokenReferences(store *storage.Store) CheckResult {
	name := "All references valid"

	pearls, err := store.DB().All()
	if err != nil {
		return CheckResult{Name: name, Passed: false, Issues: []string{fmt.Sprintf("list pearls: %v", err)}}
	}

	ids := make(map[string]bool)
	for _, p := range pearls {
		ids[p.ID] = true
	}

	var broken []string
	for _, p := range pearls {
		for _, ref := range p.References {
			if !ids[ref] {
				broken = append(broken, fmt.Sprintf("%s -> %s", p.ID, ref))
			}
		}
	}

	if len(broken) > 0 {
		return CheckResult{Name: name, Passed: false, Issues: broken}
	}

	return CheckResult{Name: name, Passed: true}
}

func checkConfigValidity(configPath string) CheckResult {
	name := "Config valid"

	_, err := config.Load(configPath)
	if err != nil {
		return CheckResult{Name: name, Passed: false, Issues: []string{err.Error()}}
	}

	return CheckResult{Name: name, Passed: true}
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/cmd/ -run TestCheck -v`
Expected: All tests PASS.

**Step 5: Commit**

```bash
git add internal/cmd/doctor.go internal/cmd/doctor_test.go
git commit -m "feat: add pearls doctor command"
```

---

## Task 4: Add `godotenv` and Database Driver Dependencies

Add the new Go dependencies needed for the introspect feature.

**Files:**
- Modify: `go.mod`

**Step 1: Add dependencies**

Run:
```bash
cd /Users/robertschmit/Projects/pearls && go get github.com/joho/godotenv@v1.5.1 github.com/lib/pq@latest github.com/go-sql-driver/mysql@latest
```

**Step 2: Tidy**

Run:
```bash
go mod tidy
```

**Step 3: Verify**

Run:
```bash
grep -E 'godotenv|lib/pq|go-sql-driver' go.mod
```
Expected: All three dependencies listed.

**Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add database driver dependencies for introspect"
```

---

## Task 5: Introspect Interface and Types

Define the introspection interface and shared data types.

**Files:**
- Create: `internal/introspect/introspect.go`
- Test: `internal/introspect/introspect_test.go`

**Step 1: Write the failing test**

Create `internal/introspect/introspect_test.go`:

```go
package introspect

import (
	"testing"
)

func TestSchemaTypes(t *testing.T) {
	// Verify types are constructible
	col := Column{
		Name:       "id",
		DataType:   "bigint",
		Nullable:   false,
		Default:    "nextval('users_id_seq')",
		PrimaryKey: true,
	}
	if col.Name != "id" {
		t.Errorf("Name = %q, want id", col.Name)
	}

	fk := ForeignKey{
		Column:          "org_id",
		ReferencesTable: "organizations",
		ReferencesCol:   "id",
	}
	if fk.Column != "org_id" {
		t.Errorf("Column = %q, want org_id", fk.Column)
	}

	idx := Index{
		Name:    "users_pkey",
		Columns: []string{"id"},
		Unique:  true,
	}
	if !idx.Unique {
		t.Error("expected unique index")
	}

	tbl := Table{
		Name:        "users",
		Schema:      "public",
		Columns:     []Column{col},
		ForeignKeys: []ForeignKey{fk},
		Indexes:     []Index{idx},
	}
	if tbl.Name != "users" {
		t.Errorf("Name = %q, want users", tbl.Name)
	}
}

func TestGenerateTableContent(t *testing.T) {
	tbl := Table{
		Name:   "users",
		Schema: "public",
		Columns: []Column{
			{Name: "id", DataType: "bigint", Nullable: false, Default: "nextval('users_id_seq')", PrimaryKey: true},
			{Name: "email", DataType: "varchar(255)", Nullable: false, Constraints: "UNIQUE"},
			{Name: "name", DataType: "text", Nullable: true},
		},
		ForeignKeys: []ForeignKey{
			{Column: "org_id", ReferencesTable: "organizations", ReferencesCol: "id"},
		},
		Indexes: []Index{
			{Name: "users_pkey", Columns: []string{"id"}, Unique: true},
			{Name: "users_email_idx", Columns: []string{"email"}, Unique: true},
		},
	}

	content := GenerateTableContent(tbl, "db.postgres")
	if content == "" {
		t.Fatal("content should not be empty")
	}
	if !containsAll(content, "users", "bigint", "varchar(255)", "org_id", "organizations", "users_pkey") {
		t.Error("content missing expected fields")
	}
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !contains(s, sub) {
			return false
		}
	}
	return true
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && searchString(s, sub)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/introspect/ -run TestSchema -v`
Expected: FAIL — package doesn't exist.

**Step 3: Implement the interface and types**

Create `internal/introspect/introspect.go`:

```go
// Package introspect provides database schema introspection.
package introspect

import (
	"fmt"
	"strings"
)

// Introspector connects to a database and discovers schemas and tables.
type Introspector interface {
	// Connect establishes a connection to the database.
	Connect(connStr string) error
	// Schemas returns all schemas in the database.
	Schemas() ([]string, error)
	// Tables returns all tables in the given schema.
	Tables(schema string) ([]Table, error)
	// Close closes the database connection.
	Close() error
}

// Table represents a discovered database table.
type Table struct {
	Name        string
	Schema      string
	Columns     []Column
	ForeignKeys []ForeignKey
	Indexes     []Index
}

// Column represents a table column.
type Column struct {
	Name        string
	DataType    string
	Nullable    bool
	Default     string
	PrimaryKey  bool
	Constraints string
}

// ForeignKey represents a foreign key relationship.
type ForeignKey struct {
	Column          string
	ReferencesTable string
	ReferencesCol   string
	ReferencesSchema string
}

// Index represents a table index.
type Index struct {
	Name    string
	Columns []string
	Unique  bool
}

// GenerateTableContent produces markdown documentation for a table.
func GenerateTableContent(tbl Table, prefix string) string {
	var sb strings.Builder

	sb.WriteString("# ")
	sb.WriteString(tbl.Name)
	sb.WriteString("\n\n")

	// Columns
	sb.WriteString("## Columns\n\n")
	sb.WriteString("| Column | Type | Nullable | Default | Constraints |\n")
	sb.WriteString("|--------|------|----------|---------|-------------|\n")
	for _, col := range tbl.Columns {
		nullable := "NO"
		if col.Nullable {
			nullable = "YES"
		}
		constraints := col.Constraints
		if col.PrimaryKey {
			if constraints != "" {
				constraints = "PRIMARY KEY, " + constraints
			} else {
				constraints = "PRIMARY KEY"
			}
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n",
			col.Name, col.DataType, nullable, col.Default, constraints))
	}

	// Foreign Keys
	if len(tbl.ForeignKeys) > 0 {
		sb.WriteString("\n## Foreign Keys\n\n")
		sb.WriteString("| Column | References |\n")
		sb.WriteString("|--------|-----------|\n")
		for _, fk := range tbl.ForeignKeys {
			refSchema := fk.ReferencesSchema
			if refSchema == "" {
				refSchema = tbl.Schema
			}
			refID := fmt.Sprintf("%s.%s.%s.%s", prefix, refSchema, fk.ReferencesTable, fk.ReferencesCol)
			sb.WriteString(fmt.Sprintf("| %s | %s |\n", fk.Column, refID))
		}
	}

	// Indexes
	if len(tbl.Indexes) > 0 {
		sb.WriteString("\n## Indexes\n\n")
		sb.WriteString("| Name | Columns | Unique |\n")
		sb.WriteString("|------|---------|--------|\n")
		for _, idx := range tbl.Indexes {
			unique := "NO"
			if idx.Unique {
				unique = "YES"
			}
			sb.WriteString(fmt.Sprintf("| %s | %s | %s |\n",
				idx.Name, strings.Join(idx.Columns, ", "), unique))
		}
	}

	return sb.String()
}

// DefaultEnvVar returns the default environment variable name for a database type.
func DefaultEnvVar(dbType string) string {
	switch strings.ToLower(dbType) {
	case "postgres":
		return "PEARLS_POSTGRES_URL"
	case "mysql":
		return "PEARLS_MYSQL_URL"
	case "sqlite":
		return "PEARLS_SQLITE_PATH"
	default:
		return ""
	}
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/introspect/ -v`
Expected: All tests PASS.

**Step 5: Commit**

```bash
git add internal/introspect/introspect.go internal/introspect/introspect_test.go
git commit -m "feat: add introspect interface and types"
```

---

## Task 6: PostgreSQL Introspector

Implements the `Introspector` interface for PostgreSQL.

**Files:**
- Create: `internal/introspect/postgres.go`
- Test: `internal/introspect/postgres_test.go`

**Step 1: Write the test**

Create `internal/introspect/postgres_test.go`:

```go
package introspect

import (
	"os"
	"testing"
)

// TestPostgresIntrospector requires a live Postgres connection.
// Set PEARLS_TEST_POSTGRES_URL to run.
func TestPostgresIntrospector(t *testing.T) {
	connStr := os.Getenv("PEARLS_TEST_POSTGRES_URL")
	if connStr == "" {
		t.Skip("PEARLS_TEST_POSTGRES_URL not set, skipping Postgres integration test")
	}

	pg := &PostgresIntrospector{}
	if err := pg.Connect(connStr); err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pg.Close()

	schemas, err := pg.Schemas()
	if err != nil {
		t.Fatalf("schemas: %v", err)
	}
	if len(schemas) == 0 {
		t.Fatal("expected at least one schema")
	}

	t.Logf("schemas: %v", schemas)

	// Try to get tables from first schema
	tables, err := pg.Tables(schemas[0])
	if err != nil {
		t.Fatalf("tables: %v", err)
	}
	t.Logf("tables in %s: %d", schemas[0], len(tables))

	for _, tbl := range tables {
		if len(tbl.Columns) == 0 {
			t.Errorf("table %s has no columns", tbl.Name)
		}
		t.Logf("  %s: %d columns, %d fks, %d indexes",
			tbl.Name, len(tbl.Columns), len(tbl.ForeignKeys), len(tbl.Indexes))
	}
}
```

**Step 2: Implement PostgreSQL introspector**

Create `internal/introspect/postgres.go`:

```go
package introspect

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

// PostgresIntrospector introspects PostgreSQL databases.
type PostgresIntrospector struct {
	db *sql.DB
}

func (p *PostgresIntrospector) Connect(connStr string) error {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("open postgres: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("ping postgres: %w", err)
	}
	p.db = db
	return nil
}

func (p *PostgresIntrospector) Schemas() ([]string, error) {
	rows, err := p.db.Query(`
		SELECT schema_name
		FROM information_schema.schemata
		WHERE schema_name NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		ORDER BY schema_name
	`)
	if err != nil {
		return nil, fmt.Errorf("query schemas: %w", err)
	}
	defer rows.Close()

	var schemas []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, fmt.Errorf("scan schema: %w", err)
		}
		schemas = append(schemas, s)
	}
	return schemas, rows.Err()
}

func (p *PostgresIntrospector) Tables(schema string) ([]Table, error) {
	rows, err := p.db.Query(`
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = $1 AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`, schema)
	if err != nil {
		return nil, fmt.Errorf("query tables: %w", err)
	}
	defer rows.Close()

	var tables []Table
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan table: %w", err)
		}

		tbl := Table{Name: name, Schema: schema}

		cols, err := p.columns(schema, name)
		if err != nil {
			return nil, fmt.Errorf("columns for %s.%s: %w", schema, name, err)
		}
		tbl.Columns = cols

		fks, err := p.foreignKeys(schema, name)
		if err != nil {
			return nil, fmt.Errorf("foreign keys for %s.%s: %w", schema, name, err)
		}
		tbl.ForeignKeys = fks

		idxs, err := p.indexes(schema, name)
		if err != nil {
			return nil, fmt.Errorf("indexes for %s.%s: %w", schema, name, err)
		}
		tbl.Indexes = idxs

		tables = append(tables, tbl)
	}

	return tables, rows.Err()
}

func (p *PostgresIntrospector) columns(schema, table string) ([]Column, error) {
	rows, err := p.db.Query(`
		SELECT
			c.column_name,
			c.data_type,
			c.is_nullable,
			COALESCE(c.column_default, ''),
			CASE WHEN pk.column_name IS NOT NULL THEN true ELSE false END as is_pk
		FROM information_schema.columns c
		LEFT JOIN (
			SELECT ku.column_name
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage ku
				ON tc.constraint_name = ku.constraint_name
				AND tc.table_schema = ku.table_schema
			WHERE tc.constraint_type = 'PRIMARY KEY'
				AND tc.table_schema = $1
				AND tc.table_name = $2
		) pk ON c.column_name = pk.column_name
		WHERE c.table_schema = $1 AND c.table_name = $2
		ORDER BY c.ordinal_position
	`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []Column
	for rows.Next() {
		var col Column
		var nullable string
		if err := rows.Scan(&col.Name, &col.DataType, &nullable, &col.Default, &col.PrimaryKey); err != nil {
			return nil, err
		}
		col.Nullable = nullable == "YES"
		cols = append(cols, col)
	}
	return cols, rows.Err()
}

func (p *PostgresIntrospector) foreignKeys(schema, table string) ([]ForeignKey, error) {
	rows, err := p.db.Query(`
		SELECT
			kcu.column_name,
			ccu.table_name AS references_table,
			ccu.column_name AS references_column,
			ccu.table_schema AS references_schema
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage ccu
			ON tc.constraint_name = ccu.constraint_name
			AND tc.table_schema = ccu.table_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
			AND tc.table_schema = $1
			AND tc.table_name = $2
		ORDER BY kcu.column_name
	`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fks []ForeignKey
	for rows.Next() {
		var fk ForeignKey
		if err := rows.Scan(&fk.Column, &fk.ReferencesTable, &fk.ReferencesCol, &fk.ReferencesSchema); err != nil {
			return nil, err
		}
		fks = append(fks, fk)
	}
	return fks, rows.Err()
}

func (p *PostgresIntrospector) indexes(schema, table string) ([]Index, error) {
	rows, err := p.db.Query(`
		SELECT
			i.relname AS index_name,
			array_to_string(ARRAY(
				SELECT a.attname
				FROM pg_attribute a
				WHERE a.attrelid = i.oid
				ORDER BY a.attnum
			), ', ') AS columns,
			ix.indisunique
		FROM pg_index ix
		JOIN pg_class t ON t.oid = ix.indrelid
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_namespace n ON n.oid = t.relnamespace
		WHERE n.nspname = $1 AND t.relname = $2
		ORDER BY i.relname
	`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var idxs []Index
	for rows.Next() {
		var idx Index
		var colStr string
		if err := rows.Scan(&idx.Name, &colStr, &idx.Unique); err != nil {
			return nil, err
		}
		idx.Columns = splitColumns(colStr)
		idxs = append(idxs, idx)
	}
	return idxs, rows.Err()
}

func (p *PostgresIntrospector) Close() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}
```

**Step 3: Verify compilation**

Run: `go build ./internal/introspect/`
Expected: Compiles without errors.

**Step 4: Commit**

```bash
git add internal/introspect/postgres.go internal/introspect/postgres_test.go
git commit -m "feat: add PostgreSQL introspector"
```

---

## Task 7: MySQL Introspector

Implements the `Introspector` interface for MySQL.

**Files:**
- Create: `internal/introspect/mysql.go`
- Test: `internal/introspect/mysql_test.go`

**Step 1: Write the test**

Create `internal/introspect/mysql_test.go`:

```go
package introspect

import (
	"os"
	"testing"
)

// TestMySQLIntrospector requires a live MySQL connection.
// Set PEARLS_TEST_MYSQL_URL to run.
func TestMySQLIntrospector(t *testing.T) {
	connStr := os.Getenv("PEARLS_TEST_MYSQL_URL")
	if connStr == "" {
		t.Skip("PEARLS_TEST_MYSQL_URL not set, skipping MySQL integration test")
	}

	my := &MySQLIntrospector{}
	if err := my.Connect(connStr); err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer my.Close()

	schemas, err := my.Schemas()
	if err != nil {
		t.Fatalf("schemas: %v", err)
	}
	if len(schemas) == 0 {
		t.Fatal("expected at least one schema")
	}

	t.Logf("schemas: %v", schemas)

	tables, err := my.Tables(schemas[0])
	if err != nil {
		t.Fatalf("tables: %v", err)
	}
	t.Logf("tables in %s: %d", schemas[0], len(tables))
}
```

**Step 2: Implement MySQL introspector**

Create `internal/introspect/mysql.go`:

```go
package introspect

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

// MySQLIntrospector introspects MySQL databases.
type MySQLIntrospector struct {
	db *sql.DB
}

func (m *MySQLIntrospector) Connect(connStr string) error {
	// Strip mysql:// prefix if present for go-sql-driver compatibility
	connStr = strings.TrimPrefix(connStr, "mysql://")

	db, err := sql.Open("mysql", connStr)
	if err != nil {
		return fmt.Errorf("open mysql: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("ping mysql: %w", err)
	}
	m.db = db
	return nil
}

func (m *MySQLIntrospector) Schemas() ([]string, error) {
	rows, err := m.db.Query(`
		SELECT SCHEMA_NAME
		FROM information_schema.SCHEMATA
		WHERE SCHEMA_NAME NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys')
		ORDER BY SCHEMA_NAME
	`)
	if err != nil {
		return nil, fmt.Errorf("query schemas: %w", err)
	}
	defer rows.Close()

	var schemas []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		schemas = append(schemas, s)
	}
	return schemas, rows.Err()
}

func (m *MySQLIntrospector) Tables(schema string) ([]Table, error) {
	rows, err := m.db.Query(`
		SELECT TABLE_NAME
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = ? AND TABLE_TYPE = 'BASE TABLE'
		ORDER BY TABLE_NAME
	`, schema)
	if err != nil {
		return nil, fmt.Errorf("query tables: %w", err)
	}
	defer rows.Close()

	var tables []Table
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}

		tbl := Table{Name: name, Schema: schema}

		cols, err := m.columns(schema, name)
		if err != nil {
			return nil, fmt.Errorf("columns for %s.%s: %w", schema, name, err)
		}
		tbl.Columns = cols

		fks, err := m.foreignKeys(schema, name)
		if err != nil {
			return nil, fmt.Errorf("foreign keys for %s.%s: %w", schema, name, err)
		}
		tbl.ForeignKeys = fks

		idxs, err := m.indexes(schema, name)
		if err != nil {
			return nil, fmt.Errorf("indexes for %s.%s: %w", schema, name, err)
		}
		tbl.Indexes = idxs

		tables = append(tables, tbl)
	}

	return tables, rows.Err()
}

func (m *MySQLIntrospector) columns(schema, table string) ([]Column, error) {
	rows, err := m.db.Query(`
		SELECT
			COLUMN_NAME,
			COLUMN_TYPE,
			IS_NULLABLE,
			COALESCE(COLUMN_DEFAULT, ''),
			COLUMN_KEY
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION
	`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []Column
	for rows.Next() {
		var col Column
		var nullable, key string
		if err := rows.Scan(&col.Name, &col.DataType, &nullable, &col.Default, &key); err != nil {
			return nil, err
		}
		col.Nullable = nullable == "YES"
		col.PrimaryKey = key == "PRI"
		if key == "UNI" {
			col.Constraints = "UNIQUE"
		}
		cols = append(cols, col)
	}
	return cols, rows.Err()
}

func (m *MySQLIntrospector) foreignKeys(schema, table string) ([]ForeignKey, error) {
	rows, err := m.db.Query(`
		SELECT
			COLUMN_NAME,
			REFERENCED_TABLE_NAME,
			REFERENCED_COLUMN_NAME,
			REFERENCED_TABLE_SCHEMA
		FROM information_schema.KEY_COLUMN_USAGE
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
			AND REFERENCED_TABLE_NAME IS NOT NULL
		ORDER BY COLUMN_NAME
	`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fks []ForeignKey
	for rows.Next() {
		var fk ForeignKey
		if err := rows.Scan(&fk.Column, &fk.ReferencesTable, &fk.ReferencesCol, &fk.ReferencesSchema); err != nil {
			return nil, err
		}
		fks = append(fks, fk)
	}
	return fks, rows.Err()
}

func (m *MySQLIntrospector) indexes(schema, table string) ([]Index, error) {
	rows, err := m.db.Query(`
		SELECT
			INDEX_NAME,
			GROUP_CONCAT(COLUMN_NAME ORDER BY SEQ_IN_INDEX),
			CASE WHEN NON_UNIQUE = 0 THEN true ELSE false END
		FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		GROUP BY INDEX_NAME, NON_UNIQUE
		ORDER BY INDEX_NAME
	`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var idxs []Index
	for rows.Next() {
		var idx Index
		var colStr string
		if err := rows.Scan(&idx.Name, &colStr, &idx.Unique); err != nil {
			return nil, err
		}
		idx.Columns = splitColumns(colStr)
		idxs = append(idxs, idx)
	}
	return idxs, rows.Err()
}

func (m *MySQLIntrospector) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}
```

**Step 3: Verify compilation**

Run: `go build ./internal/introspect/`
Expected: Compiles without errors.

**Step 4: Commit**

```bash
git add internal/introspect/mysql.go internal/introspect/mysql_test.go
git commit -m "feat: add MySQL introspector"
```

---

## Task 8: SQLite Introspector

Implements the `Introspector` interface for SQLite.

**Files:**
- Create: `internal/introspect/sqlite.go`
- Test: `internal/introspect/sqlite_test.go`

**Step 1: Write the failing test**

Create `internal/introspect/sqlite_test.go`:

```go
package introspect

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestSQLiteIntrospector(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create a test SQLite database with some tables
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL UNIQUE,
			name TEXT
		);
		CREATE TABLE orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL REFERENCES users(id),
			total REAL NOT NULL DEFAULT 0.0,
			created_at TEXT NOT NULL
		);
		CREATE INDEX idx_orders_user ON orders(user_id);
	`)
	if err != nil {
		t.Fatalf("create tables: %v", err)
	}
	db.Close()

	// Introspect
	si := &SQLiteIntrospector{}
	if err := si.Connect(dbPath); err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer si.Close()

	schemas, err := si.Schemas()
	if err != nil {
		t.Fatalf("schemas: %v", err)
	}
	// SQLite has a single "main" schema
	if len(schemas) != 1 || schemas[0] != "main" {
		t.Errorf("schemas = %v, want [main]", schemas)
	}

	tables, err := si.Tables("main")
	if err != nil {
		t.Fatalf("tables: %v", err)
	}
	if len(tables) != 2 {
		t.Fatalf("expected 2 tables, got %d", len(tables))
	}

	// Check users table
	var usersTable Table
	for _, tbl := range tables {
		if tbl.Name == "users" {
			usersTable = tbl
			break
		}
	}
	if usersTable.Name == "" {
		t.Fatal("users table not found")
	}
	if len(usersTable.Columns) != 3 {
		t.Errorf("users columns = %d, want 3", len(usersTable.Columns))
	}

	// Check orders table has FK
	var ordersTable Table
	for _, tbl := range tables {
		if tbl.Name == "orders" {
			ordersTable = tbl
			break
		}
	}
	if len(ordersTable.ForeignKeys) != 1 {
		t.Errorf("orders FKs = %d, want 1", len(ordersTable.ForeignKeys))
	}
	if len(ordersTable.ForeignKeys) > 0 {
		fk := ordersTable.ForeignKeys[0]
		if fk.Column != "user_id" {
			t.Errorf("FK column = %q, want user_id", fk.Column)
		}
		if fk.ReferencesTable != "users" {
			t.Errorf("FK references = %q, want users", fk.ReferencesTable)
		}
	}

	// Check index
	if len(ordersTable.Indexes) == 0 {
		t.Error("orders should have indexes")
	}
}

func TestSQLiteIntrospectorFileNotFound(t *testing.T) {
	si := &SQLiteIntrospector{}
	err := si.Connect("/nonexistent/path/to/db")
	// SQLite creates files on open, so check if it actually can't connect
	if err == nil {
		si.Close()
		os.Remove("/nonexistent/path/to/db")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/introspect/ -run TestSQLite -v`
Expected: FAIL — type not defined.

**Step 3: Implement SQLite introspector**

Create `internal/introspect/sqlite.go`:

```go
package introspect

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteIntrospector introspects SQLite databases.
type SQLiteIntrospector struct {
	db *sql.DB
}

func (s *SQLiteIntrospector) Connect(connStr string) error {
	db, err := sql.Open("sqlite3", connStr+"?mode=ro")
	if err != nil {
		return fmt.Errorf("open sqlite: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("ping sqlite: %w", err)
	}
	s.db = db
	return nil
}

func (s *SQLiteIntrospector) Schemas() ([]string, error) {
	return []string{"main"}, nil
}

func (s *SQLiteIntrospector) Tables(schema string) ([]Table, error) {
	rows, err := s.db.Query(`
		SELECT name FROM sqlite_master
		WHERE type = 'table' AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("query tables: %w", err)
	}
	defer rows.Close()

	var tables []Table
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}

		tbl := Table{Name: name, Schema: schema}

		cols, err := s.columns(name)
		if err != nil {
			return nil, fmt.Errorf("columns for %s: %w", name, err)
		}
		tbl.Columns = cols

		fks, err := s.foreignKeys(name)
		if err != nil {
			return nil, fmt.Errorf("foreign keys for %s: %w", name, err)
		}
		tbl.ForeignKeys = fks

		idxs, err := s.indexes(name)
		if err != nil {
			return nil, fmt.Errorf("indexes for %s: %w", name, err)
		}
		tbl.Indexes = idxs

		tables = append(tables, tbl)
	}

	return tables, rows.Err()
}

func (s *SQLiteIntrospector) columns(table string) ([]Column, error) {
	rows, err := s.db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []Column
	for rows.Next() {
		var cid int
		var col Column
		var notNull int
		var pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &col.Name, &col.DataType, &notNull, &dflt, &pk); err != nil {
			return nil, err
		}
		col.Nullable = notNull == 0
		col.PrimaryKey = pk > 0
		if dflt.Valid {
			col.Default = dflt.String
		}
		cols = append(cols, col)
	}
	return cols, rows.Err()
}

func (s *SQLiteIntrospector) foreignKeys(table string) ([]ForeignKey, error) {
	rows, err := s.db.Query(fmt.Sprintf("PRAGMA foreign_key_list(%s)", table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fks []ForeignKey
	for rows.Next() {
		var id, seq int
		var refTable, from, to, onUpdate, onDelete, match string
		if err := rows.Scan(&id, &seq, &refTable, &from, &to, &onUpdate, &onDelete, &match); err != nil {
			return nil, err
		}
		fks = append(fks, ForeignKey{
			Column:          from,
			ReferencesTable: refTable,
			ReferencesCol:   to,
		})
	}
	return fks, rows.Err()
}

func (s *SQLiteIntrospector) indexes(table string) ([]Index, error) {
	rows, err := s.db.Query(fmt.Sprintf("PRAGMA index_list(%s)", table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var idxs []Index
	for rows.Next() {
		var seq int
		var idx Index
		var origin, partial string
		if err := rows.Scan(&seq, &idx.Name, &idx.Unique, &origin, &partial); err != nil {
			return nil, err
		}

		// Get columns for this index
		colRows, err := s.db.Query(fmt.Sprintf("PRAGMA index_info(%s)", idx.Name))
		if err != nil {
			return nil, err
		}
		for colRows.Next() {
			var seqno, cid int
			var colName string
			if err := colRows.Scan(&seqno, &cid, &colName); err != nil {
				colRows.Close()
				return nil, err
			}
			idx.Columns = append(idx.Columns, colName)
		}
		colRows.Close()

		idxs = append(idxs, idx)
	}
	return idxs, rows.Err()
}

func (s *SQLiteIntrospector) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// splitColumns splits a comma-separated column list.
func splitColumns(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, len(parts))
	for i, p := range parts {
		result[i] = strings.TrimSpace(p)
	}
	return result
}
```

**Step 4: Run tests**

Run: `go test ./internal/introspect/ -run TestSQLite -v`
Expected: All tests PASS.

**Step 5: Commit**

```bash
git add internal/introspect/sqlite.go internal/introspect/sqlite_test.go
git commit -m "feat: add SQLite introspector"
```

---

## Task 9: Pearl Generation Logic

Converts introspected schemas/tables into pearl objects.

**Files:**
- Create: `internal/introspect/generate.go`
- Test: `internal/introspect/generate_test.go`

**Step 1: Write the failing test**

Create `internal/introspect/generate_test.go`:

```go
package introspect

import (
	"testing"

	"github.com/justrnr500/pearls/internal/pearl"
)

func TestGeneratePearls(t *testing.T) {
	tables := map[string][]Table{
		"public": {
			{
				Name:   "users",
				Schema: "public",
				Columns: []Column{
					{Name: "id", DataType: "bigint", PrimaryKey: true},
					{Name: "email", DataType: "varchar(255)"},
				},
				ForeignKeys: []ForeignKey{},
				Indexes: []Index{
					{Name: "users_pkey", Columns: []string{"id"}, Unique: true},
				},
			},
			{
				Name:   "orders",
				Schema: "public",
				Columns: []Column{
					{Name: "id", DataType: "bigint", PrimaryKey: true},
					{Name: "user_id", DataType: "bigint"},
				},
				ForeignKeys: []ForeignKey{
					{Column: "user_id", ReferencesTable: "users", ReferencesCol: "id"},
				},
				Indexes: []Index{},
			},
		},
	}

	result := GeneratePearls("db.postgres", tables, "PEARLS_POSTGRES_URL")

	// Should create: 1 database pearl + 1 schema pearl + 2 table pearls = 4
	if len(result) != 4 {
		t.Fatalf("expected 4 pearls, got %d", len(result))
	}

	// Check database pearl
	dbPearl := findPearl(result, "db.postgres")
	if dbPearl == nil {
		t.Fatal("database pearl not found")
	}
	if dbPearl.Type != pearl.TypeDatabase {
		t.Errorf("db pearl type = %q, want database", dbPearl.Type)
	}

	// Check schema pearl
	schemaPearl := findPearl(result, "db.postgres.public")
	if schemaPearl == nil {
		t.Fatal("schema pearl not found")
	}
	if schemaPearl.Type != pearl.TypeSchema {
		t.Errorf("schema pearl type = %q, want schema", schemaPearl.Type)
	}
	if schemaPearl.Parent != "db.postgres" {
		t.Errorf("schema parent = %q, want db.postgres", schemaPearl.Parent)
	}

	// Check table pearl
	usersPearl := findPearl(result, "db.postgres.public.users")
	if usersPearl == nil {
		t.Fatal("users pearl not found")
	}
	if usersPearl.Type != pearl.TypeTable {
		t.Errorf("users type = %q, want table", usersPearl.Type)
	}
	if usersPearl.Parent != "db.postgres.public" {
		t.Errorf("users parent = %q, want db.postgres.public", usersPearl.Parent)
	}

	// Check orders pearl has reference to users
	ordersPearl := findPearl(result, "db.postgres.public.orders")
	if ordersPearl == nil {
		t.Fatal("orders pearl not found")
	}
	if len(ordersPearl.References) != 1 {
		t.Fatalf("orders refs = %d, want 1", len(ordersPearl.References))
	}
	if ordersPearl.References[0] != "db.postgres.public.users" {
		t.Errorf("orders ref = %q, want db.postgres.public.users", ordersPearl.References[0])
	}
}

func TestGeneratePearlsContent(t *testing.T) {
	tables := map[string][]Table{
		"public": {
			{
				Name:   "users",
				Schema: "public",
				Columns: []Column{
					{Name: "id", DataType: "bigint", PrimaryKey: true},
				},
			},
		},
	}

	result := GeneratePearls("db.pg", tables, "DB_URL")
	usersPearl := findPearl(result, "db.pg.public.users")
	if usersPearl == nil {
		t.Fatal("users pearl not found")
	}
	if usersPearl.GeneratedContent == "" {
		t.Error("expected generated content")
	}
}

func findPearl(pearls []GeneratedPearl, id string) *GeneratedPearl {
	for i := range pearls {
		if pearls[i].Pearl.ID == id {
			return &pearls[i]
		}
	}
	return nil
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/introspect/ -run TestGenerate -v`
Expected: FAIL — function not defined.

**Step 3: Implement generation logic**

Create `internal/introspect/generate.go`:

```go
package introspect

import (
	"fmt"
	"time"

	"github.com/justrnr500/pearls/internal/pearl"
)

// GeneratedPearl pairs a Pearl with its markdown content.
type GeneratedPearl struct {
	Pearl            pearl.Pearl
	GeneratedContent string
}

// GeneratePearls creates pearl objects from introspected tables.
// tables is keyed by schema name.
func GeneratePearls(prefix string, tables map[string][]Table, envVar string) []GeneratedPearl {
	var result []GeneratedPearl
	now := time.Now()

	// Database pearl
	dbPearl := GeneratedPearl{
		Pearl: pearl.Pearl{
			ID:          prefix,
			Name:        pearl.LastSegment(prefix),
			Namespace:   pearl.ParentNamespace(prefix),
			Type:        pearl.TypeDatabase,
			Description: fmt.Sprintf("Database introspected from ${%s}", envVar),
			Status:      pearl.StatusActive,
			Connection: &pearl.ConnectionInfo{
				Type: "database",
				Host: fmt.Sprintf("${%s}", envVar),
			},
			CreatedAt: now,
			UpdatedAt: now,
			CreatedBy: "pearls-introspect",
		},
	}
	result = append(result, dbPearl)

	for schemaName, schemaTables := range tables {
		schemaID := prefix + "." + schemaName

		// Schema pearl
		schemaPearl := GeneratedPearl{
			Pearl: pearl.Pearl{
				ID:          schemaID,
				Name:        schemaName,
				Namespace:   prefix,
				Type:        pearl.TypeSchema,
				Description: fmt.Sprintf("Schema %s", schemaName),
				Parent:      prefix,
				Status:      pearl.StatusActive,
				CreatedAt:   now,
				UpdatedAt:   now,
				CreatedBy:   "pearls-introspect",
			},
		}
		result = append(result, schemaPearl)

		for _, tbl := range schemaTables {
			tableID := schemaID + "." + tbl.Name

			// Build references from foreign keys
			var refs []string
			for _, fk := range tbl.ForeignKeys {
				refSchema := fk.ReferencesSchema
				if refSchema == "" {
					refSchema = schemaName
				}
				refs = append(refs, prefix+"."+refSchema+"."+fk.ReferencesTable)
			}

			content := GenerateTableContent(tbl, prefix)

			tablePearl := GeneratedPearl{
				Pearl: pearl.Pearl{
					ID:          tableID,
					Name:        tbl.Name,
					Namespace:   schemaID,
					Type:        pearl.TypeTable,
					Description: fmt.Sprintf("Table %s.%s", schemaName, tbl.Name),
					References:  refs,
					Parent:      schemaID,
					Status:      pearl.StatusActive,
					CreatedAt:   now,
					UpdatedAt:   now,
					CreatedBy:   "pearls-introspect",
				},
				GeneratedContent: content,
			}
			result = append(result, tablePearl)
		}
	}

	return result
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/introspect/ -run TestGenerate -v`
Expected: All tests PASS.

**Step 5: Commit**

```bash
git add internal/introspect/generate.go internal/introspect/generate_test.go
git commit -m "feat: add pearl generation from introspected schemas"
```

---

## Task 10: `pearls introspect` CLI Command

Wires the introspect package to a Cobra command with `.env` loading.

**Files:**
- Create: `internal/cmd/introspect.go`

**Step 1: Implement the command**

Create `internal/cmd/introspect.go`:

```go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"

	"github.com/justrnr500/pearls/internal/config"
	"github.com/justrnr500/pearls/internal/introspect"
)

var introspectCmd = &cobra.Command{
	Use:   "introspect <type>",
	Short: "Auto-generate pearls from database schemas",
	Long: `Connect to a live database and generate pearls for discovered schemas and tables.

Supported types: postgres, mysql, sqlite

Credentials are read from .env in the repo root.
Default env vars: PEARLS_POSTGRES_URL, PEARLS_MYSQL_URL, PEARLS_SQLITE_PATH

Examples:
  pearls introspect postgres --prefix db.postgres
  pearls introspect mysql --prefix db.mysql --schema mydb
  pearls introspect sqlite --prefix db.local
  pearls introspect postgres --env DATABASE_URL --prefix db.main
  pearls introspect postgres --prefix db.pg --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runIntrospect,
}

var (
	introspectPrefix       string
	introspectEnv          string
	introspectSchema       string
	introspectDryRun       bool
	introspectSkipExisting bool
)

func init() {
	rootCmd.AddCommand(introspectCmd)
	introspectCmd.Flags().StringVar(&introspectPrefix, "prefix", "", "Namespace prefix for generated pearls (required)")
	introspectCmd.Flags().StringVar(&introspectEnv, "env", "", "Override env var name for connection string")
	introspectCmd.Flags().StringVar(&introspectSchema, "schema", "", "Limit to a specific schema")
	introspectCmd.Flags().BoolVar(&introspectDryRun, "dry-run", false, "Print what would be created without writing")
	introspectCmd.Flags().BoolVar(&introspectSkipExisting, "skip-existing", false, "Don't overwrite pearls that already exist")
	introspectCmd.MarkFlagRequired("prefix")
}

func runIntrospect(cmd *cobra.Command, args []string) error {
	dbType := args[0]

	// Validate type
	switch dbType {
	case "postgres", "mysql", "sqlite":
		// ok
	default:
		return fmt.Errorf("unsupported database type %q: must be postgres, mysql, or sqlite", dbType)
	}

	// Load .env from repo root
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}
	root, err := config.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("find pearls root: %w", err)
	}

	envPath := filepath.Join(root, ".env")
	godotenv.Load(envPath) // Best effort — .env may not exist

	// Determine env var
	envVar := introspectEnv
	if envVar == "" {
		envVar = introspect.DefaultEnvVar(dbType)
	}
	if envVar == "" {
		return fmt.Errorf("could not determine env var for type %q", dbType)
	}

	connStr := os.Getenv(envVar)
	if connStr == "" {
		return fmt.Errorf("connection string not found: set %s in .env or environment", envVar)
	}

	// Create introspector
	var intro introspect.Introspector
	switch dbType {
	case "postgres":
		intro = &introspect.PostgresIntrospector{}
	case "mysql":
		intro = &introspect.MySQLIntrospector{}
	case "sqlite":
		intro = &introspect.SQLiteIntrospector{}
	}

	fmt.Printf("Connecting to %s...\n", dbType)
	if err := intro.Connect(connStr); err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer intro.Close()

	// Discover schemas
	schemas, err := intro.Schemas()
	if err != nil {
		return fmt.Errorf("discover schemas: %w", err)
	}

	if introspectSchema != "" {
		// Filter to specific schema
		found := false
		for _, s := range schemas {
			if s == introspectSchema {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("schema %q not found (available: %v)", introspectSchema, schemas)
		}
		schemas = []string{introspectSchema}
	}

	fmt.Printf("Found %d schema(s): %v\n", len(schemas), schemas)

	// Discover tables per schema
	allTables := make(map[string][]introspect.Table)
	totalTables := 0
	for _, schema := range schemas {
		tables, err := intro.Tables(schema)
		if err != nil {
			return fmt.Errorf("discover tables in %s: %w", schema, err)
		}
		allTables[schema] = tables
		totalTables += len(tables)
		fmt.Printf("  %s: %d table(s)\n", schema, len(tables))
	}

	// Generate pearls
	generated := introspect.GeneratePearls(introspectPrefix, allTables, envVar)

	if introspectDryRun {
		fmt.Printf("\nDry run — would create %d pearl(s):\n", len(generated))
		for _, gp := range generated {
			fmt.Printf("  %s (%s)\n", gp.Pearl.ID, gp.Pearl.Type)
		}
		return nil
	}

	// Get store
	store, _, err := getStore()
	if err != nil {
		return err
	}
	defer store.Close()

	// Create pearls
	created := 0
	skipped := 0
	for _, gp := range generated {
		// Check if pearl already exists
		existing, _ := store.Get(gp.Pearl.ID)
		if existing != nil {
			if introspectSkipExisting {
				skipped++
				continue
			}
			// Delete existing to recreate
			store.Delete(gp.Pearl.ID)
		}

		p := gp.Pearl
		if err := store.Create(&p, gp.GeneratedContent); err != nil {
			return fmt.Errorf("create pearl %s: %w", gp.Pearl.ID, err)
		}
		created++
	}

	fmt.Printf("\n✓ Created %d pearl(s)", created)
	if skipped > 0 {
		fmt.Printf(" (%d skipped)", skipped)
	}
	fmt.Println()

	return nil
}
```

**Step 2: Verify compilation**

Run: `go build ./cmd/pearls`
Expected: Compiles without errors.

**Step 3: Test dry-run (without a database)**

Run: `go run ./cmd/pearls introspect postgres --prefix db.pg --dry-run`
Expected: Error about missing connection string (correct behavior without .env).

**Step 4: Commit**

```bash
git add internal/cmd/introspect.go
git commit -m "feat: add pearls introspect command"
```

---

## Task 11: Update `pearls init` to Gitignore `.env`

Ensure `pearls init` adds `.env` to the repo root's `.gitignore`.

**Files:**
- Modify: `internal/cmd/init.go`

**Step 1: Add `.env` gitignore logic to init**

At the end of `runInit`, after the existing `.gitignore` creation for `.pearls/`, add:

```go
// Ensure .env is in repo root .gitignore
rootGitignore := filepath.Join(cwd, ".gitignore")
ensureGitignoreEntry(rootGitignore, ".env")
```

Add helper function:

```go
func ensureGitignoreEntry(path, entry string) {
	data, _ := os.ReadFile(path)
	content := string(data)

	// Check if entry already exists
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) == entry {
			return
		}
	}

	// Append entry
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += entry + "\n"
	os.WriteFile(path, []byte(content), 0644)
}
```

Add `"strings"` to the import block.

**Step 2: Verify existing tests still pass**

Run: `go build ./cmd/pearls`
Expected: Compiles without errors.

**Step 3: Commit**

```bash
git add internal/cmd/init.go
git commit -m "feat: init ensures .env is gitignored"
```

---

## Task 12: End-to-End Integration Test

Tests the full flow: init, introspect a SQLite DB, doctor checks.

**Files:**
- Create: `internal/cmd/integration_test.go`

**Step 1: Write the integration test**

Create `internal/cmd/integration_test.go`:

```go
package cmd

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/justrnr500/pearls/internal/introspect"
	"github.com/justrnr500/pearls/internal/pearl"
	"github.com/justrnr500/pearls/internal/storage"
)

func TestIntrospectSQLiteEndToEnd(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test SQLite database
	testDBPath := filepath.Join(tmpDir, "test.db")
	db, err := sql.Open("sqlite3", testDBPath)
	if err != nil {
		t.Fatalf("create test db: %v", err)
	}
	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			name TEXT
		);
		CREATE TABLE posts (
			id INTEGER PRIMARY KEY,
			user_id INTEGER REFERENCES users(id),
			title TEXT NOT NULL
		);
	`)
	if err != nil {
		t.Fatalf("create tables: %v", err)
	}
	db.Close()

	// Introspect the database
	si := &introspect.SQLiteIntrospector{}
	if err := si.Connect(testDBPath); err != nil {
		t.Fatalf("connect: %v", err)
	}

	schemas, _ := si.Schemas()
	allTables := make(map[string][]introspect.Table)
	for _, schema := range schemas {
		tables, _ := si.Tables(schema)
		allTables[schema] = tables
	}
	si.Close()

	// Generate pearls
	generated := introspect.GeneratePearls("db.test", allTables, "TEST_DB")

	// Verify count: 1 db + 1 schema + 2 tables = 4
	if len(generated) != 4 {
		t.Fatalf("expected 4 generated pearls, got %d", len(generated))
	}

	// Create a pearls store and persist
	storeDir := filepath.Join(tmpDir, "pearls-store")
	os.MkdirAll(filepath.Join(storeDir, "content"), 0755)

	store, err := storage.NewStore(
		filepath.Join(storeDir, "pearls.db"),
		filepath.Join(storeDir, "pearls.jsonl"),
		filepath.Join(storeDir, "content"),
	)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	for _, gp := range generated {
		p := gp.Pearl
		if err := store.Create(&p, gp.GeneratedContent); err != nil {
			t.Fatalf("create pearl %s: %v", gp.Pearl.ID, err)
		}
	}

	// Verify pearls were created
	all, _ := store.DB().All()
	if len(all) != 4 {
		t.Errorf("expected 4 pearls in store, got %d", len(all))
	}

	// Verify types
	for _, p := range all {
		switch {
		case p.ID == "db.test":
			if p.Type != pearl.TypeDatabase {
				t.Errorf("%s type = %q, want database", p.ID, p.Type)
			}
		case p.ID == "db.test.main":
			if p.Type != pearl.TypeSchema {
				t.Errorf("%s type = %q, want schema", p.ID, p.Type)
			}
		default:
			if p.Type != pearl.TypeTable {
				t.Errorf("%s type = %q, want table", p.ID, p.Type)
			}
		}
	}

	// Run doctor checks on the store
	syncResult := checkJSONLSync(store)
	if !syncResult.Passed {
		t.Errorf("JSONL sync check failed: %v", syncResult.Issues)
	}

	missingResult := checkMissingContent(store)
	if !missingResult.Passed {
		t.Errorf("missing content check failed: %v", missingResult.Issues)
	}

	brokenResult := checkBrokenReferences(store)
	// This may have broken refs since FK targets may not resolve to existing pearls
	// depending on naming. For this test, just log.
	t.Logf("broken refs check: passed=%v, issues=%v", brokenResult.Passed, brokenResult.Issues)
}
```

**Step 2: Run the integration test**

Run: `go test ./internal/cmd/ -run TestIntrospectSQLiteEndToEnd -v`
Expected: PASS.

**Step 3: Commit**

```bash
git add internal/cmd/integration_test.go
git commit -m "test: add end-to-end introspect integration test"
```

---

## Task 13: Final Verification

Run all tests and verify the full build.

**Step 1: Run all tests**

Run: `go test ./... -v`
Expected: All tests PASS.

**Step 2: Build both binaries**

Run:
```bash
go build -o pearls ./cmd/pearls && go build -o pl ./cmd/pl
```
Expected: Both binaries compile.

**Step 3: Verify commands are registered**

Run:
```bash
./pearls --help
```
Expected: Shows `doctor`, `introspect`, `onboard` in the command list alongside existing commands.

**Step 4: Clean up binaries**

Run:
```bash
rm -f pearls pl
```

**Step 5: Final commit**

If any loose changes remain:
```bash
git add -A && git commit -m "chore: phase 3 final cleanup"
```
