package cmd

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/justrnr500/pearls/internal/storage"
	"github.com/spf13/cobra"
)

//go:embed templates/prime-triggers.md
var primeTriggers string

//go:embed templates/prime-reference.md
var primeReference string

var primeCmd = &cobra.Command{
	Use:   "prime",
	Short: "Output AI-optimized workflow context",
	Long: `Output essential Pearls workflow context for AI agent session priming.

Outputs a catalog summary, discovery triggers, and quick command reference.
Adapts output based on catalog size (empty, small, or large).

If .pearls/PRIME.md exists, outputs that file instead.

Designed for Claude Code SessionStart hooks to prime agents with
pearls awareness after session start or context compaction.`,
	RunE: runPrime,
}

func init() {
	rootCmd.AddCommand(primeCmd)
}

func runPrime(cmd *cobra.Command, args []string) error {
	store, paths, err := getStore()
	if err != nil {
		// Not in a pearls project — silent exit
		return nil
	}
	defer store.Close()

	// Check for PRIME.md override
	overridePath := ""
	if paths != nil {
		overridePath = paths.Root + "/.pearls/PRIME.md"
	}

	return writePrimeOutput(os.Stdout, store, overridePath)
}

// writePrimeOutput generates the prime context and writes it to w.
// If overridePath points to an existing file, its content is used instead.
func writePrimeOutput(w io.Writer, store *storage.Store, overridePath string) error {
	// Check for override file
	if overridePath != "" {
		if data, err := os.ReadFile(overridePath); err == nil {
			_, err := w.Write(data)
			return err
		}
	}

	// Get all pearls to build summary
	pearls, err := store.List(storage.ListOptions{})
	if err != nil {
		return fmt.Errorf("list pearls: %w", err)
	}

	count := len(pearls)

	switch {
	case count == 0:
		return writeEmptyPrime(w)
	case count <= 20:
		return writeSmallPrime(w, store, count)
	default:
		return writeLargePrime(w, store, count)
	}
}

func writeEmptyPrime(w io.Writer) error {
	_, err := fmt.Fprintf(w, "# Pearls Context\n\nPearls is installed but the catalog is empty. As you work, save reusable knowledge:\n\n%s\n%s", primeTriggers, primeReference)
	return err
}

func writeSmallPrime(w io.Writer, store *storage.Store, count int) error {
	summary := buildTypeSummary(store)

	_, err := fmt.Fprintf(w, "# Pearls Context\n\nYour catalog has %d pearls: %s\n\nBefore working on unfamiliar code, check for existing knowledge with `pl context --for <path>`.\n\n%s\n%s", count, summary, primeTriggers, primeReference)
	return err
}

func writeLargePrime(w io.Writer, store *storage.Store, count int) error {
	summary := buildTypeSummary(store)
	scopes := buildScopeSummary(store)

	scopeSection := ""
	if scopes != "" {
		scopeSection = fmt.Sprintf("\nScopes: %s\n", scopes)
	}

	_, err := fmt.Fprintf(w, "# Pearls Context\n\nYour catalog has %d pearls: %s\n%s\nSearch before starting work: `pl search \"<query>\" --semantic` or `pl context --for <path>`.\n\n%s\n%s", count, summary, scopeSection, primeTriggers, primeReference)
	return err
}

// buildTypeSummary returns a string like "3 table, 2 api, 1 convention"
func buildTypeSummary(store *storage.Store) string {
	pearls, err := store.List(storage.ListOptions{})
	if err != nil {
		return "unknown"
	}

	counts := make(map[string]int)
	for _, p := range pearls {
		counts[string(p.Type)]++
	}

	// Sort by count descending, then name ascending
	type typeCount struct {
		name  string
		count int
	}
	var sorted []typeCount
	for name, count := range counts {
		sorted = append(sorted, typeCount{name, count})
	}
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].count != sorted[j].count {
			return sorted[i].count > sorted[j].count
		}
		return sorted[i].name < sorted[j].name
	})

	parts := make([]string, len(sorted))
	for i, tc := range sorted {
		parts[i] = fmt.Sprintf("%d %s", tc.count, tc.name)
	}
	return strings.Join(parts, ", ")
}

// buildScopeSummary returns a string like "backend, frontend, payments"
func buildScopeSummary(store *storage.Store) string {
	pearls, err := store.List(storage.ListOptions{})
	if err != nil {
		return ""
	}

	scopeSet := make(map[string]bool)
	for _, p := range pearls {
		for _, s := range p.Scopes {
			scopeSet[s] = true
		}
	}

	if len(scopeSet) == 0 {
		return ""
	}

	scopes := make([]string, 0, len(scopeSet))
	for s := range scopeSet {
		scopes = append(scopes, s)
	}
	sort.Strings(scopes)
	return strings.Join(scopes, ", ")
}
