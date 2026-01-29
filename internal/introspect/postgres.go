package introspect

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

// PostgresIntrospector implements the Introspector interface for PostgreSQL databases.
type PostgresIntrospector struct {
	db *sql.DB
}

// Connect opens a connection to the PostgreSQL database and verifies it with a ping.
func (p *PostgresIntrospector) Connect(connStr string) error {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("postgres open: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("postgres ping: %w", err)
	}
	p.db = db
	return nil
}

// Schemas returns all user schemas, excluding internal PostgreSQL schemas.
func (p *PostgresIntrospector) Schemas() ([]string, error) {
	const query = `
		SELECT schema_name
		FROM information_schema.schemata
		WHERE schema_name NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		ORDER BY schema_name`

	rows, err := p.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("schemas query: %w", err)
	}
	defer rows.Close()

	var schemas []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, fmt.Errorf("schemas scan: %w", err)
		}
		schemas = append(schemas, s)
	}
	return schemas, rows.Err()
}

// Tables returns all base tables in the given schema, fully populated with
// columns, foreign keys, and indexes.
func (p *PostgresIntrospector) Tables(schema string) ([]Table, error) {
	const query = `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = $1
		  AND table_type = 'BASE TABLE'
		ORDER BY table_name`

	rows, err := p.db.Query(query, schema)
	if err != nil {
		return nil, fmt.Errorf("tables query: %w", err)
	}
	defer rows.Close()

	var tables []Table
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("tables scan: %w", err)
		}
		tables = append(tables, Table{
			Name:   name,
			Schema: schema,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Populate columns, foreign keys, and indexes for each table.
	for i := range tables {
		cols, err := p.columns(schema, tables[i].Name)
		if err != nil {
			return nil, fmt.Errorf("columns for %s.%s: %w", schema, tables[i].Name, err)
		}
		tables[i].Columns = cols

		fks, err := p.foreignKeys(schema, tables[i].Name)
		if err != nil {
			return nil, fmt.Errorf("foreign keys for %s.%s: %w", schema, tables[i].Name, err)
		}
		tables[i].ForeignKeys = fks

		idxs, err := p.indexes(schema, tables[i].Name)
		if err != nil {
			return nil, fmt.Errorf("indexes for %s.%s: %w", schema, tables[i].Name, err)
		}
		tables[i].Indexes = idxs
	}

	return tables, nil
}

// Close closes the underlying database connection.
func (p *PostgresIntrospector) Close() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

// columns retrieves column metadata for a table, including primary key detection.
func (p *PostgresIntrospector) columns(schema, table string) ([]Column, error) {
	const query = `
		SELECT
			c.column_name,
			c.data_type,
			c.is_nullable,
			COALESCE(c.column_default, ''),
			CASE WHEN tc.constraint_type = 'PRIMARY KEY' THEN true ELSE false END AS is_pk
		FROM information_schema.columns c
		LEFT JOIN information_schema.key_column_usage kcu
			ON  c.table_schema = kcu.table_schema
			AND c.table_name   = kcu.table_name
			AND c.column_name  = kcu.column_name
		LEFT JOIN information_schema.table_constraints tc
			ON  kcu.constraint_schema = tc.constraint_schema
			AND kcu.constraint_name   = tc.constraint_name
			AND tc.constraint_type    = 'PRIMARY KEY'
		WHERE c.table_schema = $1
		  AND c.table_name   = $2
		ORDER BY c.ordinal_position`

	rows, err := p.db.Query(query, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []Column
	for rows.Next() {
		var col Column
		var nullable string
		if err := rows.Scan(&col.Name, &col.DataType, &nullable, &col.Default, &col.PrimaryKey); err != nil {
			return nil, err
		}
		col.Nullable = nullable == "YES"
		cols = append(cols, col)
	}
	return cols, rows.Err()
}

// foreignKeys retrieves foreign key relationships for a table.
func (p *PostgresIntrospector) foreignKeys(schema, table string) ([]ForeignKey, error) {
	const query = `
		SELECT
			kcu.column_name,
			ccu.table_name  AS ref_table,
			ccu.column_name AS ref_column,
			ccu.table_schema AS ref_schema
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON  tc.constraint_schema = kcu.constraint_schema
			AND tc.constraint_name   = kcu.constraint_name
		JOIN information_schema.constraint_column_usage ccu
			ON  tc.constraint_schema = ccu.constraint_schema
			AND tc.constraint_name   = ccu.constraint_name
		WHERE tc.constraint_type = 'FOREIGN KEY'
		  AND tc.table_schema    = $1
		  AND tc.table_name      = $2
		ORDER BY kcu.column_name`

	rows, err := p.db.Query(query, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fks []ForeignKey
	for rows.Next() {
		var fk ForeignKey
		if err := rows.Scan(&fk.Column, &fk.ReferencesTable, &fk.ReferencesCol, &fk.ReferencesSchema); err != nil {
			return nil, err
		}
		fks = append(fks, fk)
	}
	return fks, rows.Err()
}

// indexes retrieves index metadata for a table using PostgreSQL system catalogs.
func (p *PostgresIntrospector) indexes(schema, table string) ([]Index, error) {
	const query = `
		SELECT
			ic.relname                     AS index_name,
			ix.indisunique                 AS is_unique,
			string_agg(a.attname, ',' ORDER BY array_position(ix.indkey, a.attnum)) AS columns
		FROM pg_index ix
		JOIN pg_class tc  ON tc.oid = ix.indrelid
		JOIN pg_class ic  ON ic.oid = ix.indexrelid
		JOIN pg_namespace n ON n.oid = tc.relnamespace
		JOIN pg_attribute a ON a.attrelid = tc.oid AND a.attnum = ANY(ix.indkey)
		WHERE n.nspname = $1
		  AND tc.relname = $2
		GROUP BY ic.relname, ix.indisunique
		ORDER BY ic.relname`

	rows, err := p.db.Query(query, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var idxs []Index
	for rows.Next() {
		var idx Index
		var colStr string
		if err := rows.Scan(&idx.Name, &idx.Unique, &colStr); err != nil {
			return nil, err
		}
		idx.Columns = SplitColumns(colStr)
		idxs = append(idxs, idx)
	}
	return idxs, rows.Err()
}
