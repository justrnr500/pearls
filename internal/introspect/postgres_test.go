package introspect

import (
	"os"
	"testing"
)

func TestPostgresIntrospector(t *testing.T) {
	connStr := os.Getenv("PEARLS_TEST_POSTGRES_URL")
	if connStr == "" {
		t.Skip("PEARLS_TEST_POSTGRES_URL not set; skipping integration test")
	}

	p := &PostgresIntrospector{}

	// Connect
	if err := p.Connect(connStr); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer p.Close()

	// List schemas
	schemas, err := p.Schemas()
	if err != nil {
		t.Fatalf("Schemas: %v", err)
	}
	if len(schemas) == 0 {
		t.Fatal("expected at least one schema")
	}
	t.Logf("schemas: %v", schemas)

	// List tables from first schema
	tables, err := p.Tables(schemas[0])
	if err != nil {
		t.Fatalf("Tables(%s): %v", schemas[0], err)
	}
	t.Logf("tables in %s: %d", schemas[0], len(tables))

	for _, tbl := range tables {
		if len(tbl.Columns) == 0 {
			t.Errorf("table %s.%s has no columns", tbl.Schema, tbl.Name)
		}
		t.Logf("  %s.%s: %d cols, %d fks, %d indexes",
			tbl.Schema, tbl.Name,
			len(tbl.Columns), len(tbl.ForeignKeys), len(tbl.Indexes))
	}
}
