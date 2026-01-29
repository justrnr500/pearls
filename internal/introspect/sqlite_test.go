package introspect

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestSQLiteIntrospector(t *testing.T) {
	// Create a temp SQLite database.
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open temp db: %v", err)
	}

	// Create schema.
	stmts := []string{
		`CREATE TABLE users (
			id    INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL UNIQUE,
			name  TEXT
		)`,
		`CREATE TABLE orders (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id    INTEGER NOT NULL REFERENCES users(id),
			total      REAL NOT NULL DEFAULT 0.0,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX idx_orders_user ON orders(user_id)`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("exec %q: %v", stmt, err)
		}
	}
	db.Close()

	// Introspect the database.
	intro := &SQLiteIntrospector{}
	if err := intro.Connect(dbPath); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer intro.Close()

	// Test Schemas.
	schemas, err := intro.Schemas()
	if err != nil {
		t.Fatalf("Schemas: %v", err)
	}
	if len(schemas) != 1 || schemas[0] != "main" {
		t.Fatalf("expected [main], got %v", schemas)
	}

	// Test Tables.
	tables, err := intro.Tables("main")
	if err != nil {
		t.Fatalf("Tables: %v", err)
	}
	if len(tables) != 2 {
		t.Fatalf("expected 2 tables, got %d", len(tables))
	}

	// Tables are ordered by name: orders, users.
	orders := tables[0]
	users := tables[1]

	if orders.Name != "orders" {
		t.Fatalf("expected first table 'orders', got %q", orders.Name)
	}
	if users.Name != "users" {
		t.Fatalf("expected second table 'users', got %q", users.Name)
	}

	// Verify users columns.
	if len(users.Columns) != 3 {
		t.Fatalf("users: expected 3 columns, got %d", len(users.Columns))
	}
	// id column
	if users.Columns[0].Name != "id" || !users.Columns[0].PrimaryKey {
		t.Errorf("users.id: expected PK, got %+v", users.Columns[0])
	}
	// email column
	if users.Columns[1].Name != "email" || users.Columns[1].Nullable {
		t.Errorf("users.email: expected NOT NULL, got %+v", users.Columns[1])
	}
	// name column
	if users.Columns[2].Name != "name" || !users.Columns[2].Nullable {
		t.Errorf("users.name: expected nullable, got %+v", users.Columns[2])
	}

	// Verify orders columns.
	if len(orders.Columns) != 4 {
		t.Fatalf("orders: expected 4 columns, got %d", len(orders.Columns))
	}

	// Verify orders foreign keys.
	if len(orders.ForeignKeys) != 1 {
		t.Fatalf("orders: expected 1 FK, got %d", len(orders.ForeignKeys))
	}
	fk := orders.ForeignKeys[0]
	if fk.Column != "user_id" || fk.ReferencesTable != "users" || fk.ReferencesCol != "id" {
		t.Errorf("orders FK: expected user_id->users(id), got %+v", fk)
	}

	// Verify orders indexes include idx_orders_user.
	found := false
	for _, idx := range orders.Indexes {
		if idx.Name == "idx_orders_user" {
			found = true
			if idx.Unique {
				t.Errorf("idx_orders_user should not be unique")
			}
			if len(idx.Columns) != 1 || idx.Columns[0] != "user_id" {
				t.Errorf("idx_orders_user: expected [user_id], got %v", idx.Columns)
			}
		}
	}
	if !found {
		t.Errorf("orders: idx_orders_user not found in indexes: %+v", orders.Indexes)
	}

	// Verify users has a unique index on email (auto-created by UNIQUE constraint).
	foundEmailIdx := false
	for _, idx := range users.Indexes {
		if idx.Unique && len(idx.Columns) == 1 && idx.Columns[0] == "email" {
			foundEmailIdx = true
		}
	}
	if !foundEmailIdx {
		t.Errorf("users: expected unique index on email, indexes: %+v", users.Indexes)
	}

	// Clean up: verify Close works.
	if err := intro.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}

	// Remove the temp file (TempDir handles this, but be explicit).
	os.Remove(dbPath)
}
