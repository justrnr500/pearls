// Package storage provides persistence for pearls data.
package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/justrnr500/pearls/internal/pearl"
)

const schema = `
CREATE TABLE IF NOT EXISTS pearls (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	namespace TEXT NOT NULL DEFAULT '',
	type TEXT NOT NULL,
	tags TEXT NOT NULL DEFAULT '[]',
	description TEXT NOT NULL DEFAULT '',
	content_path TEXT NOT NULL DEFAULT '',
	content_hash TEXT NOT NULL DEFAULT '',
	refs TEXT NOT NULL DEFAULT '[]',
	parent TEXT NOT NULL DEFAULT '',
	connection TEXT,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	created_by TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT 'active'
);

CREATE INDEX IF NOT EXISTS idx_pearls_namespace ON pearls(namespace);
CREATE INDEX IF NOT EXISTS idx_pearls_type ON pearls(type);
CREATE INDEX IF NOT EXISTS idx_pearls_status ON pearls(status);
`

// DB wraps the SQLite database connection.
type DB struct {
	db   *sql.DB
	path string
}

// OpenDB opens or creates a SQLite database at the given path.
func OpenDB(path string) (*DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Initialize schema
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize schema: %w", err)
	}

	return &DB{db: db, path: path}, nil
}

// Close closes the database connection.
func (d *DB) Close() error {
	return d.db.Close()
}

// Path returns the database file path.
func (d *DB) Path() string {
	return d.path
}

