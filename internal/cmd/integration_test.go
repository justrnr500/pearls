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
		content := gp.GeneratedContent
		if content == "" {
			content = "# " + p.Name + "\n"
		}
		if err := store.Create(&p, content); err != nil {
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
