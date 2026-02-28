package cmd

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/justrnr500/pearls/internal/pearl"
	"github.com/justrnr500/pearls/internal/storage"
)

func setupClutchTestStore(t *testing.T) *storage.Store {
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
	return store
}

func createNonRequiredPearl(t *testing.T, store *storage.Store, id, ns, name string, typ pearl.AssetType) {
	t.Helper()
	now := time.Now()
	p := &pearl.Pearl{
		ID: id, Name: name, Namespace: ns,
		Type: typ, Status: pearl.StatusActive,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := store.Create(p, "# "+name); err != nil {
		t.Fatalf("create pearl %s: %v", id, err)
	}
}

func createRequiredPearl(t *testing.T, store *storage.Store, id, ns, name string, typ pearl.AssetType, priority int) {
	t.Helper()
	now := time.Now()
	p := &pearl.Pearl{
		ID: id, Name: name, Namespace: ns,
		Type: typ, Status: pearl.StatusActive,
		Required:  true,
		Priority:  priority,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := store.Create(p, "# "+name+"\n\nContent for "+id+"\n"); err != nil {
		t.Fatalf("create pearl %s: %v", id, err)
	}
}

func TestClutch_Empty(t *testing.T) {
	store := setupClutchTestStore(t)
	defer store.Close()

	reqTrue := true
	pearls, err := store.List(storage.ListOptions{Required: &reqTrue})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(pearls) != 0 {
		t.Errorf("expected 0 required pearls, got %d", len(pearls))
	}
}

func TestClutch_RequiredOnly(t *testing.T) {
	store := setupClutchTestStore(t)
	defer store.Close()

	createRequiredPearl(t, store, "db.users", "db", "users", pearl.TypeTable, 10)
	createRequiredPearl(t, store, "api.auth", "api", "auth", pearl.TypeAPI, 5)
	createNonRequiredPearl(t, store, "db.logs", "db", "logs", pearl.TypeTable)

	reqTrue := true
	pearls, err := store.List(storage.ListOptions{Required: &reqTrue})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(pearls) != 2 {
		t.Errorf("expected 2 required pearls, got %d", len(pearls))
	}

	// Should be sorted by priority descending
	if pearls[0].ID != "db.users" {
		t.Errorf("expected first pearl to be db.users (priority 10), got %s", pearls[0].ID)
	}
	if pearls[1].ID != "api.auth" {
		t.Errorf("expected second pearl to be api.auth (priority 5), got %s", pearls[1].ID)
	}
}

func TestClutch_PrioritySorting(t *testing.T) {
	store := setupClutchTestStore(t)
	defer store.Close()

	createRequiredPearl(t, store, "low.pearl", "low", "pearl", pearl.TypeCustom, 1)
	createRequiredPearl(t, store, "high.pearl", "high", "pearl", pearl.TypeCustom, 100)
	createRequiredPearl(t, store, "mid.pearl", "mid", "pearl", pearl.TypeCustom, 50)

	reqTrue := true
	pearls, err := store.List(storage.ListOptions{Required: &reqTrue})
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if len(pearls) != 3 {
		t.Fatalf("expected 3 pearls, got %d", len(pearls))
	}

	// Priority descending
	if pearls[0].Priority != 100 {
		t.Errorf("expected first pearl priority 100, got %d", pearls[0].Priority)
	}
	if pearls[1].Priority != 50 {
		t.Errorf("expected second pearl priority 50, got %d", pearls[1].Priority)
	}
	if pearls[2].Priority != 1 {
		t.Errorf("expected third pearl priority 1, got %d", pearls[2].Priority)
	}
}

func TestClutch_ContentOutput(t *testing.T) {
	store := setupClutchTestStore(t)
	defer store.Close()

	createRequiredPearl(t, store, "db.users", "db", "users", pearl.TypeTable, 10)
	createRequiredPearl(t, store, "api.auth", "api", "auth", pearl.TypeAPI, 5)

	reqTrue := true
	pearls, err := store.List(storage.ListOptions{Required: &reqTrue})
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	var output strings.Builder
	for i, p := range pearls {
		if i > 0 {
			output.WriteString("\n---\n\n")
		}
		content, err := store.GetContent(p)
		if err != nil {
			t.Fatalf("get content: %v", err)
		}
		output.WriteString(content)
		if !strings.HasSuffix(content, "\n") {
			output.WriteString("\n")
		}
	}

	out := output.String()

	if !strings.Contains(out, "# users") {
		t.Error("output should contain users content")
	}
	if !strings.Contains(out, "# auth") {
		t.Error("output should contain auth content")
	}
	if !strings.Contains(out, "\n---\n\n") {
		t.Error("output should contain separator between pearls")
	}
}

func TestClutch_BriefOutput(t *testing.T) {
	store := setupClutchTestStore(t)
	defer store.Close()

	now := time.Now()
	p := &pearl.Pearl{
		ID: "db.users", Name: "users", Namespace: "db",
		Type: pearl.TypeTable, Status: pearl.StatusActive,
		Required:    true,
		Priority:    10,
		Description: "Users table",
		Tags:        []string{"pii", "core"},
		CreatedAt:   now, UpdatedAt: now,
	}
	if err := store.Create(p, "# Users\n\nFull content here\n"); err != nil {
		t.Fatalf("create: %v", err)
	}

	reqTrue := true
	pearls, err := store.List(storage.ListOptions{Required: &reqTrue})
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	var output strings.Builder
	for _, p := range pearls {
		output.WriteString("## " + p.ID + "\n\n")
		output.WriteString("- **Type:** " + string(p.Type) + "\n")
		output.WriteString("- **Status:** " + string(p.Status) + "\n")
		if p.Description != "" {
			output.WriteString("- **Description:** " + p.Description + "\n")
		}
		if len(p.Tags) > 0 {
			output.WriteString("- **Tags:** " + strings.Join(p.Tags, ", ") + "\n")
		}
		output.WriteString("\n")
	}

	out := output.String()

	if !strings.Contains(out, "## db.users") {
		t.Error("brief output should contain pearl ID header")
	}
	if !strings.Contains(out, "**Type:** table") {
		t.Error("brief output should contain type")
	}
	if !strings.Contains(out, "**Description:** Users table") {
		t.Error("brief output should contain description")
	}
	if !strings.Contains(out, "**Tags:** pii, core") {
		t.Error("brief output should contain tags")
	}
	if strings.Contains(out, "Full content here") {
		t.Error("brief output should NOT contain full content")
	}
}

func TestClutch_JSONOutput(t *testing.T) {
	store := setupClutchTestStore(t)
	defer store.Close()

	createRequiredPearl(t, store, "db.users", "db", "users", pearl.TypeTable, 10)
	createRequiredPearl(t, store, "api.auth", "api", "auth", pearl.TypeAPI, 5)

	reqTrue := true
	pearls, err := store.List(storage.ListOptions{Required: &reqTrue})
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	err = enc.Encode(map[string]interface{}{
		"pearls": pearls,
		"count":  len(pearls),
	})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	count := result["count"].(float64)
	if count != 2 {
		t.Errorf("expected count 2, got %v", count)
	}

	pearlList := result["pearls"].([]interface{})
	if len(pearlList) != 2 {
		t.Errorf("expected 2 pearls in JSON, got %d", len(pearlList))
	}
}