// Insert adds a new pearl to the database.
func (d *DB) Insert(p *pearl.Pearl) error {
	tags, err := json.Marshal(p.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}

	refs, err := json.Marshal(p.References)
	if err != nil {
		return fmt.Errorf("marshal refs: %w", err)
	}

	var connJSON []byte
	if p.Connection != nil {
		connJSON, err = json.Marshal(p.Connection)
		if err != nil {
			return fmt.Errorf("marshal connection: %w", err)
		}
	}

	_, err = d.db.Exec(`
		INSERT INTO pearls (id, name, namespace, type, tags, description, content_path, content_hash, refs, parent, connection, created_at, updated_at, created_by, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		p.ID, p.Name, p.Namespace, p.Type, tags, p.Description,
		p.ContentPath, p.ContentHash, refs, p.Parent, connJSON,
		p.CreatedAt.Format(time.RFC3339), p.UpdatedAt.Format(time.RFC3339),
		p.CreatedBy, p.Status,
	)
	if err != nil {
		return fmt.Errorf("insert pearl: %w", err)
	}

	return nil
}

// Update updates an existing pearl in the database.
func (d *DB) Update(p *pearl.Pearl) error {
	tags, err := json.Marshal(p.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}

	refs, err := json.Marshal(p.References)
	if err != nil {
		return fmt.Errorf("marshal refs: %w", err)
	}

	var connJSON []byte
	if p.Connection != nil {
		connJSON, err = json.Marshal(p.Connection)
		if err != nil {
			return fmt.Errorf("marshal connection: %w", err)
		}
	}

	result, err := d.db.Exec(`
		UPDATE pearls SET
			name = ?, namespace = ?, type = ?, tags = ?, description = ?,
			content_path = ?, content_hash = ?, refs = ?, parent = ?,
			connection = ?, updated_at = ?, created_by = ?, status = ?
		WHERE id = ?
	`,
		p.Name, p.Namespace, p.Type, tags, p.Description,
		p.ContentPath, p.ContentHash, refs, p.Parent, connJSON,
		p.UpdatedAt.Format(time.RFC3339), p.CreatedBy, p.Status,
		p.ID,
	)
	if err != nil {
		return fmt.Errorf("update pearl: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("pearl not found: %s", p.ID)
	}

	return nil
}

// Delete removes a pearl from the database.
func (d *DB) Delete(id string) error {
	result, err := d.db.Exec("DELETE FROM pearls WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete pearl: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("pearl not found: %s", id)
	}

	return nil
}

// Get retrieves a pearl by ID.
func (d *DB) Get(id string) (*pearl.Pearl, error) {
	row := d.db.QueryRow(`
		SELECT id, name, namespace, type, tags, description, content_path, content_hash, refs, parent, connection, created_at, updated_at, created_by, status
		FROM pearls WHERE id = ?
	`, id)

	return scanPearl(row)
}

// List retrieves all pearls matching the given filters.
func (d *DB) List(opts ListOptions) ([]*pearl.Pearl, error) {
	query := "SELECT id, name, namespace, type, tags, description, content_path, content_hash, refs, parent, connection, created_at, updated_at, created_by, status FROM pearls WHERE 1=1"
	args := []interface{}{}

	if opts.Namespace != "" {
		query += " AND (namespace = ? OR namespace LIKE ?)"
		args = append(args, opts.Namespace, opts.Namespace+".%")
	}
	if opts.Type != "" {
		query += " AND type = ?"
		args = append(args, opts.Type)
	}
	if opts.Status != "" {
		query += " AND status = ?"
		args = append(args, opts.Status)
	}
	if opts.Tag != "" {
		query += " AND tags LIKE ?"
		args = append(args, "%\""+opts.Tag+"\"%")
	}

	query += " ORDER BY namespace, name"

	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query pearls: %w", err)
	}
	defer rows.Close()

	var pearls []*pearl.Pearl
	for rows.Next() {
		p, err := scanPearlRows(rows)
		if err != nil {
			return nil, err
		}
		pearls = append(pearls, p)
	}

	return pearls, rows.Err()
}

// ListOptions specifies filters for listing pearls.
type ListOptions struct {
	Namespace string
	Type      string
	Status    string
	Tag       string
	Limit     int
}

// Search performs a keyword search on pearls using LIKE.
func (d *DB) Search(query string, limit int) ([]*pearl.Pearl, error) {
	if limit <= 0 {
		limit = 50
	}

	// Simple LIKE-based search across searchable fields
	pattern := "%" + query + "%"
	rows, err := d.db.Query(`
		SELECT id, name, namespace, type, tags, description, content_path, content_hash, refs, parent, connection, created_at, updated_at, created_by, status
		FROM pearls
		WHERE id LIKE ? OR name LIKE ? OR namespace LIKE ? OR description LIKE ? OR tags LIKE ?
		ORDER BY namespace, name
		LIMIT ?
	`, pattern, pattern, pattern, pattern, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("search pearls: %w", err)
	}
	defer rows.Close()

	var pearls []*pearl.Pearl
	for rows.Next() {
		p, err := scanPearlRows(rows)
		if err != nil {
			return nil, err
		}
		pearls = append(pearls, p)
	}

	return pearls, rows.Err()
}

// All retrieves all pearls from the database.
func (d *DB) All() ([]*pearl.Pearl, error) {
	return d.List(ListOptions{})
}

// Count returns the number of pearls in the database.
func (d *DB) Count() (int, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM pearls").Scan(&count)
	return count, err
}

// scanner interface for both sql.Row and sql.Rows
type scanner interface {
	Scan(dest ...interface{}) error
}

func scanPearl(row *sql.Row) (*pearl.Pearl, error) {
	var p pearl.Pearl
	var tags, refs, connJSON []byte
	var createdAt, updatedAt string

	err := row.Scan(
		&p.ID, &p.Name, &p.Namespace, &p.Type, &tags, &p.Description,
		&p.ContentPath, &p.ContentHash, &refs, &p.Parent, &connJSON,
		&createdAt, &updatedAt, &p.CreatedBy, &p.Status,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan pearl: %w", err)
	}

	if err := json.Unmarshal(tags, &p.Tags); err != nil {
		return nil, fmt.Errorf("unmarshal tags: %w", err)
	}
	if err := json.Unmarshal(refs, &p.References); err != nil {
		return nil, fmt.Errorf("unmarshal refs: %w", err)
	}
	if len(connJSON) > 0 {
		p.Connection = &pearl.ConnectionInfo{}
		if err := json.Unmarshal(connJSON, p.Connection); err != nil {
			return nil, fmt.Errorf("unmarshal connection: %w", err)
		}
	}

	p.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	p.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return &p, nil
}

func scanPearlRows(rows *sql.Rows) (*pearl.Pearl, error) {
	var p pearl.Pearl
	var tags, refs, connJSON []byte
	var createdAt, updatedAt string

	err := rows.Scan(
		&p.ID, &p.Name, &p.Namespace, &p.Type, &tags, &p.Description,
		&p.ContentPath, &p.ContentHash, &refs, &p.Parent, &connJSON,
		&createdAt, &updatedAt, &p.CreatedBy, &p.Status,
	)
	if err != nil {
		return nil, fmt.Errorf("scan pearl: %w", err)
	}

	if err := json.Unmarshal(tags, &p.Tags); err != nil {
		return nil, fmt.Errorf("unmarshal tags: %w", err)
	}
	if err := json.Unmarshal(refs, &p.References); err != nil {
		return nil, fmt.Errorf("unmarshal refs: %w", err)
	}
	if len(connJSON) > 0 {
		p.Connection = &pearl.ConnectionInfo{}
		if err := json.Unmarshal(connJSON, p.Connection); err != nil {
			return nil, fmt.Errorf("unmarshal connection: %w", err)
		}
	}

	p.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	p.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return &p, nil
}
