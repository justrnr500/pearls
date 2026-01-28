package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/justrnr500/pearls/internal/pearl"
)

func TestDBCRUD(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "pearls-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "pearls.db")
	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	now := time.Now().Truncate(time.Second)

	// Test Insert
	p := &pearl.Pearl{
		ID:          "db.postgres.users",
		Name:        "users",
		Namespace:   "db.postgres",
		Type:        pearl.TypeTable,
		Tags:        []string{"pii", "core"},
		Description: "User accounts",
		ContentPath: "db/postgres/users.md",
		ContentHash: "abc123",
		Status:      pearl.StatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
		CreatedBy:   "test",
	}

	if err := db.Insert(p); err != nil {
		t.Fatalf("insert: %v", err)
	}

	// Test Get
	got, err := db.Get("db.postgres.users")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil {
		t.Fatal("get returned nil")
	}
	if got.ID != p.ID {
		t.Errorf("ID = %q, want %q", got.ID, p.ID)
	}
	if got.Type != p.Type {
		t.Errorf("Type = %q, want %q", got.Type, p.Type)
	}
	if len(got.Tags) != 2 {
		t.Errorf("len(Tags) = %d, want 2", len(got.Tags))
	}

	// Test Update
	p.Description = "Updated description"
	p.UpdatedAt = time.Now().Truncate(time.Second)
	if err := db.Update(p); err != nil {
		t.Fatalf("update: %v", err)
	}

	got, _ = db.Get("db.postgres.users")
	if got.Description != "Updated description" {
		t.Errorf("Description = %q, want %q", got.Description, "Updated description")
	}

	// Test List
	p2 := &pearl.Pearl{
		ID:        "db.postgres.orders",
		Name:      "orders",
		Namespace: "db.postgres",
		Type:      pearl.TypeTable,
		Status:    pearl.StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	db.Insert(p2)

	list, err := db.List(ListOptions{Namespace: "db.postgres"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("list length = %d, want 2", len(list))
	}

	// Test Delete
	if err := db.Delete("db.postgres.orders"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	got, _ = db.Get("db.postgres.orders")
	if got != nil {
		t.Error("pearl should be deleted")
	}

	// Test Count
	count, err := db.Count()
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}

func TestJSONL(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pearls-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	jsonlPath := filepath.Join(tmpDir, "pearls.jsonl")
	j := NewJSONL(jsonlPath)

	now := time.Now()

	// Test Append and ReadAll
	p1 := &pearl.Pearl{
		ID:        "test.pearl1",
		Name:      "pearl1",
		Type:      pearl.TypeTable,
		Status:    pearl.StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	p2 := &pearl.Pearl{
		ID:        "test.pearl2",
		Name:      "pearl2",
		Type:      pearl.TypeAPI,
		Status:    pearl.StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := j.Append(p1); err != nil {
		t.Fatalf("append p1: %v", err)
	}
	if err := j.Append(p2); err != nil {
		t.Fatalf("append p2: %v", err)
	}

	pearls, err := j.ReadAll()
	if err != nil {
		t.Fatalf("read all: %v", err)
	}
	if len(pearls) != 2 {
		t.Errorf("len = %d, want 2", len(pearls))
	}

	// Test WriteAll (overwrite)
	if err := j.WriteAll([]*pearl.Pearl{p1}); err != nil {
		t.Fatalf("write all: %v", err)
	}

	pearls, _ = j.ReadAll()
	if len(pearls) != 1 {
		t.Errorf("len after overwrite = %d, want 1", len(pearls))
	}
}

func TestContent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pearls-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	c := NewContent(tmpDir)

	// Test PathForPearl
	path := c.PathForPearl("db.postgres", "users")
	if path != "db/postgres/users.md" {
		t.Errorf("PathForPearl = %q, want %q", path, "db/postgres/users.md")
	}

	// Test Write and Read
	content := "# Test\n\nThis is a test."
	if err := c.Write(path, content); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := c.Read(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got != content {
		t.Errorf("content = %q, want %q", got, content)
	}

	// Test Exists
	if !c.Exists(path) {
		t.Error("file should exist")
	}

	// Test Hash
	hash, err := c.Hash(path)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if hash == "" {
		t.Error("hash should not be empty")
	}

	// Verify hash is consistent
	if HashString(content) != hash {
		t.Error("hash mismatch")
	}

	// Test Delete
	if err := c.Delete(path); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if c.Exists(path) {
		t.Error("file should be deleted")
	}
}

func TestStore(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pearls-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(
		filepath.Join(tmpDir, "pearls.db"),
		filepath.Join(tmpDir, "pearls.jsonl"),
		filepath.Join(tmpDir, "content"),
	)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	now := time.Now()

	// Test Create
	p := &pearl.Pearl{
		ID:          "db.postgres.users",
		Name:        "users",
		Namespace:   "db.postgres",
		Type:        pearl.TypeTable,
		Description: "User accounts",
		Status:      pearl.StatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	content := "# users\n\nUser accounts table."
	if err := store.Create(p, content); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Verify in database
	got, err := store.Get("db.postgres.users")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil {
		t.Fatal("pearl not found in db")
	}

	// Verify content
	gotContent, err := store.GetContent(got)
	if err != nil {
		t.Fatalf("get content: %v", err)
	}
	if gotContent != content {
		t.Errorf("content = %q, want %q", gotContent, content)
	}

	// Verify content hash was set
	if got.ContentHash == "" {
		t.Error("content hash should be set")
	}

	// Verify JSONL was written
	pearls, _ := store.JSONL().ReadAll()
	if len(pearls) != 1 {
		t.Errorf("jsonl length = %d, want 1", len(pearls))
	}

	// Test Update
	newContent := "# users\n\nUpdated content."
	got.Description = "Updated"
	got.UpdatedAt = time.Now()
	if err := store.Update(got, &newContent); err != nil {
		t.Fatalf("update: %v", err)
	}

	gotContent, _ = store.GetContent(got)
	if gotContent != newContent {
		t.Errorf("updated content = %q, want %q", gotContent, newContent)
	}

	// Test Delete
	if err := store.Delete("db.postgres.users"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	got, _ = store.Get("db.postgres.users")
	if got != nil {
		t.Error("pearl should be deleted")
	}
}

func TestSyncFromJSONL(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pearls-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	jsonlPath := filepath.Join(tmpDir, "pearls.jsonl")
	dbPath := filepath.Join(tmpDir, "pearls.db")

	// Pre-populate JSONL
	j := NewJSONL(jsonlPath)
	now := time.Now()
	p := &pearl.Pearl{
		ID:        "test.pearl",
		Name:      "pearl",
		Namespace: "test",
		Type:      pearl.TypeTable,
		Status:    pearl.StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	j.WriteAll([]*pearl.Pearl{p})

	// Create store and sync from JSONL
	store, err := NewStore(dbPath, jsonlPath, filepath.Join(tmpDir, "content"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	if err := store.SyncFromJSONL(); err != nil {
		t.Fatalf("sync from jsonl: %v", err)
	}

	// Verify database has the pearl
	got, err := store.Get("test.pearl")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil {
		t.Fatal("pearl should exist after sync")
	}
	if got.ID != p.ID {
		t.Errorf("ID = %q, want %q", got.ID, p.ID)
	}
}
