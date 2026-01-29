package introspect

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

// MySQLIntrospector implements Introspector for MySQL databases.
type MySQLIntrospector struct {
	db *sql.DB
}

// Connect opens a connection to a MySQL database.
// It strips the "mysql://" prefix if present, since go-sql-driver expects DSN format.
func (m *MySQLIntrospector) Connect(connStr string) error {
	dsn := strings.TrimPrefix(connStr, "mysql://")
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("mysql open: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("mysql ping: %w", err)
	}
	m.db = db
	return nil
}

// Schemas returns all user schemas, excluding system schemas.
func (m *MySQLIntrospector) Schemas() ([]string, error) {
	rows, err := m.db.Query(`
		SELECT SCHEMA_NAME
		FROM information_schema.SCHEMATA
		WHERE SCHEMA_NAME NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys')
		ORDER BY SCHEMA_NAME`)
	if err != nil {
		return nil, fmt.Errorf("mysql schemas: %w", err)
	}
	defer rows.Close()

	var schemas []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, fmt.Errorf("mysql schemas scan: %w", err)
		}
		schemas = append(schemas, s)
	}
	return schemas, rows.Err()
}

// Tables returns all base tables in the given schema, with columns, foreign keys, and indexes.
func (m *MySQLIntrospector) Tables(schema string) ([]Table, error) {
	rows, err := m.db.Query(`
		SELECT TABLE_NAME
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = ? AND TABLE_TYPE = 'BASE TABLE'
		ORDER BY TABLE_NAME`, schema)
	if err != nil {
		return nil, fmt.Errorf("mysql tables: %w", err)
	}
	defer rows.Close()

	var tables []Table
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("mysql tables scan: %w", err)
		}
		tables = append(tables, Table{Name: name, Schema: schema})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range tables {
		cols, err := m.columns(schema, tables[i].Name)
		if err != nil {
			return nil, err
		}
		tables[i].Columns = cols

		fks, err := m.foreignKeys(schema, tables[i].Name)
		if err != nil {
			return nil, err
		}
		tables[i].ForeignKeys = fks

		idxs, err := m.indexes(schema, tables[i].Name)
		if err != nil {
			return nil, err
		}
		tables[i].Indexes = idxs
	}

	return tables, nil
}

// Close closes the database connection.
func (m *MySQLIntrospector) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

// columns retrieves column metadata for the given schema and table.
func (m *MySQLIntrospector) columns(schema, table string) ([]Column, error) {
	rows, err := m.db.Query(`
		SELECT COLUMN_NAME, COLUMN_TYPE, IS_NULLABLE, COALESCE(COLUMN_DEFAULT, ''), COLUMN_KEY
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION`, schema, table)
	if err != nil {
		return nil, fmt.Errorf("mysql columns: %w", err)
	}
	defer rows.Close()

	var cols []Column
	for rows.Next() {
		var c Column
		var nullable, columnKey string
		if err := rows.Scan(&c.Name, &c.DataType, &nullable, &c.Default, &columnKey); err != nil {
			return nil, fmt.Errorf("mysql columns scan: %w", err)
		}
		c.Nullable = nullable == "YES"
		if columnKey == "PRI" {
			c.PrimaryKey = true
		}
		if columnKey == "UNI" {
			c.Constraints = "UNIQUE"
		}
		cols = append(cols, c)
	}
	return cols, rows.Err()
}

// foreignKeys retrieves foreign key relationships for the given schema and table.
func (m *MySQLIntrospector) foreignKeys(schema, table string) ([]ForeignKey, error) {
	rows, err := m.db.Query(`
		SELECT COLUMN_NAME, REFERENCED_TABLE_SCHEMA, REFERENCED_TABLE_NAME, REFERENCED_COLUMN_NAME
		FROM information_schema.KEY_COLUMN_USAGE
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? AND REFERENCED_TABLE_NAME IS NOT NULL
		ORDER BY COLUMN_NAME`, schema, table)
	if err != nil {
		return nil, fmt.Errorf("mysql foreign keys: %w", err)
	}
	defer rows.Close()

	var fks []ForeignKey
	for rows.Next() {
		var fk ForeignKey
		if err := rows.Scan(&fk.Column, &fk.ReferencesSchema, &fk.ReferencesTable, &fk.ReferencesCol); err != nil {
			return nil, fmt.Errorf("mysql foreign keys scan: %w", err)
		}
		fks = append(fks, fk)
	}
	return fks, rows.Err()
}

// indexes retrieves index metadata for the given schema and table.
func (m *MySQLIntrospector) indexes(schema, table string) ([]Index, error) {
	rows, err := m.db.Query(`
		SELECT INDEX_NAME, GROUP_CONCAT(COLUMN_NAME ORDER BY SEQ_IN_INDEX) AS columns, NON_UNIQUE
		FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		GROUP BY INDEX_NAME, NON_UNIQUE
		ORDER BY INDEX_NAME`, schema, table)
	if err != nil {
		return nil, fmt.Errorf("mysql indexes: %w", err)
	}
	defer rows.Close()

	var idxs []Index
	for rows.Next() {
		var idx Index
		var colStr string
		var nonUnique int
		if err := rows.Scan(&idx.Name, &colStr, &nonUnique); err != nil {
			return nil, fmt.Errorf("mysql indexes scan: %w", err)
		}
		idx.Columns = SplitColumns(colStr)
		idx.Unique = nonUnique == 0
		idxs = append(idxs, idx)
	}
	return idxs, rows.Err()
}
