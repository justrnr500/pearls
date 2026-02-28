package storage

import (
	"fmt"

	"github.com/justrnr500/pearls/internal/pearl"
)

// Store provides a unified interface to pearl storage.
// It syncs between SQLite (fast queries) and JSONL (git-tracked source of truth).
type Store struct {
	db      *DB
	jsonl   *JSONL
	content *Content
}

// NewStore creates a new store with the given paths.
func NewStore(dbPath, jsonlPath, contentDir string) (*Store, error) {
	db, err := OpenDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	return &Store{
		db:      db,
		jsonl:   NewJSONL(jsonlPath),
		content: NewContent(contentDir),
	}, nil
}

// Close closes the store.
func (s *Store) Close() error {
	return s.db.Close()
}

// DB returns the underlying database.
func (s *Store) DB() *DB {
	return s.db
}

// Content returns the content manager.
func (s *Store) Content() *Content {
	return s.content
}

// JSONL returns the JSONL handler.
func (s *Store) JSONL() *JSONL {
	return s.jsonl
}

// Create creates a new pearl with content.
func (s *Store) Create(p *pearl.Pearl, content string) error {
	// Generate content path if not set
	if p.ContentPath == "" {
		p.ContentPath = s.content.PathForPearl(p.Namespace, p.Name)
	}

	// Write content file
	if content != "" {
		if err := s.content.Write(p.ContentPath, content); err != nil {
			return fmt.Errorf("write content: %w", err)
		}
		p.ContentHash = HashString(content)
	}

	// Insert into database
	if err := s.db.Insert(p); err != nil {
		// Rollback content file on failure
		s.content.Delete(p.ContentPath)
		return fmt.Errorf("insert pearl: %w", err)
	}

	// Append to JSONL
	if err := s.jsonl.Append(p); err != nil {
		// Note: DB insert succeeded, JSONL can be rebuilt from DB
		return fmt.Errorf("append to jsonl: %w", err)
	}

	return nil
}

// Get retrieves a pearl by ID.
func (s *Store) Get(id string) (*pearl.Pearl, error) {
	return s.db.Get(id)
}

// GetContent retrieves a pearl's markdown content.
func (s *Store) GetContent(p *pearl.Pearl) (string, error) {
	if p.ContentPath == "" {
		return "", nil
	}
	return s.content.Read(p.ContentPath)
}

// Update updates a pearl and optionally its content.
func (s *Store) Update(p *pearl.Pearl, content *string) error {
	// Update content if provided
	if content != nil && p.ContentPath != "" {
		if err := s.content.Write(p.ContentPath, *content); err != nil {
			return fmt.Errorf("write content: %w", err)
		}
		p.ContentHash = HashString(*content)
	}

	// Update in database
	if err := s.db.Update(p); err != nil {
		return fmt.Errorf("update pearl: %w", err)
	}

	// Rewrite JSONL (full rebuild for updates)
	if err := s.syncToJSONL(); err != nil {
		return fmt.Errorf("sync to jsonl: %w", err)
	}

	return nil
}

// Delete removes a pearl and its content.
func (s *Store) Delete(id string) error {
	// Get pearl first to find content path
	p, err := s.db.Get(id)
	if err != nil {
		return fmt.Errorf("get pearl: %w", err)
	}
	if p == nil {
		return fmt.Errorf("pearl not found: %s", id)
	}

	// Delete from database
	if err := s.db.Delete(id); err != nil {
		return fmt.Errorf("delete pearl: %w", err)
	}

	// Delete content file
	if p.ContentPath != "" {
		s.content.Delete(p.ContentPath)
	}

	// Rewrite JSONL
	if err := s.syncToJSONL(); err != nil {
		return fmt.Errorf("sync to jsonl: %w", err)
	}

	return nil
}

// List retrieves pearls matching the given options.
func (s *Store) List(opts ListOptions) ([]*pearl.Pearl, error) {
	return s.db.List(opts)
}

// Search performs a full-text search.
func (s *Store) Search(query string, limit int) ([]*pearl.Pearl, error) {
	return s.db.Search(query, limit)
}

// FindByScope returns all pearls that belong to the given scope.
func (s *Store) FindByScope(scope string) ([]*pearl.Pearl, error) {
	return s.db.FindByScope(scope)
}

// FindByGlob returns all pearls whose glob patterns match the given file path.
func (s *Store) FindByGlob(path string) ([]*pearl.Pearl, error) {
	return s.db.FindByGlob(path)
}

// SyncFromJSONL rebuilds the database from the JSONL file.
// This is the "JSONL is source of truth" operation.
func (s *Store) SyncFromJSONL() error {
	pearls, err := s.jsonl.ReadAll()
	if err != nil {
		return fmt.Errorf("read jsonl: %w", err)
	}

	// Clear existing database
	if _, err := s.db.db.Exec("DELETE FROM pearls"); err != nil {
		return fmt.Errorf("clear database: %w", err)
	}

	// Insert all pearls
	for _, p := range pearls {
		if err := s.db.Insert(p); err != nil {
			return fmt.Errorf("insert pearl %s: %w", p.ID, err)
		}
	}

	return nil
}

// syncToJSONL writes the database contents to the JSONL file.
func (s *Store) syncToJSONL() error {
	pearls, err := s.db.All()
	if err != nil {
		return fmt.Errorf("get all pearls: %w", err)
	}

	return s.jsonl.WriteAll(pearls)
}

// SyncToJSONL exports the database to JSONL (public version).
func (s *Store) SyncToJSONL() error {
	return s.syncToJSONL()
}

// RefreshContentHashes updates content hashes for all pearls.
func (s *Store) RefreshContentHashes() error {
	pearls, err := s.db.All()
	if err != nil {
		return fmt.Errorf("get all pearls: %w", err)
	}

	for _, p := range pearls {
		if p.ContentPath == "" {
			continue
		}

		hash, err := s.content.Hash(p.ContentPath)
		if err != nil {
			continue // Skip missing files
		}

		if hash != p.ContentHash {
			p.ContentHash = hash
			if err := s.db.Update(p); err != nil {
				return fmt.Errorf("update pearl %s: %w", p.ID, err)
			}
		}
	}

	return s.syncToJSONL()
}

