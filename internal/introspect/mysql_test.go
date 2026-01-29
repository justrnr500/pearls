package introspect

import (
	"os"
	"testing"
)

func TestMySQLIntrospector(t *testing.T) {
	connStr := os.Getenv("PEARLS_TEST_MYSQL_URL")
	if connStr == "" {
		t.Skip("PEARLS_TEST_MYSQL_URL not set, skipping MySQL integration test")
	}

	m := &MySQLIntrospector{}

	if err := m.Connect(connStr); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer m.Close()

	schemas, err := m.Schemas()
	if err != nil {
		t.Fatalf("Schemas: %v", err)
	}
	t.Logf("Schemas: %v", schemas)

	if len(schemas) == 0 {
		t.Log("No user schemas found, skipping table listing")
		return
	}

	for _, schema := range schemas {
		tables, err := m.Tables(schema)
		if err != nil {
			t.Fatalf("Tables(%q): %v", schema, err)
		}
		t.Logf("Schema %q: %d tables", schema, len(tables))
		for _, tbl := range tables {
			t.Logf("  Table %s: %d columns, %d FKs, %d indexes",
				tbl.Name, len(tbl.Columns), len(tbl.ForeignKeys), len(tbl.Indexes))
		}
	}
}
