package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/justrnr500/pearls/internal/pearl"
)

func TestDBCRUD(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "pearls-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "pearls.db")
	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	now := time.Now().Truncate(time.Second)

	// Test Insert
	p := &pearl.Pearl{
		ID:          "db.postgres.users",
		Name:        "users",
		Namespace:   "db.postgres",
		Type:        pearl.TypeTable,
		Tags:        []string{"pii", "core"},
		Description: "User accounts",
		ContentPath: "db/postgres/users.md",
		ContentHash: "abc123",
		Status:      pearl.StatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
		CreatedBy:   "test",
	}

	if err := db.Insert(p); err != nil {
		t.Fatalf("insert: %v", err)
	}

	// Test Get
	got, err := db.Get("db.postgres.users")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil {
		t.Fatal("get returned nil")
	}
	if got.ID != p.ID {
		t.Errorf("ID = %q, want %q", got.ID, p.ID)
	}
	if got.Type != p.Type {
		t.Errorf("Type = %q, want %q", got.Type, p.Type)
	}
	if len(got.Tags) != 2 {
		t.Errorf("len(Tags) = %d, want 2", len(got.Tags))
	}

	// Test Update
	p.Description = "Updated description"
	p.UpdatedAt = time.Now().Truncate(time.Second)
	if err := db.Update(p); err != nil {
		t.Fatalf("update: %v", err)
	}

	got, _ = db.Get("db.postgres.users")
	if got.Description != "Updated description" {
		t.Errorf("Description = %q, want %q", got.Description, "Updated description")
	}

	// Test List
	p2 := &pearl.Pearl{
		ID:        "db.postgres.orders",
		Name:      "orders",
		Namespace: "db.postgres",
		Type:      pearl.TypeTable,
		Status:    pearl.StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	db.Insert(p2)

	list, err := db.List(ListOptions{Namespace: "db.postgres"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("list length = %d, want 2", len(list))
	}

	// Test Delete
	if err := db.Delete("db.postgres.orders"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	got, _ = db.Get("db.postgres.orders")
	if got != nil {
		t.Error("pearl should be deleted")
	}

	// Test Count
	count, err := db.Count()
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}

func TestJSONL(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pearls-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	jsonlPath := filepath.Join(tmpDir, "pearls.jsonl")
	j := NewJSONL(jsonlPath)

	now := time.Now()

	// Test Append and ReadAll
	p1 := &pearl.Pearl{
		ID:        "test.pearl1",
		Name:      "pearl1",
		Type:      pearl.TypeTable,
		Status:    pearl.StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	p2 := &pearl.Pearl{
		ID:        "test.pearl2",
		Name:      "pearl2",
		Type:      pearl.TypeAPI,
		Status:    pearl.StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := j.Append(p1); err != nil {
		t.Fatalf("append p1: %v", err)
	}
	if err := j.Append(p2); err != nil {
		t.Fatalf("append p2: %v", err)
	}

	pearls, err := j.ReadAll()
	if err != nil {
		t.Fatalf("read all: %v", err)
	}
	if len(pearls) != 2 {
		t.Errorf("len = %d, want 2", len(pearls))
	}

	// Test WriteAll (overwrite)
	if err := j.WriteAll([]*pearl.Pearl{p1}); err != nil {
		t.Fatalf("write all: %v", err)
	}

	pearls, _ = j.ReadAll()
	if len(pearls) != 1 {
		t.Errorf("len after overwrite = %d, want 1", len(pearls))
	}
}

func TestContent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pearls-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	c := NewContent(tmpDir)

	// Test PathForPearl
	path := c.PathForPearl("db.postgres", "users")
	if path != "db/postgres/users.md" {
		t.Errorf("PathForPearl = %q, want %q", path, "db/postgres/users.md")
	}

	// Test Write and Read
	content := "# Test\n\nThis is a test."
	if err := c.Write(path, content); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := c.Read(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got != content {
		t.Errorf("content = %q, want %q", got, content)
	}

	// Test Exists
	if !c.Exists(path) {
		t.Error("file should exist")
	}

	// Test Hash
	hash, err := c.Hash(path)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if hash == "" {
		t.Error("hash should not be empty")
	}

	// Verify hash is consistent
	if HashString(content) != hash {
		t.Error("hash mismatch")
	}

	// Test Delete
	if err := c.Delete(path); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if c.Exists(path) {
		t.Error("file should be deleted")
	}
}

func TestStore(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pearls-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(
		filepath.Join(tmpDir, "pearls.db"),
		filepath.Join(tmpDir, "pearls.jsonl"),
		filepath.Join(tmpDir, "content"),
	)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	now := time.Now()

	// Test Create
	p := &pearl.Pearl{
		ID:          "db.postgres.users",
		Name:        "users",
		Namespace:   "db.postgres",
		Type:        pearl.TypeTable,
		Description: "User accounts",
		Status:      pearl.StatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	content := "# users\n\nUser accounts table."
	if err := store.Create(p, content); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Verify in database
	got, err := store.Get("db.postgres.users")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil {
		t.Fatal("pearl not found in db")
	}

	// Verify content
	gotContent, err := store.GetContent(got)
	if err != nil {
		t.Fatalf("get content: %v", err)
	}
	if gotContent != content {
		t.Errorf("content = %q, want %q", gotContent, content)
	}

	// Verify content hash was set
	if got.ContentHash == "" {
		t.Error("content hash should be set")
	}

	// Verify JSONL was written
	pearls, _ := store.JSONL().ReadAll()
	if len(pearls) != 1 {
		t.Errorf("jsonl length = %d, want 1", len(pearls))
	}

	// Test Update
	newContent := "# users\n\nUpdated content."
	got.Description = "Updated"
	got.UpdatedAt = time.Now()
	if err := store.Update(got, &newContent); err != nil {
		t.Fatalf("update: %v", err)
	}

	gotContent, _ = store.GetContent(got)
	if gotContent != newContent {
		t.Errorf("updated content = %q, want %q", gotContent, newContent)
	}

	// Test Delete
	if err := store.Delete("db.postgres.users"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	got, _ = store.Get("db.postgres.users")
	if got != nil {
		t.Error("pearl should be deleted")
	}
}

func TestFindReferencingPearls(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pearls-refs-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "pearls.db")
	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	now := time.Now()

	// Create pearls with references
	pearls := []*pearl.Pearl{
		{
			ID: "db.postgres.users", Name: "users", Namespace: "db.postgres",
			Type: pearl.TypeTable, Status: pearl.StatusActive,
			References: []string{}, // No references
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "db.postgres.orders", Name: "orders", Namespace: "db.postgres",
			Type: pearl.TypeTable, Status: pearl.StatusActive,
			References: []string{"db.postgres.users"}, // References users
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "db.postgres.payments", Name: "payments", Namespace: "db.postgres",
			Type: pearl.TypeTable, Status: pearl.StatusActive,
			References: []string{"db.postgres.users", "db.postgres.orders"}, // References both
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "api.stripe.customers", Name: "customers", Namespace: "api.stripe",
			Type: pearl.TypeAPI, Status: pearl.StatusActive,
			References: []string{"db.postgres.users"}, // Also references users
			CreatedAt: now, UpdatedAt: now,
		},
	}

	for _, p := range pearls {
		if err := db.Insert(p); err != nil {
			t.Fatalf("insert %s: %v", p.ID, err)
		}
	}

	// Test: find what references "db.postgres.users"
	refs, err := db.FindReferencingPearls("db.postgres.users")
	if err != nil {
		t.Fatalf("find referencing pearls: %v", err)
	}

	// Should find: orders, payments, customers (3 pearls reference users)
	if len(refs) != 3 {
		t.Errorf("expected 3 pearls referencing users, got %d: %v", len(refs), refs)
	}

	// Test: find what references "db.postgres.orders"
	refs, err = db.FindReferencingPearls("db.postgres.orders")
	if err != nil {
		t.Fatalf("find referencing pearls: %v", err)
	}

	// Should find: payments (1 pearl references orders)
	if len(refs) != 1 {
		t.Errorf("expected 1 pearl referencing orders, got %d: %v", len(refs), refs)
	}
	if len(refs) > 0 && refs[0] != "db.postgres.payments" {
		t.Errorf("expected payments to reference orders, got %s", refs[0])
	}

	// Test: find what references "api.stripe.customers" (nothing)
	refs, err = db.FindReferencingPearls("api.stripe.customers")
	if err != nil {
		t.Fatalf("find referencing pearls: %v", err)
	}

	if len(refs) != 0 {
		t.Errorf("expected 0 pearls referencing customers, got %d: %v", len(refs), refs)
	}
}

func TestGlobsAndScopes(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pearls-globs-scopes-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "pearls.db")
	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	now := time.Now().Truncate(time.Second)

	// Insert pearls with globs and scopes
	pearls := []*pearl.Pearl{
		{
			ID: "conv.api-style", Name: "api-style", Namespace: "conv",
			Type: pearl.TypeCustom, Status: pearl.StatusActive,
			Globs:  []string{"src/api/**/*.go", "pkg/handlers/**"},
			Scopes: []string{"backend", "api"},
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "conv.frontend-style", Name: "frontend-style", Namespace: "conv",
			Type: pearl.TypeCustom, Status: pearl.StatusActive,
			Globs:  []string{"src/web/**/*.tsx", "src/web/**/*.ts"},
			Scopes: []string{"frontend"},
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "conv.general", Name: "general", Namespace: "conv",
			Type: pearl.TypeCustom, Status: pearl.StatusActive,
			Globs:  []string{},
			Scopes: []string{"backend", "frontend"},
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "conv.no-globs", Name: "no-globs", Namespace: "conv",
			Type: pearl.TypeCustom, Status: pearl.StatusActive,
			Scopes: []string{},
			CreatedAt: now, UpdatedAt: now,
		},
	}

	for _, p := range pearls {
		if err := db.Insert(p); err != nil {
			t.Fatalf("insert %s: %v", p.ID, err)
		}
	}

	// Test roundtrip: Get should return globs and scopes
	t.Run("Roundtrip", func(t *testing.T) {
		got, err := db.Get("conv.api-style")
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if got == nil {
			t.Fatal("pearl not found")
		}
		if len(got.Globs) != 2 {
			t.Errorf("len(Globs) = %d, want 2", len(got.Globs))
		}
		if len(got.Scopes) != 2 {
			t.Errorf("len(Scopes) = %d, want 2", len(got.Scopes))
		}
		if got.Globs[0] != "src/api/**/*.go" {
			t.Errorf("Globs[0] = %q, want %q", got.Globs[0], "src/api/**/*.go")
		}
		if got.Scopes[0] != "backend" {
			t.Errorf("Scopes[0] = %q, want %q", got.Scopes[0], "backend")
		}
	})

	// Test FindByScope
	t.Run("FindByScope", func(t *testing.T) {
		// "backend" scope: api-style + general
		results, err := db.FindByScope("backend")
		if err != nil {
			t.Fatalf("find by scope: %v", err)
		}
		if len(results) != 2 {
			t.Errorf("expected 2 pearls with backend scope, got %d", len(results))
		}

		// "frontend" scope: frontend-style + general
		results, err = db.FindByScope("frontend")
		if err != nil {
			t.Fatalf("find by scope: %v", err)
		}
		if len(results) != 2 {
			t.Errorf("expected 2 pearls with frontend scope, got %d", len(results))
		}

		// "api" scope: api-style only
		results, err = db.FindByScope("api")
		if err != nil {
			t.Fatalf("find by scope: %v", err)
		}
		if len(results) != 1 {
			t.Errorf("expected 1 pearl with api scope, got %d", len(results))
		}
		if len(results) > 0 && results[0].ID != "conv.api-style" {
			t.Errorf("expected conv.api-style, got %s", results[0].ID)
		}

		// nonexistent scope
		results, err = db.FindByScope("nonexistent")
		if err != nil {
			t.Fatalf("find by scope: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("expected 0 pearls with nonexistent scope, got %d", len(results))
		}
	})

	// Test FindByGlob
	t.Run("FindByGlob", func(t *testing.T) {
		// Path matching api-style globs
		results, err := db.FindByGlob("src/api/v1/handler.go")
		if err != nil {
			t.Fatalf("find by glob: %v", err)
		}
		if len(results) != 1 {
			t.Errorf("expected 1 match for api path, got %d", len(results))
		}
		if len(results) > 0 && results[0].ID != "conv.api-style" {
			t.Errorf("expected conv.api-style, got %s", results[0].ID)
		}

		// Path matching frontend-style globs
		results, err = db.FindByGlob("src/web/components/Button.tsx")
		if err != nil {
			t.Fatalf("find by glob: %v", err)
		}
		if len(results) != 1 {
			t.Errorf("expected 1 match for frontend path, got %d", len(results))
		}
		if len(results) > 0 && results[0].ID != "conv.frontend-style" {
			t.Errorf("expected conv.frontend-style, got %s", results[0].ID)
		}

		// Path matching nothing
		results, err = db.FindByGlob("docs/README.md")
		if err != nil {
			t.Fatalf("find by glob: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("expected 0 matches for docs path, got %d", len(results))
		}
	})

	// Test Update preserves globs and scopes
	t.Run("UpdatePreservesGlobsScopes", func(t *testing.T) {
		p, _ := db.Get("conv.api-style")
		p.Globs = []string{"src/api/**/*.go", "pkg/handlers/**", "internal/api/**"}
		p.Scopes = []string{"backend", "api", "infra"}
		p.UpdatedAt = time.Now().Truncate(time.Second)

		if err := db.Update(p); err != nil {
			t.Fatalf("update: %v", err)
		}

		got, _ := db.Get("conv.api-style")
		if len(got.Globs) != 3 {
			t.Errorf("len(Globs) after update = %d, want 3", len(got.Globs))
		}
		if len(got.Scopes) != 3 {
			t.Errorf("len(Scopes) after update = %d, want 3", len(got.Scopes))
		}
	})
}

func TestEndToEndContextInjection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pearls-e2e-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(
		filepath.Join(tmpDir, "pearls.db"),
		filepath.Join(tmpDir, "pearls.jsonl"),
		filepath.Join(tmpDir, "content"),
	)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	now := time.Now()

	t.Run("CreateWithGlobs_FindByGlob", func(t *testing.T) {
		p := &pearl.Pearl{
			ID: "conv.error-handling", Name: "error-handling", Namespace: "conv",
			Type: "convention", Status: pearl.StatusActive,
			Globs:       []string{"src/api/**/*.go", "internal/handlers/**"},
			Description: "How we handle errors",
			CreatedAt:   now, UpdatedAt: now, CreatedBy: "test",
		}
		content := "# Error Handling\n\nAlways wrap errors with context."

		if err := store.Create(p, content); err != nil {
			t.Fatalf("create: %v", err)
		}

		// Should match
		results, err := store.FindByGlob("src/api/v1/users.go")
		if err != nil {
			t.Fatalf("find by glob: %v", err)
		}
		if len(results) != 1 || results[0].ID != "conv.error-handling" {
			t.Errorf("expected conv.error-handling, got %v", results)
		}

		// Should not match
		results, err = store.FindByGlob("docs/README.md")
		if err != nil {
			t.Fatalf("find by glob: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("expected 0 results for docs path, got %d", len(results))
		}

		// Verify content roundtrip
		gotContent, err := store.GetContent(p)
		if err != nil {
			t.Fatalf("get content: %v", err)
		}
		if gotContent != content {
			t.Errorf("content mismatch: got %q, want %q", gotContent, content)
		}
	})

	t.Run("CreateWithScopes_FindByScope", func(t *testing.T) {
		p := &pearl.Pearl{
			ID: "conv.logging", Name: "logging", Namespace: "conv",
			Type: "convention", Status: pearl.StatusActive,
			Scopes:      []string{"backend", "observability"},
			Description: "Logging conventions",
			CreatedAt:   now, UpdatedAt: now, CreatedBy: "test",
		}
		content := "# Logging\n\nUse structured logging."

		if err := store.Create(p, content); err != nil {
			t.Fatalf("create: %v", err)
		}

		results, err := store.FindByScope("backend")
		if err != nil {
			t.Fatalf("find by scope: %v", err)
		}
		if len(results) != 1 || results[0].ID != "conv.logging" {
			t.Errorf("expected conv.logging, got %v", results)
		}

		results, err = store.FindByScope("nonexistent")
		if err != nil {
			t.Fatalf("find by scope: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("expected 0 results for nonexistent scope, got %d", len(results))
		}
	})

	t.Run("OverlappingGlobsScopes_UnionDedup", func(t *testing.T) {
		p := &pearl.Pearl{
			ID: "conv.testing", Name: "testing", Namespace: "conv",
			Type: "convention", Status: pearl.StatusActive,
			Globs:       []string{"src/api/**/*.go"},
			Scopes:      []string{"backend"},
			Description: "Testing conventions",
			CreatedAt:   now, UpdatedAt: now, CreatedBy: "test",
		}
		content := "# Testing\n\nWrite table-driven tests."

		if err := store.Create(p, content); err != nil {
			t.Fatalf("create: %v", err)
		}

		// Both conv.error-handling and conv.testing match src/api/**/*.go
		globResults, err := store.FindByGlob("src/api/v1/users.go")
		if err != nil {
			t.Fatalf("find by glob: %v", err)
		}
		if len(globResults) != 2 {
			t.Errorf("expected 2 glob matches, got %d", len(globResults))
		}

		// Both conv.logging and conv.testing match backend scope
		scopeResults, err := store.FindByScope("backend")
		if err != nil {
			t.Fatalf("find by scope: %v", err)
		}
		if len(scopeResults) != 2 {
			t.Errorf("expected 2 scope matches, got %d", len(scopeResults))
		}

		// Simulate union dedup as context.go does
		seen := make(map[string]bool)
		var union []*pearl.Pearl
		for _, p := range globResults {
			if !seen[p.ID] {
				seen[p.ID] = true
				union = append(union, p)
			}
		}
		for _, p := range scopeResults {
			if !seen[p.ID] {
				seen[p.ID] = true
				union = append(union, p)
			}
		}
		// conv.error-handling (glob) + conv.testing (both) + conv.logging (scope) = 3
		if len(union) != 3 {
			t.Errorf("expected 3 unique pearls in union, got %d", len(union))
		}
	})

	t.Run("CreateWithInlineContent", func(t *testing.T) {
		p := &pearl.Pearl{
			ID: "decisions.auth", Name: "auth", Namespace: "decisions",
			Type: "brainstorm", Status: pearl.StatusActive,
			Globs:       []string{"src/auth/**"},
			Description: "Auth system redesign",
			CreatedAt:   now, UpdatedAt: now, CreatedBy: "test",
		}
		inlineContent := "# Auth Decision\n\nUse JWT with short-lived tokens.\n\n## Rationale\n\nStateless auth scales better."

		if err := store.Create(p, inlineContent); err != nil {
			t.Fatalf("create: %v", err)
		}

		gotContent, err := store.GetContent(p)
		if err != nil {
			t.Fatalf("get content: %v", err)
		}
		if gotContent != inlineContent {
			t.Errorf("inline content mismatch: got %q", gotContent)
		}

		// Verify JSONL has all 4 pearls
		all, err := store.JSONL().ReadAll()
		if err != nil {
			t.Fatalf("read jsonl: %v", err)
		}
		if len(all) != 4 {
			t.Errorf("expected 4 pearls in JSONL, got %d", len(all))
		}
	})
}

func TestSyncFromJSONL(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pearls-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	jsonlPath := filepath.Join(tmpDir, "pearls.jsonl")
	dbPath := filepath.Join(tmpDir, "pearls.db")

	// Pre-populate JSONL
	j := NewJSONL(jsonlPath)
	now := time.Now()
	p := &pearl.Pearl{
		ID:        "test.pearl",
		Name:      "pearl",
		Namespace: "test",
		Type:      pearl.TypeTable,
		Status:    pearl.StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	j.WriteAll([]*pearl.Pearl{p})

	// Create store and sync from JSONL
	store, err := NewStore(dbPath, jsonlPath, filepath.Join(tmpDir, "content"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	if err := store.SyncFromJSONL(); err != nil {
		t.Fatalf("sync from jsonl: %v", err)
	}

	// Verify database has the pearl
	got, err := store.Get("test.pearl")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil {
		t.Fatal("pearl should exist after sync")
	}
	if got.ID != p.ID {
		t.Errorf("ID = %q, want %q", got.ID, p.ID)
	}
}

func TestRequiredAndPriority(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pearls-reqpri-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "pearls.db")
	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	now := time.Now().Truncate(time.Second)

	// Insert pearls with different required/priority values
	pearls := []*pearl.Pearl{
		{
			ID: "test.low", Name: "low", Namespace: "test",
			Type: pearl.TypeTable, Status: pearl.StatusActive,
			Required: false, Priority: 1,
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "test.high", Name: "high", Namespace: "test",
			Type: pearl.TypeTable, Status: pearl.StatusActive,
			Required: true, Priority: 10,
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "test.medium", Name: "medium", Namespace: "test",
			Type: pearl.TypeTable, Status: pearl.StatusActive,
			Required: true, Priority: 5,
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "test.zero", Name: "zero", Namespace: "test",
			Type: pearl.TypeTable, Status: pearl.StatusActive,
			Required: false, Priority: 0,
			CreatedAt: now, UpdatedAt: now,
		},
	}

	for _, p := range pearls {
		if err := db.Insert(p); err != nil {
			t.Fatalf("insert %s: %v", p.ID, err)
		}
	}

	// Test roundtrip: Get preserves required and priority
	t.Run("Roundtrip", func(t *testing.T) {
		got, err := db.Get("test.high")
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if got == nil {
			t.Fatal("pearl not found")
		}
		if !got.Required {
			t.Error("expected Required to be true")
		}
		if got.Priority != 10 {
			t.Errorf("Priority = %d, want 10", got.Priority)
		}

		got, err = db.Get("test.low")
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if got.Required {
			t.Error("expected Required to be false")
		}
		if got.Priority != 1 {
			t.Errorf("Priority = %d, want 1", got.Priority)
		}
	})

	// Test List with Required filter
	t.Run("ListRequiredFilter", func(t *testing.T) {
		reqTrue := true
		results, err := db.List(ListOptions{Required: &reqTrue})
		if err != nil {
			t.Fatalf("list required=true: %v", err)
		}
		if len(results) != 2 {
			t.Errorf("expected 2 required pearls, got %d", len(results))
		}
		for _, p := range results {
			if !p.Required {
				t.Errorf("pearl %s should be required", p.ID)
			}
		}

		reqFalse := false
		results, err = db.List(ListOptions{Required: &reqFalse})
		if err != nil {
			t.Fatalf("list required=false: %v", err)
		}
		if len(results) != 2 {
			t.Errorf("expected 2 non-required pearls, got %d", len(results))
		}
		for _, p := range results {
			if p.Required {
				t.Errorf("pearl %s should not be required", p.ID)
			}
		}
	})

	// Test List orders by priority DESC
	t.Run("PriorityOrdering", func(t *testing.T) {
		results, err := db.List(ListOptions{})
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if len(results) != 4 {
			t.Fatalf("expected 4 pearls, got %d", len(results))
		}

		// Should be ordered by priority DESC, then namespace, name
		// priority 10 (high), priority 5 (medium), priority 1 (low), priority 0 (zero)
		expectedOrder := []string{"test.high", "test.medium", "test.low", "test.zero"}
		for i, p := range results {
			if p.ID != expectedOrder[i] {
				t.Errorf("position %d: got %s, want %s", i, p.ID, expectedOrder[i])
			}
		}
	})

	// Test Update preserves required and priority
	t.Run("UpdatePreservesFields", func(t *testing.T) {
		p, _ := db.Get("test.low")
		p.Required = true
		p.Priority = 100
		p.UpdatedAt = time.Now().Truncate(time.Second)

		if err := db.Update(p); err != nil {
			t.Fatalf("update: %v", err)
		}

		got, _ := db.Get("test.low")
		if !got.Required {
			t.Error("expected Required to be true after update")
		}
		if got.Priority != 100 {
			t.Errorf("Priority = %d, want 100 after update", got.Priority)
		}
	})

	// Test default values (zero values)
	t.Run("DefaultValues", func(t *testing.T) {
		p := &pearl.Pearl{
			ID: "test.defaults", Name: "defaults", Namespace: "test",
			Type: pearl.TypeTable, Status: pearl.StatusActive,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := db.Insert(p); err != nil {
			t.Fatalf("insert: %v", err)
		}

		got, _ := db.Get("test.defaults")
		if got.Required {
			t.Error("expected Required to default to false")
		}
		if got.Priority != 0 {
			t.Errorf("expected Priority to default to 0, got %d", got.Priority)
		}
	})
}
