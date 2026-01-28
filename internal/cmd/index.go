package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/justrnr500/pearls/internal/config"
	"github.com/justrnr500/pearls/internal/embedding"
	"github.com/justrnr500/pearls/internal/storage"
)

var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Manage vector search index",
	Long: `Manage the vector search index for semantic search.

Use 'index --rebuild' to regenerate embeddings for all pearls.`,
}

var (
	indexRebuild bool
	indexJSON    bool
)

func init() {
	rootCmd.AddCommand(indexCmd)
	indexCmd.Flags().BoolVar(&indexRebuild, "rebuild", false, "Rebuild all embeddings")
	indexCmd.Flags().BoolVar(&indexJSON, "json", false, "Output as JSON")
	indexCmd.RunE = runIndex
}

func runIndex(cmd *cobra.Command, args []string) error {
	if indexRebuild {
		return runIndexRebuild()
	}

	// Default: show index status
	return runIndexStatus()
}

func runIndexStatus() error {
	store, _, err := getStore()
	if err != nil {
		return err
	}
	defer store.Close()

	pearlCount, err := store.DB().Count()
	if err != nil {
		return fmt.Errorf("count pearls: %w", err)
	}

	embeddingCount, err := store.DB().EmbeddingCount()
	if err != nil {
		return fmt.Errorf("count embeddings: %w", err)
	}

	cfg, _ := getConfig()
	vectorEnabled := cfg != nil && cfg.VectorSearch.Enabled

	if indexJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]interface{}{
			"vector_search_enabled": vectorEnabled,
			"pearl_count":           pearlCount,
			"embedding_count":       embeddingCount,
			"indexed_percent":       indexedPercent(embeddingCount, pearlCount),
		})
	}

	fmt.Printf("Vector Search Index\n")
	fmt.Printf("───────────────────\n")
	fmt.Printf("Enabled:     %v\n", vectorEnabled)
	fmt.Printf("Pearls:      %d\n", pearlCount)
	fmt.Printf("Indexed:     %d (%.0f%%)\n", embeddingCount, indexedPercent(embeddingCount, pearlCount))

	if embeddingCount < pearlCount {
		fmt.Printf("\nRun 'pearls index --rebuild' to index all pearls.\n")
	}

	return nil
}

func runIndexRebuild() error {
	// Get paths for store and config
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	root, err := config.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("not in a pearls directory: run 'pearls init' first")
	}

	paths := config.ResolvePaths(root)

	// Load config
	cfg, err := config.Load(paths.Config)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if !cfg.VectorSearch.Enabled {
		return fmt.Errorf("vector search is disabled in config\nSet vector_search.enabled: true in %s", paths.Config)
	}

	// Open store without embedder (we'll create our own)
	store, err := storage.NewStore(paths.DB, paths.JSONL, paths.Content)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer store.Close()

	// Create embedder
	modelPath := cfg.VectorSearch.ModelPath
	if modelPath == "" {
		modelPath = embedding.DefaultModelDir()
	}

	fmt.Printf("Initializing embedding model...\n")
	embedder, err := embedding.New(modelPath)
	if err != nil {
		return fmt.Errorf("create embedder: %w", err)
	}
	defer embedder.Close()

	// Get all pearls
	pearls, err := store.DB().All()
	if err != nil {
		return fmt.Errorf("get pearls: %w", err)
	}

	if len(pearls) == 0 {
		fmt.Printf("No pearls to index.\n")
		return nil
	}

	// Clear existing embeddings
	fmt.Printf("Clearing existing embeddings...\n")
	if err := store.DB().ClearEmbeddings(); err != nil {
		return fmt.Errorf("clear embeddings: %w", err)
	}

	// Generate embeddings for each pearl
	var indexed, failed int
	for i, p := range pearls {
		// Show progress
		fmt.Printf("\r[%d/%d] Indexing %s...", i+1, len(pearls), truncateID(p.ID, 40))

		// Get content
		content, err := store.GetContent(p)
		if err != nil {
			failed++
			continue
		}

		// Generate embedding text
		embText := p.Description
		if content != "" {
			if embText != "" {
				embText += "\n\n"
			}
			embText += content
		}

		if embText == "" {
			failed++
			continue
		}

		// Generate embedding
		emb, err := embedder.Embed(embText)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nWarning: failed to embed %s: %v\n", p.ID, err)
			failed++
			continue
		}

		// Store embedding
		if err := store.DB().InsertEmbedding(p.ID, emb); err != nil {
			fmt.Fprintf(os.Stderr, "\nWarning: failed to store embedding for %s: %v\n", p.ID, err)
			failed++
			continue
		}

		indexed++
	}

	// Clear progress line and print summary
	fmt.Printf("\r%s\r", "                                                                ")

	if indexJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]interface{}{
			"total":   len(pearls),
			"indexed": indexed,
			"failed":  failed,
		})
	}

	fmt.Printf("✓ Indexed %d pearls", indexed)
	if failed > 0 {
		fmt.Printf(" (%d failed)", failed)
	}
	fmt.Printf("\n")

	return nil
}

func indexedPercent(indexed, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(indexed) / float64(total) * 100
}

func truncateID(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
