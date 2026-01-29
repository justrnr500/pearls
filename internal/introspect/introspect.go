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
	Column           string
	ReferencesTable  string
	ReferencesCol    string
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

// SplitColumns splits a comma-separated column list.
func SplitColumns(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, len(parts))
	for i, p := range parts {
		result[i] = strings.TrimSpace(p)
	}
	return result
}
