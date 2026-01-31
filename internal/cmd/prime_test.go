package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/justrnr500/pearls/internal/pearl"
	"github.com/justrnr500/pearls/internal/storage"
)

func setupPrimeTestStore(t *testing.T) (*storage.Store, string) {
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

func createTestPearl(t *testing.T, store *storage.Store, id, ns, name string, typ pearl.AssetType) {
	t.Helper()
	now := time.Now()
	p := &pearl.Pearl{
		ID: id, Name: name, Namespace: ns,
		Type: typ, Status: pearl.StatusActive,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := store.Create(p, "# "+name); err != nil {
		t.Fatalf("create pearl %s: %v", id, err)
	}
}

func TestPrimeOutput_EmptyCatalog(t *testing.T) {
	store, _ := setupPrimeTestStore(t)
	defer store.Close()

	var buf bytes.Buffer
	err := writePrimeOutput(&buf, store, "")
	if err != nil {
		t.Fatalf("writePrimeOutput: %v", err)
	}

	out := buf.String()

	// Empty catalog should mention creating pearls
	if !strings.Contains(out, "pl create") {
		t.Error("empty catalog output should mention pl create")
	}

	// Should contain discovery triggers
	if !strings.Contains(out, "convention") {
		t.Error("output should contain discovery triggers mentioning conventions")
	}

	// Should NOT contain a catalog summary with counts
	if strings.Contains(out, "table:") || strings.Contains(out, "Your catalog") {
		t.Error("empty catalog should not show catalog summary")
	}
}

func TestPrimeOutput_SmallCatalog(t *testing.T) {
	store, _ := setupPrimeTestStore(t)
	defer store.Close()

	// Create 5 pearls of mixed types
	createTestPearl(t, store, "db.users", "db", "users", pearl.TypeTable)
	createTestPearl(t, store, "db.orders", "db", "orders", pearl.TypeTable)
	createTestPearl(t, store, "api.stripe", "api", "stripe", pearl.TypeAPI)
	createTestPearl(t, store, "db.products", "db", "products", pearl.TypeTable)
	createTestPearl(t, store, "api.auth", "api", "auth", pearl.TypeEndpoint)

	var buf bytes.Buffer
	err := writePrimeOutput(&buf, store, "")
	if err != nil {
		t.Fatalf("writePrimeOutput: %v", err)
	}

	out := buf.String()

	// Should show catalog summary
	if !strings.Contains(out, "5 pearls") {
		t.Errorf("small catalog should show pearl count, got:\n%s", out)
	}

	// Should break down by type
	if !strings.Contains(out, "table") {
		t.Error("should show type breakdown including 'table'")
	}

	// Should contain discovery triggers
	if !strings.Contains(out, "convention") {
		t.Error("output should contain discovery triggers")
	}

	// Should contain quick reference
	if !strings.Contains(out, "pl search") {
		t.Error("output should contain quick reference with pl search")
	}
}

func TestPrimeOutput_LargeCatalog(t *testing.T) {
	store, _ := setupPrimeTestStore(t)
	defer store.Close()

	// Create 25 pearls with scopes
	for i := 0; i < 15; i++ {
		now := time.Now()
		p := &pearl.Pearl{
			ID: "db.table" + string(rune('a'+i)), Name: "table" + string(rune('a'+i)), Namespace: "db",
			Type: pearl.TypeTable, Status: pearl.StatusActive,
			Scopes:    []string{"backend"},
			CreatedAt: now, UpdatedAt: now,
		}
		store.Create(p, "# Table")
	}
	for i := 0; i < 10; i++ {
		now := time.Now()
		p := &pearl.Pearl{
			ID: "api.svc" + string(rune('a'+i)), Name: "svc" + string(rune('a'+i)), Namespace: "api",
			Type: pearl.TypeAPI, Status: pearl.StatusActive,
			Scopes:    []string{"frontend"},
			CreatedAt: now, UpdatedAt: now,
		}
		store.Create(p, "# API")
	}

	var buf bytes.Buffer
	err := writePrimeOutput(&buf, store, "")
	if err != nil {
		t.Fatalf("writePrimeOutput: %v", err)
	}

	out := buf.String()

	// Should show count
	if !strings.Contains(out, "25 pearls") {
		t.Errorf("large catalog should show pearl count, got:\n%s", out)
	}

	// Should mention searching
	if !strings.Contains(out, "pl search") {
		t.Error("large catalog should remind agent to search")
	}

	// Should mention scopes
	if !strings.Contains(out, "backend") || !strings.Contains(out, "frontend") {
		t.Error("large catalog should list scopes")
	}
}

func TestPrimeOutput_OverrideFile(t *testing.T) {
	store, tmpDir := setupPrimeTestStore(t)
	defer store.Close()

	// Create a .pearls directory with PRIME.md override
	pearlsDir := filepath.Join(tmpDir, ".pearls")
	os.MkdirAll(pearlsDir, 0755)
	override := "# Custom Prime\n\nThis is a custom override.\n"
	os.WriteFile(filepath.Join(pearlsDir, "PRIME.md"), []byte(override), 0644)

	var buf bytes.Buffer
	err := writePrimeOutput(&buf, store, filepath.Join(pearlsDir, "PRIME.md"))
	if err != nil {
		t.Fatalf("writePrimeOutput: %v", err)
	}

	out := buf.String()

	if out != override {
		t.Errorf("expected override content, got:\n%s", out)
	}
}

func TestPrimeOutput_OverrideFileMissing(t *testing.T) {
	store, _ := setupPrimeTestStore(t)
	defer store.Close()

	// Non-existent override path should fall through to generated output
	var buf bytes.Buffer
	err := writePrimeOutput(&buf, store, "/nonexistent/PRIME.md")
	if err != nil {
		t.Fatalf("writePrimeOutput: %v", err)
	}

	out := buf.String()

	// Should get generated output, not empty
	if !strings.Contains(out, "pl create") {
		t.Error("missing override should fall through to generated output")
	}
}
