package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/justrnr500/pearls/internal/config"
	"github.com/justrnr500/pearls/internal/storage"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check catalog health",
	Long: `Run health checks on the pearls catalog to diagnose common issues.

Checks:
  - JSONL/SQLite sync (pearl counts and IDs match)
  - Orphaned content (markdown files with no pearl)
  - Missing content (pearls with content_path that doesn't exist)
  - Broken references (pearls referencing IDs that don't exist)
  - Config validity (config.yaml parses without errors)`,
	RunE: runDoctor,
}

var doctorJSON bool

func init() {
	rootCmd.AddCommand(doctorCmd)
	doctorCmd.Flags().BoolVar(&doctorJSON, "json", false, "Output as JSON")
}

// CheckResult represents the result of a single health check.
type CheckResult struct {
	Name   string   `json:"name"`
	Passed bool     `json:"passed"`
	Issues []string `json:"issues,omitempty"`
}

func runDoctor(cmd *cobra.Command, args []string) error {
	store, paths, err := getStore()
	if err != nil {
		return err
	}
	defer store.Close()

	checks := []CheckResult{
		checkJSONLSync(store),
		checkOrphanedContent(store),
		checkMissingContent(store),
		checkBrokenReferences(store),
		checkConfigValidity(paths.Config),
	}

	if doctorJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(checks)
	}

	allPassed := true
	for _, c := range checks {
		if c.Passed {
			fmt.Printf("✓ %s\n", c.Name)
		} else {
			allPassed = false
			fmt.Printf("✗ %s\n", c.Name)
			for _, issue := range c.Issues {
				fmt.Printf("    %s\n", issue)
			}
		}
	}

	if !allPassed {
		return fmt.Errorf("some checks failed")
	}

	return nil
}

func checkJSONLSync(store *storage.Store) CheckResult {
	name := "JSONL/SQLite in sync"

	jsonlPearls, err := store.JSONL().ReadAll()
	if err != nil {
		return CheckResult{Name: name, Passed: false, Issues: []string{fmt.Sprintf("read JSONL: %v", err)}}
	}

	dbCount, err := store.DB().Count()
	if err != nil {
		return CheckResult{Name: name, Passed: false, Issues: []string{fmt.Sprintf("count DB: %v", err)}}
	}

	if len(jsonlPearls) != dbCount {
		return CheckResult{
			Name:   name,
			Passed: false,
			Issues: []string{fmt.Sprintf("count mismatch: JSONL=%d, SQLite=%d", len(jsonlPearls), dbCount)},
		}
	}

	jsonlIDs := make(map[string]bool)
	for _, p := range jsonlPearls {
		jsonlIDs[p.ID] = true
	}

	dbPearls, err := store.DB().All()
	if err != nil {
		return CheckResult{Name: name, Passed: false, Issues: []string{fmt.Sprintf("list DB: %v", err)}}
	}

	var missingInJSONL []string
	for _, p := range dbPearls {
		if !jsonlIDs[p.ID] {
			missingInJSONL = append(missingInJSONL, p.ID)
		}
	}

	if len(missingInJSONL) > 0 {
		return CheckResult{
			Name:   name,
			Passed: false,
			Issues: []string{fmt.Sprintf("in SQLite but not JSONL: %v", missingInJSONL)},
		}
	}

	return CheckResult{Name: fmt.Sprintf("JSONL/SQLite in sync (%d pearls)", dbCount), Passed: true}
}

func checkOrphanedContent(store *storage.Store) CheckResult {
	name := "No orphaned content files"

	files, err := store.Content().ListFiles()
	if err != nil {
		return CheckResult{Name: name, Passed: false, Issues: []string{fmt.Sprintf("list files: %v", err)}}
	}

	pearls, err := store.DB().All()
	if err != nil {
		return CheckResult{Name: name, Passed: false, Issues: []string{fmt.Sprintf("list pearls: %v", err)}}
	}

	pearlPaths := make(map[string]bool)
	for _, p := range pearls {
		if p.ContentPath != "" {
			pearlPaths[p.ContentPath] = true
		}
	}

	var orphans []string
	for _, f := range files {
		if !pearlPaths[f] {
			orphans = append(orphans, f)
		}
	}

	if len(orphans) > 0 {
		return CheckResult{Name: name, Passed: false, Issues: orphans}
	}

	return CheckResult{Name: name, Passed: true}
}

func checkMissingContent(store *storage.Store) CheckResult {
	name := "No missing content files"

	pearls, err := store.DB().All()
	if err != nil {
		return CheckResult{Name: name, Passed: false, Issues: []string{fmt.Sprintf("list pearls: %v", err)}}
	}

	var missing []string
	for _, p := range pearls {
		if p.ContentPath != "" && !store.Content().Exists(p.ContentPath) {
			missing = append(missing, p.ID)
		}
	}

	if len(missing) > 0 {
		return CheckResult{
			Name:   name,
			Passed: false,
			Issues: []string{fmt.Sprintf("%d pearls missing content: %v", len(missing), missing)},
		}
	}

	return CheckResult{Name: name, Passed: true}
}

func checkBrokenReferences(store *storage.Store) CheckResult {
	name := "All references valid"

	pearls, err := store.DB().All()
	if err != nil {
		return CheckResult{Name: name, Passed: false, Issues: []string{fmt.Sprintf("list pearls: %v", err)}}
	}

	ids := make(map[string]bool)
	for _, p := range pearls {
		ids[p.ID] = true
	}

	var broken []string
	for _, p := range pearls {
		for _, ref := range p.References {
			if !ids[ref] {
				broken = append(broken, fmt.Sprintf("%s -> %s", p.ID, ref))
			}
		}
	}

	if len(broken) > 0 {
		return CheckResult{Name: name, Passed: false, Issues: broken}
	}

	return CheckResult{Name: name, Passed: true}
}

func checkConfigValidity(configPath string) CheckResult {
	name := "Config valid"

	_, err := config.Load(configPath)
	if err != nil {
		return CheckResult{Name: name, Passed: false, Issues: []string{err.Error()}}
	}

	return CheckResult{Name: name, Passed: true}
}
