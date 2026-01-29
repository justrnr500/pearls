package introspect

import (
	"strings"
	"testing"
)

func TestSchemaTypes(t *testing.T) {
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

	for _, want := range []string{"users", "bigint", "varchar(255)", "org_id", "organizations", "users_pkey", "PRIMARY KEY", "# users", "## Columns", "## Foreign Keys", "## Indexes"} {
		if !strings.Contains(content, want) {
			t.Errorf("content missing %q", want)
		}
	}
}

func TestGenerateTableContent_NoFKsOrIndexes(t *testing.T) {
	tbl := Table{
		Name:   "simple",
		Schema: "public",
		Columns: []Column{
			{Name: "id", DataType: "integer", PrimaryKey: true},
		},
	}

	content := GenerateTableContent(tbl, "db.pg")
	if strings.Contains(content, "## Foreign Keys") {
		t.Error("should not have FK section with no FKs")
	}
	if strings.Contains(content, "## Indexes") {
		t.Error("should not have index section with no indexes")
	}
}

func TestDefaultEnvVar(t *testing.T) {
	tests := []struct {
		dbType string
		want   string
	}{
		{"postgres", "PEARLS_POSTGRES_URL"},
		{"mysql", "PEARLS_MYSQL_URL"},
		{"sqlite", "PEARLS_SQLITE_PATH"},
		{"Postgres", "PEARLS_POSTGRES_URL"},
		{"unknown", ""},
	}
	for _, tt := range tests {
		got := DefaultEnvVar(tt.dbType)
		if got != tt.want {
			t.Errorf("DefaultEnvVar(%q) = %q, want %q", tt.dbType, got, tt.want)
		}
	}
}
