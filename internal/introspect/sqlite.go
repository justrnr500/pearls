package introspect

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteIntrospector implements Introspector for SQLite databases.
type SQLiteIntrospector struct {
	db *sql.DB
}

// Connect opens a read-only connection to the SQLite database at connStr.
func (s *SQLiteIntrospector) Connect(connStr string) error {
	// Ensure read-only mode.
	dsn := connStr
	if strings.Contains(dsn, "?") {
		dsn += "&mode=ro"
	} else {
		dsn += "?mode=ro"
	}
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return fmt.Errorf("sqlite open: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("sqlite ping: %w", err)
	}
	s.db = db
	return nil
}

// Schemas returns the list of schemas. SQLite only has "main".
func (s *SQLiteIntrospector) Schemas() ([]string, error) {
	return []string{"main"}, nil
}

// Tables returns all user tables in the given schema.
func (s *SQLiteIntrospector) Tables(schema string) ([]Table, error) {
	rows, err := s.db.Query(
		"SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name",
	)
	if err != nil {
		return nil, fmt.Errorf("sqlite tables: %w", err)
	}
	defer rows.Close()

	var tables []Table
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("sqlite scan table: %w", err)
		}
		cols, err := s.columns(name)
		if err != nil {
			return nil, err
		}
		fks, err := s.foreignKeys(name)
		if err != nil {
			return nil, err
		}
		idxs, err := s.indexes(name)
		if err != nil {
			return nil, err
		}
		tables = append(tables, Table{
			Name:        name,
			Schema:      schema,
			Columns:     cols,
			ForeignKeys: fks,
			Indexes:     idxs,
		})
	}
	return tables, rows.Err()
}

// Close closes the database connection.
func (s *SQLiteIntrospector) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// columns returns column metadata for the given table using PRAGMA table_info.
func (s *SQLiteIntrospector) columns(table string) ([]Column, error) {
	rows, err := s.db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return nil, fmt.Errorf("sqlite columns(%s): %w", table, err)
	}
	defer rows.Close()

	var cols []Column
	for rows.Next() {
		var (
			cid      int
			name     string
			dataType string
			notnull  int
			dflt     sql.NullString
			pk       int
		)
		if err := rows.Scan(&cid, &name, &dataType, &notnull, &dflt, &pk); err != nil {
			return nil, fmt.Errorf("sqlite scan column(%s): %w", table, err)
		}
		col := Column{
			Name:       name,
			DataType:   dataType,
			Nullable:   notnull == 0,
			PrimaryKey: pk > 0,
		}
		if dflt.Valid {
			col.Default = dflt.String
		}
		cols = append(cols, col)
	}
	return cols, rows.Err()
}

// foreignKeys returns foreign key metadata for the given table using PRAGMA foreign_key_list.
func (s *SQLiteIntrospector) foreignKeys(table string) ([]ForeignKey, error) {
	rows, err := s.db.Query(fmt.Sprintf("PRAGMA foreign_key_list(%s)", table))
	if err != nil {
		return nil, fmt.Errorf("sqlite fks(%s): %w", table, err)
	}
	defer rows.Close()

	var fks []ForeignKey
	for rows.Next() {
		var (
			id       int
			seq      int
			refTable string
			from     string
			to       string
			onUpdate string
			onDelete string
			match    string
		)
		if err := rows.Scan(&id, &seq, &refTable, &from, &to, &onUpdate, &onDelete, &match); err != nil {
			return nil, fmt.Errorf("sqlite scan fk(%s): %w", table, err)
		}
		fks = append(fks, ForeignKey{
			Column:          from,
			ReferencesTable: refTable,
			ReferencesCol:   to,
		})
	}
	return fks, rows.Err()
}

// indexes returns index metadata for the given table using PRAGMA index_list and PRAGMA index_info.
func (s *SQLiteIntrospector) indexes(table string) ([]Index, error) {
	rows, err := s.db.Query(fmt.Sprintf("PRAGMA index_list(%s)", table))
	if err != nil {
		return nil, fmt.Errorf("sqlite indexes(%s): %w", table, err)
	}
	defer rows.Close()

	var idxs []Index
	for rows.Next() {
		var (
			seq     int
			name    string
			unique  int
			origin  string
			partial int
		)
		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			return nil, fmt.Errorf("sqlite scan index(%s): %w", table, err)
		}
		cols, err := s.indexColumns(name)
		if err != nil {
			return nil, err
		}
		idxs = append(idxs, Index{
			Name:    name,
			Columns: cols,
			Unique:  unique == 1,
		})
	}
	return idxs, rows.Err()
}

// indexColumns returns the column names for an index using PRAGMA index_info.
func (s *SQLiteIntrospector) indexColumns(indexName string) ([]string, error) {
	rows, err := s.db.Query(fmt.Sprintf("PRAGMA index_info(%s)", indexName))
	if err != nil {
		return nil, fmt.Errorf("sqlite index_info(%s): %w", indexName, err)
	}
	defer rows.Close()

	var cols []string
	for rows.Next() {
		var (
			seqno int
			cid   int
			name  string
		)
		if err := rows.Scan(&seqno, &cid, &name); err != nil {
			return nil, fmt.Errorf("sqlite scan index_info(%s): %w", indexName, err)
		}
		cols = append(cols, name)
	}
	return cols, rows.Err()
}
