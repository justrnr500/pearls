package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justrnr500/pearls/internal/storage"
)

func TestOnboardTemplate(t *testing.T) {
	content := onboardTemplate()
	if !strings.Contains(content, "<!-- pearls:start -->") {
		t.Error("template missing start marker")
	}
	if !strings.Contains(content, "<!-- pearls:end -->") {
		t.Error("template missing end marker")
	}
	if !strings.Contains(content, "pl list") {
		t.Error("template missing pl list command")
	}
}

func TestOnboardNewFile(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "CLAUDE.md")

	err := onboardToFile(target, false)
	if err != nil {
		t.Fatalf("onboard to new file: %v", err)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "<!-- pearls:start -->") {
		t.Error("file missing start marker")
	}
	if !strings.Contains(content, "<!-- pearls:end -->") {
		t.Error("file missing end marker")
	}
}

func TestOnboardAppendToExisting(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "CLAUDE.md")

	os.WriteFile(target, []byte("# My Project\n\nExisting content.\n"), 0644)

	err := onboardToFile(target, false)
	if err != nil {
		t.Fatalf("onboard: %v", err)
	}

	data, _ := os.ReadFile(target)
	content := string(data)

	if !strings.HasPrefix(content, "# My Project") {
		t.Error("existing content should be preserved at top")
	}
	if !strings.Contains(content, "<!-- pearls:start -->") {
		t.Error("pearls block should be appended")
	}
}

func TestOnboardSkipIfAlreadyOnboarded(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "CLAUDE.md")

	onboardToFile(target, false)
	data1, _ := os.ReadFile(target)

	err := onboardToFile(target, false)
	if err != nil {
		t.Fatalf("second onboard: %v", err)
	}

	data2, _ := os.ReadFile(target)
	if string(data1) != string(data2) {
		t.Error("file should not change on second onboard without --force")
	}
}

func TestOnboardForceReplaces(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "CLAUDE.md")

	old := "# Existing\n\n<!-- pearls:start -->\nOLD CONTENT\n<!-- pearls:end -->\n\nAfter.\n"
	os.WriteFile(target, []byte(old), 0644)

	err := onboardToFile(target, true)
	if err != nil {
		t.Fatalf("force onboard: %v", err)
	}

	data, _ := os.ReadFile(target)
	content := string(data)

	if strings.Contains(content, "OLD CONTENT") {
		t.Error("old content should be replaced")
	}
	if !strings.Contains(content, "pl list") {
		t.Error("new template should be injected")
	}
	if !strings.Contains(content, "# Existing") {
		t.Error("content before markers should be preserved")
	}
	if !strings.Contains(content, "After.") {
		t.Error("content after markers should be preserved")
	}
}

func TestSetupClaudeHooks_RegistersSessionStart(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, ".claude", "settings.json")
	os.MkdirAll(filepath.Dir(settingsPath), 0755)

	// Write empty settings
	os.WriteFile(settingsPath, []byte("{}"), 0644)

	err := registerHook(settingsPath, filepath.Join(tmpDir, "pearls-context.sh"))
	if err != nil {
		t.Fatalf("registerHook: %v", err)
	}

	// Now register the session start hook
	err = registerSessionStartHook(settingsPath)
	if err != nil {
		t.Fatalf("registerSessionStartHook: %v", err)
	}

	data, _ := os.ReadFile(settingsPath)
	var settings map[string]interface{}
	json.Unmarshal(data, &settings)

	hooks, _ := settings["hooks"].(map[string]interface{})
	if hooks == nil {
		t.Fatal("expected hooks section in settings")
	}

	// Check SessionStart exists
	sessionStart, ok := hooks["SessionStart"]
	if !ok {
		t.Fatal("expected SessionStart hook")
	}

	// Check it contains pearls clutch (not pearls prime)
	raw, _ := json.Marshal(sessionStart)
	if !strings.Contains(string(raw), "pearls clutch") {
		t.Error("SessionStart hook should run 'pearls clutch'")
	}
	if strings.Contains(string(raw), "pearls prime") {
		t.Error("SessionStart hook should not contain 'pearls prime'")
	}

	// Check UserPromptSubmit still exists
	if _, ok := hooks["UserPromptSubmit"]; !ok {
		t.Error("UserPromptSubmit hook should still be present")
	}
}

func TestRegisterSessionStartHook_NoDuplicates(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, ".claude", "settings.json")
	os.MkdirAll(filepath.Dir(settingsPath), 0755)

	os.WriteFile(settingsPath, []byte("{}"), 0644)

	// Register twice
	registerSessionStartHook(settingsPath)
	registerSessionStartHook(settingsPath)

	data, _ := os.ReadFile(settingsPath)
	var settings map[string]interface{}
	json.Unmarshal(data, &settings)

	hooks, _ := settings["hooks"].(map[string]interface{})
	sessionStart, _ := hooks["SessionStart"].([]interface{})

	if len(sessionStart) != 1 {
		t.Errorf("expected 1 SessionStart hook entry, got %d", len(sessionStart))
	}
}

func TestRegisterSessionStartHook_UpgradeFromPrime(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, ".claude", "settings.json")
	os.MkdirAll(filepath.Dir(settingsPath), 0755)

	// Write settings with old "pearls prime" hook
	oldSettings := `{
  "hooks": {
    "SessionStart": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "pearls prime",
            "timeout": 10
          }
        ]
      }
    ]
  }
}`
	os.WriteFile(settingsPath, []byte(oldSettings), 0644)

	err := registerSessionStartHook(settingsPath)
	if err != nil {
		t.Fatalf("registerSessionStartHook: %v", err)
	}

	data, _ := os.ReadFile(settingsPath)
	var settings map[string]interface{}
	json.Unmarshal(data, &settings)

	hooks, _ := settings["hooks"].(map[string]interface{})
	sessionStart, _ := hooks["SessionStart"].([]interface{})

	// Should still be 1 entry (upgraded in place, not added new)
	if len(sessionStart) != 1 {
		t.Errorf("expected 1 SessionStart hook entry after upgrade, got %d", len(sessionStart))
	}

	// Check it now contains pearls clutch
	raw, _ := json.Marshal(sessionStart)
	if !strings.Contains(string(raw), "pearls clutch") {
		t.Error("upgraded hook should run 'pearls clutch'")
	}
	if strings.Contains(string(raw), "pearls prime") {
		t.Error("upgraded hook should not contain 'pearls prime'")
	}
}

func TestCreateSeedPearls(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewStore(
		filepath.Join(tmpDir, "pearls.db"),
		filepath.Join(tmpDir, "pearls.jsonl"),
		filepath.Join(tmpDir, "content"),
	)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	seeds := seedPearls()
	if len(seeds) != 2 {
		t.Fatalf("expected 2 seed pearls, got %d", len(seeds))
	}

	// Verify seed definitions
	triggersSeed := seeds[0]
	if triggersSeed.ID != "sys.triggers" {
		t.Errorf("expected sys.triggers, got %s", triggersSeed.ID)
	}
	refSeed := seeds[1]
	if refSeed.ID != "sys.reference" {
		t.Errorf("expected sys.reference, got %s", refSeed.ID)
	}

	// Verify embedded content is non-empty
	if len(seedTriggersContent) == 0 {
		t.Error("seed triggers content should not be empty")
	}
	if len(seedReferenceContent) == 0 {
		t.Error("seed reference content should not be empty")
	}
}
