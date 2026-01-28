package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
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

// mockEmbedder is a simple embedder for testing
type mockEmbedder struct {
	embedCount int
}

func (m *mockEmbedder) Embed(text string) ([]float32, error) {
	m.embedCount++
	// Generate a simple embedding based on text length
	emb := make([]float32, 384)
	for i := range emb {
		emb[i] = float32(len(text)%100) * 0.01 * float32(i+1) * 0.001
	}
	return emb, nil
}

func (m *mockEmbedder) Close() error {
	return nil
}

func TestStoreWithEmbedder(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pearls-embedder-test-*")
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

	// Set up mock embedder
	embedder := &mockEmbedder{}
	store.SetEmbedder(embedder)

	if !store.HasEmbedder() {
		t.Error("expected HasEmbedder to be true")
	}

	now := time.Now()

	// Test Create generates embedding
	t.Run("CreateGeneratesEmbedding", func(t *testing.T) {
		p := &pearl.Pearl{
			ID:          "test.pearl1",
			Name:        "pearl1",
			Namespace:   "test",
			Type:        pearl.TypeTable,
			Description: "Test pearl",
			Status:      pearl.StatusActive,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		content := "# Test Pearl\n\nThis is test content."

		if err := store.Create(p, content); err != nil {
			t.Fatalf("create: %v", err)
		}

		if embedder.embedCount != 1 {
			t.Errorf("embedCount = %d, want 1", embedder.embedCount)
		}

		// Verify embedding was stored
		has, err := store.DB().HasEmbedding("test.pearl1")
		if err != nil {
			t.Fatalf("has embedding: %v", err)
		}
		if !has {
			t.Error("expected pearl to have embedding after create")
		}
	})

	// Test Update with content change regenerates embedding
	t.Run("UpdateRegeneratesEmbedding", func(t *testing.T) {
		p, _ := store.Get("test.pearl1")
		newContent := "# Updated Pearl\n\nNew content here."
		p.UpdatedAt = time.Now()

		countBefore := embedder.embedCount
		if err := store.Update(p, &newContent); err != nil {
			t.Fatalf("update: %v", err)
		}

		if embedder.embedCount != countBefore+1 {
			t.Errorf("embedCount = %d, want %d", embedder.embedCount, countBefore+1)
		}
	})

	// Test Update without content change doesn't regenerate
	t.Run("UpdateNoContentNoRegenerate", func(t *testing.T) {
		p, _ := store.Get("test.pearl1")
		p.Description = "Updated description"
		p.UpdatedAt = time.Now()

		countBefore := embedder.embedCount
		if err := store.Update(p, nil); err != nil {
			t.Fatalf("update: %v", err)
		}

		if embedder.embedCount != countBefore {
			t.Errorf("embedding should not regenerate without content change")
		}
	})

	// Test Delete removes embedding
	t.Run("DeleteRemovesEmbedding", func(t *testing.T) {
		if err := store.Delete("test.pearl1"); err != nil {
			t.Fatalf("delete: %v", err)
		}

		has, _ := store.DB().HasEmbedding("test.pearl1")
		if has {
			t.Error("expected embedding to be deleted")
		}
	})

	// Test SearchSemantic
	t.Run("SearchSemantic", func(t *testing.T) {
		// Create a few pearls
		for i := 1; i <= 3; i++ {
			p := &pearl.Pearl{
				ID:          fmt.Sprintf("search.pearl%d", i),
				Name:        fmt.Sprintf("pearl%d", i),
				Namespace:   "search",
				Type:        pearl.TypeTable,
				Description: fmt.Sprintf("Description %d", i),
				Status:      pearl.StatusActive,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			store.Create(p, fmt.Sprintf("Content for pearl %d", i))
		}

		results, err := store.SearchSemantic("test query", 10)
		if err != nil {
			t.Fatalf("search semantic: %v", err)
		}

		if len(results) != 3 {
			t.Errorf("expected 3 results, got %d", len(results))
		}
	})
}

func TestFindReferencingPearls(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pearls-refs-test-*")
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

	now := time.Now()

	// Create pearls with references
	pearls := []*pearl.Pearl{
		{
			ID: "db.postgres.users", Name: "users", Namespace: "db.postgres",
			Type: pearl.TypeTable, Status: pearl.StatusActive,
			References: []string{}, // No references
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "db.postgres.orders", Name: "orders", Namespace: "db.postgres",
			Type: pearl.TypeTable, Status: pearl.StatusActive,
			References: []string{"db.postgres.users"}, // References users
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "db.postgres.payments", Name: "payments", Namespace: "db.postgres",
			Type: pearl.TypeTable, Status: pearl.StatusActive,
			References: []string{"db.postgres.users", "db.postgres.orders"}, // References both
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "api.stripe.customers", Name: "customers", Namespace: "api.stripe",
			Type: pearl.TypeAPI, Status: pearl.StatusActive,
			References: []string{"db.postgres.users"}, // Also references users
			CreatedAt: now, UpdatedAt: now,
		},
	}

	for _, p := range pearls {
		if err := db.Insert(p); err != nil {
			t.Fatalf("insert %s: %v", p.ID, err)
		}
	}

	// Test: find what references "db.postgres.users"
	refs, err := db.FindReferencingPearls("db.postgres.users")
	if err != nil {
		t.Fatalf("find referencing pearls: %v", err)
	}

	// Should find: orders, payments, customers (3 pearls reference users)
	if len(refs) != 3 {
		t.Errorf("expected 3 pearls referencing users, got %d: %v", len(refs), refs)
	}

	// Test: find what references "db.postgres.orders"
	refs, err = db.FindReferencingPearls("db.postgres.orders")
	if err != nil {
		t.Fatalf("find referencing pearls: %v", err)
	}

	// Should find: payments (1 pearl references orders)
	if len(refs) != 1 {
		t.Errorf("expected 1 pearl referencing orders, got %d: %v", len(refs), refs)
	}
	if len(refs) > 0 && refs[0] != "db.postgres.payments" {
		t.Errorf("expected payments to reference orders, got %s", refs[0])
	}

	// Test: find what references "api.stripe.customers" (nothing)
	refs, err = db.FindReferencingPearls("api.stripe.customers")
	if err != nil {
		t.Fatalf("find referencing pearls: %v", err)
	}

	if len(refs) != 0 {
		t.Errorf("expected 0 pearls referencing customers, got %d: %v", len(refs), refs)
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

func TestVectorSearch(t *testing.T) {
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

	// Verify sqlite-vec is loaded
	var vecVersion string
	err = db.db.QueryRow("SELECT vec_version()").Scan(&vecVersion)
	if err != nil {
		t.Fatalf("vec_version query failed: %v", err)
	}
	if vecVersion == "" {
		t.Fatal("vec_version should not be empty")
	}
	t.Logf("sqlite-vec version: %s", vecVersion)

	// Verify pearl_embeddings table exists
	var tableName string
	err = db.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='pearl_embeddings'").Scan(&tableName)
	if err != nil {
		t.Fatalf("pearl_embeddings table not found: %v", err)
	}

	// Insert test vectors
	vec1 := []float32{0.1, 0.2, 0.3, 0.4} // Simplified 4-dim for testing
	vec2 := []float32{0.9, 0.8, 0.7, 0.6}
	vec3 := []float32{0.11, 0.21, 0.31, 0.41} // Similar to vec1

	// Create a simple 4-dim test table (not the full 384-dim production table)
	_, err = db.db.Exec("CREATE VIRTUAL TABLE IF NOT EXISTS test_embeddings USING vec0(embedding FLOAT[4])")
	if err != nil {
		t.Fatalf("create test_embeddings: %v", err)
	}

	// Insert vectors
	for i, vec := range [][]float32{vec1, vec2, vec3} {
		serialized, err := sqlite_vec.SerializeFloat32(vec)
		if err != nil {
			t.Fatalf("serialize vec%d: %v", i+1, err)
		}
		_, err = db.db.Exec("INSERT INTO test_embeddings(rowid, embedding) VALUES (?, ?)", i+1, serialized)
		if err != nil {
			t.Fatalf("insert vec%d: %v", i+1, err)
		}
	}

	// Query for vectors similar to vec1
	queryVec, _ := sqlite_vec.SerializeFloat32(vec1)
	rows, err := db.db.Query(`
		SELECT rowid, distance
		FROM test_embeddings
		WHERE embedding MATCH ?
		ORDER BY distance
		LIMIT 3
	`, queryVec)
	if err != nil {
		t.Fatalf("similarity query: %v", err)
	}
	defer rows.Close()

	var results []struct {
		RowID    int
		Distance float32
	}
	for rows.Next() {
		var r struct {
			RowID    int
			Distance float32
		}
		if err := rows.Scan(&r.RowID, &r.Distance); err != nil {
			t.Fatalf("scan result: %v", err)
		}
		results = append(results, r)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// vec1 (rowid=1) should be most similar to itself (distance 0)
	if results[0].RowID != 1 {
		t.Errorf("expected rowid 1 first (exact match), got %d", results[0].RowID)
	}
	if results[0].Distance != 0 {
		t.Errorf("expected distance 0 for exact match, got %f", results[0].Distance)
	}

	// vec3 (rowid=3) should be second (similar to vec1)
	if results[1].RowID != 3 {
		t.Errorf("expected rowid 3 second (similar), got %d", results[1].RowID)
	}

	// vec2 (rowid=2) should be last (most different)
	if results[2].RowID != 2 {
		t.Errorf("expected rowid 2 last (different), got %d", results[2].RowID)
	}

	t.Logf("Vector search results: %+v", results)
}

func TestVectorOperations(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pearls-vector-ops-test-*")
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

	now := time.Now()

	// Create test pearls first (embeddings reference pearls via rowid)
	pearls := []*pearl.Pearl{
		{ID: "test.pearl1", Name: "pearl1", Type: pearl.TypeTable, Status: pearl.StatusActive, CreatedAt: now, UpdatedAt: now},
		{ID: "test.pearl2", Name: "pearl2", Type: pearl.TypeTable, Status: pearl.StatusActive, CreatedAt: now, UpdatedAt: now},
		{ID: "test.pearl3", Name: "pearl3", Type: pearl.TypeTable, Status: pearl.StatusActive, CreatedAt: now, UpdatedAt: now},
	}

	for _, p := range pearls {
		if err := db.Insert(p); err != nil {
			t.Fatalf("insert pearl %s: %v", p.ID, err)
		}
	}

	// Test embeddings (384-dim like real model, but with simple values)
	makeEmbedding := func(base float32) []float32 {
		emb := make([]float32, 384)
		for i := range emb {
			emb[i] = base + float32(i)*0.001
		}
		return emb
	}

	emb1 := makeEmbedding(0.1)
	emb2 := makeEmbedding(0.9) // Different
	emb3 := makeEmbedding(0.11) // Similar to emb1

	// Test InsertEmbedding
	t.Run("InsertEmbedding", func(t *testing.T) {
		if err := db.InsertEmbedding("test.pearl1", emb1); err != nil {
			t.Fatalf("insert embedding 1: %v", err)
		}
		if err := db.InsertEmbedding("test.pearl2", emb2); err != nil {
			t.Fatalf("insert embedding 2: %v", err)
		}
		if err := db.InsertEmbedding("test.pearl3", emb3); err != nil {
			t.Fatalf("insert embedding 3: %v", err)
		}

		count, err := db.EmbeddingCount()
		if err != nil {
			t.Fatalf("count: %v", err)
		}
		if count != 3 {
			t.Errorf("count = %d, want 3", count)
		}
	})

	// Test HasEmbedding
	t.Run("HasEmbedding", func(t *testing.T) {
		has, err := db.HasEmbedding("test.pearl1")
		if err != nil {
			t.Fatalf("has embedding: %v", err)
		}
		if !has {
			t.Error("expected pearl1 to have embedding")
		}

		has, err = db.HasEmbedding("nonexistent")
		if err != nil {
			t.Fatalf("has embedding nonexistent: %v", err)
		}
		if has {
			t.Error("expected nonexistent to not have embedding")
		}
	})

	// Test SearchSemantic
	t.Run("SearchSemantic", func(t *testing.T) {
		results, err := db.SearchSemantic(emb1, 10)
		if err != nil {
			t.Fatalf("search: %v", err)
		}

		if len(results) != 3 {
			t.Fatalf("expected 3 results, got %d", len(results))
		}

		// pearl1 should be first (exact match)
		if results[0].ID != "test.pearl1" {
			t.Errorf("expected pearl1 first, got %s", results[0].ID)
		}
		if results[0].Distance != 0 {
			t.Errorf("expected distance 0 for exact match, got %f", results[0].Distance)
		}

		// pearl3 should be second (similar to pearl1)
		if results[1].ID != "test.pearl3" {
			t.Errorf("expected pearl3 second, got %s", results[1].ID)
		}

		// pearl2 should be last (most different)
		if results[2].ID != "test.pearl2" {
			t.Errorf("expected pearl2 last, got %s", results[2].ID)
		}

		t.Logf("Search results: %+v", results)
	})

	// Test UpdateEmbedding
	t.Run("UpdateEmbedding", func(t *testing.T) {
		newEmb := makeEmbedding(0.5) // Now different from both
		if err := db.UpdateEmbedding("test.pearl1", newEmb); err != nil {
			t.Fatalf("update embedding: %v", err)
		}

		// Search with original emb1 - pearl3 should now be closest
		results, err := db.SearchSemantic(emb1, 10)
		if err != nil {
			t.Fatalf("search after update: %v", err)
		}

		if results[0].ID != "test.pearl3" {
			t.Errorf("expected pearl3 first after update, got %s", results[0].ID)
		}
	})

	// Test DeleteEmbedding
	t.Run("DeleteEmbedding", func(t *testing.T) {
		if err := db.DeleteEmbedding("test.pearl2"); err != nil {
			t.Fatalf("delete embedding: %v", err)
		}

		has, _ := db.HasEmbedding("test.pearl2")
		if has {
			t.Error("pearl2 should not have embedding after delete")
		}

		count, _ := db.EmbeddingCount()
		if count != 2 {
			t.Errorf("count after delete = %d, want 2", count)
		}
	})

	// Test ClearEmbeddings
	t.Run("ClearEmbeddings", func(t *testing.T) {
		if err := db.ClearEmbeddings(); err != nil {
			t.Fatalf("clear: %v", err)
		}

		count, _ := db.EmbeddingCount()
		if count != 0 {
			t.Errorf("count after clear = %d, want 0", count)
		}
	})
}
