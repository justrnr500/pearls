package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/justrnr500/pearls/internal/pearl"
	"github.com/justrnr500/pearls/internal/storage"
)

func setupDoctorTestStore(t *testing.T) (*storage.Store, string) {
	t.Helper()
	tmpDir := t.TempDir()

	store, err := storage.NewStore(
		filepath.Join(tmpDir, "pearls.db"),
		filepath.Join(tmpDir, "pearls.jsonl"),
		filepath.Join(tmpDir, "content"),
	)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	return store, tmpDir
}

func TestCheckJSONLSync_AllGood(t *testing.T) {
	store, _ := setupDoctorTestStore(t)
	defer store.Close()

	now := time.Now()
	p := &pearl.Pearl{
		ID: "test.a", Name: "a", Namespace: "test",
		Type: pearl.TypeTable, Status: pearl.StatusActive,
		CreatedAt: now, UpdatedAt: now,
	}
	store.Create(p, "# A")

	result := checkJSONLSync(store)
	if !result.Passed {
		t.Errorf("expected pass, got issues: %v", result.Issues)
	}
}

func TestCheckJSONLSync_Mismatch(t *testing.T) {
	store, _ := setupDoctorTestStore(t)
	defer store.Close()

	now := time.Now()
	// Insert directly into DB only (bypassing JSONL)
	p := &pearl.Pearl{
		ID: "test.orphan", Name: "orphan", Namespace: "test",
		Type: pearl.TypeTable, Status: pearl.StatusActive,
		CreatedAt: now, UpdatedAt: now,
	}
	store.DB().Insert(p)

	result := checkJSONLSync(store)
	if result.Passed {
		t.Error("expected fail due to count mismatch")
	}
}

func TestCheckOrphanedContent(t *testing.T) {
	store, tmpDir := setupDoctorTestStore(t)
	defer store.Close()

	contentDir := filepath.Join(tmpDir, "content", "orphan")
	os.MkdirAll(contentDir, 0755)
	os.WriteFile(filepath.Join(contentDir, "stale.md"), []byte("# Orphan"), 0644)

	result := checkOrphanedContent(store)
	if result.Passed {
		t.Error("expected fail due to orphaned content")
	}
	if len(result.Issues) != 1 {
		t.Errorf("expected 1 orphan, got %d", len(result.Issues))
	}
}

func TestCheckOrphanedContent_Clean(t *testing.T) {
	store, _ := setupDoctorTestStore(t)
	defer store.Close()

	now := time.Now()
	p := &pearl.Pearl{
		ID: "test.a", Name: "a", Namespace: "test",
		Type: pearl.TypeTable, Status: pearl.StatusActive,
		CreatedAt: now, UpdatedAt: now,
	}
	store.Create(p, "# A")

	result := checkOrphanedContent(store)
	if !result.Passed {
		t.Errorf("expected pass, got issues: %v", result.Issues)
	}
}

func TestCheckMissingContent(t *testing.T) {
	store, _ := setupDoctorTestStore(t)
	defer store.Close()

	now := time.Now()
	p := &pearl.Pearl{
		ID: "test.missing", Name: "missing", Namespace: "test",
		Type: pearl.TypeTable, Status: pearl.StatusActive,
		ContentPath: "test/missing.md",
		CreatedAt:   now, UpdatedAt: now,
	}
	store.DB().Insert(p)
	store.JSONL().Append(p)

	result := checkMissingContent(store)
	if result.Passed {
		t.Error("expected fail due to missing content")
	}
}

func TestCheckMissingContent_Clean(t *testing.T) {
	store, _ := setupDoctorTestStore(t)
	defer store.Close()

	now := time.Now()
	p := &pearl.Pearl{
		ID: "test.a", Name: "a", Namespace: "test",
		Type: pearl.TypeTable, Status: pearl.StatusActive,
		CreatedAt: now, UpdatedAt: now,
	}
	store.Create(p, "# A")

	result := checkMissingContent(store)
	if !result.Passed {
		t.Errorf("expected pass, got issues: %v", result.Issues)
	}
}

func TestCheckBrokenReferences(t *testing.T) {
	store, _ := setupDoctorTestStore(t)
	defer store.Close()

	now := time.Now()
	p := &pearl.Pearl{
		ID: "test.ref", Name: "ref", Namespace: "test",
		Type: pearl.TypeTable, Status: pearl.StatusActive,
		References: []string{"nonexistent.pearl"},
		CreatedAt:  now, UpdatedAt: now,
	}
	store.Create(p, "# Ref")

	result := checkBrokenReferences(store)
	if result.Passed {
		t.Error("expected fail due to broken reference")
	}
}

func TestCheckBrokenReferences_Clean(t *testing.T) {
	store, _ := setupDoctorTestStore(t)
	defer store.Close()

	now := time.Now()
	a := &pearl.Pearl{
		ID: "test.a", Name: "a", Namespace: "test",
		Type: pearl.TypeTable, Status: pearl.StatusActive,
		CreatedAt: now, UpdatedAt: now,
	}
	b := &pearl.Pearl{
		ID: "test.b", Name: "b", Namespace: "test",
		Type: pearl.TypeTable, Status: pearl.StatusActive,
		References: []string{"test.a"},
		CreatedAt:  now, UpdatedAt: now,
	}
	store.Create(a, "# A")
	store.Create(b, "# B")

	result := checkBrokenReferences(store)
	if !result.Passed {
		t.Errorf("expected pass, got issues: %v", result.Issues)
	}
}

func TestCheckConfigValidity(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	os.WriteFile(configPath, []byte("project:\n  name: test\n"), 0644)

	result := checkConfigValidity(configPath)
	if !result.Passed {
		t.Errorf("expected pass, got issues: %v", result.Issues)
	}
}

func TestCheckConfigValidity_BadYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	os.WriteFile(configPath, []byte("{{invalid yaml"), 0644)

	result := checkConfigValidity(configPath)
	if result.Passed {
		t.Error("expected fail for invalid YAML")
	}
}
