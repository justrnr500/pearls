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
					{Name: "id", DataType: "integer", PrimaryKey: true},
					{Name: "name", DataType: "text"},
				},
			},
			{
				Name:   "orders",
				Schema: "public",
				Columns: []Column{
					{Name: "id", DataType: "integer", PrimaryKey: true},
					{Name: "user_id", DataType: "integer"},
				},
				ForeignKeys: []ForeignKey{
					{
						Column:          "user_id",
						ReferencesTable: "users",
						ReferencesCol:   "id",
					},
				},
			},
		},
	}

	results := GeneratePearls("mydb", tables, "DATABASE_URL")

	// Expect 4 pearls: 1 database + 1 schema + 2 tables
	if len(results) != 4 {
		t.Fatalf("expected 4 pearls, got %d", len(results))
	}

	// Database pearl
	db := results[0]
	if db.Pearl.ID != "mydb" {
		t.Errorf("expected db ID 'mydb', got %q", db.Pearl.ID)
	}
	if db.Pearl.Type != pearl.TypeDatabase {
		t.Errorf("expected db type %q, got %q", pearl.TypeDatabase, db.Pearl.Type)
	}
	if db.Pearl.Connection == nil || db.Pearl.Connection.Host != "${DATABASE_URL}" {
		t.Errorf("expected connection host '${DATABASE_URL}', got %v", db.Pearl.Connection)
	}
	if db.Pearl.CreatedBy != "pearls-introspect" {
		t.Errorf("expected CreatedBy 'pearls-introspect', got %q", db.Pearl.CreatedBy)
	}

	// Schema pearl
	schema := results[1]
	if schema.Pearl.ID != "mydb.public" {
		t.Errorf("expected schema ID 'mydb.public', got %q", schema.Pearl.ID)
	}
	if schema.Pearl.Type != pearl.TypeSchema {
		t.Errorf("expected schema type %q, got %q", pearl.TypeSchema, schema.Pearl.Type)
	}
	if schema.Pearl.Parent != "mydb" {
		t.Errorf("expected schema parent 'mydb', got %q", schema.Pearl.Parent)
	}

	// Find users and orders table pearls
	var usersPearl, ordersPearl *GeneratedPearl
	for i := range results {
		if results[i].Pearl.ID == "mydb.public.users" {
			usersPearl = &results[i]
		}
		if results[i].Pearl.ID == "mydb.public.orders" {
			ordersPearl = &results[i]
		}
	}

	if usersPearl == nil {
		t.Fatal("users pearl not found")
	}
	if usersPearl.Pearl.Type != pearl.TypeTable {
		t.Errorf("expected users type %q, got %q", pearl.TypeTable, usersPearl.Pearl.Type)
	}
	if usersPearl.Pearl.Parent != "mydb.public" {
		t.Errorf("expected users parent 'mydb.public', got %q", usersPearl.Pearl.Parent)
	}

	if ordersPearl == nil {
		t.Fatal("orders pearl not found")
	}
	if ordersPearl.Pearl.Type != pearl.TypeTable {
		t.Errorf("expected orders type %q, got %q", pearl.TypeTable, ordersPearl.Pearl.Type)
	}
	if ordersPearl.Pearl.Parent != "mydb.public" {
		t.Errorf("expected orders parent 'mydb.public', got %q", ordersPearl.Pearl.Parent)
	}

	// Orders should reference users
	if len(ordersPearl.Pearl.References) != 1 {
		t.Fatalf("expected 1 reference on orders, got %d", len(ordersPearl.Pearl.References))
	}
	if ordersPearl.Pearl.References[0] != "mydb.public.users" {
		t.Errorf("expected orders reference 'mydb.public.users', got %q", ordersPearl.Pearl.References[0])
	}
}

func TestGeneratePearlsContent(t *testing.T) {
	tables := map[string][]Table{
		"public": {
			{
				Name:   "users",
				Schema: "public",
				Columns: []Column{
					{Name: "id", DataType: "integer", PrimaryKey: true},
				},
			},
		},
	}

	results := GeneratePearls("mydb", tables, "DATABASE_URL")

	// The table pearl should have non-empty content
	for _, r := range results {
		if r.Pearl.Type == pearl.TypeTable {
			if r.GeneratedContent == "" {
				t.Errorf("expected non-empty GeneratedContent for table pearl %q", r.Pearl.ID)
			}
		}
	}
}
