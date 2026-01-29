package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
