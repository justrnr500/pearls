package cmd

import (
	"fmt"
	"os"

	"github.com/justrnr500/pearls/internal/config"
	"github.com/justrnr500/pearls/internal/storage"
)

// getStore finds the pearls root and returns an open store.
func getStore() (*storage.Store, *config.Paths, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, nil, fmt.Errorf("get working directory: %w", err)
	}

	root, err := config.FindRoot(cwd)
	if err != nil {
		return nil, nil, fmt.Errorf("not in a pearls directory: run 'pearls init' first")
	}

	paths := config.ResolvePaths(root)
	store, err := storage.NewStore(paths.DB, paths.JSONL, paths.Content)
	if err != nil {
		return nil, nil, fmt.Errorf("open store: %w", err)
	}

	return store, paths, nil
}

// getConfig loads the configuration from the current pearls directory.
func getConfig() (*config.Config, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get working directory: %w", err)
	}

	root, err := config.FindRoot(cwd)
	if err != nil {
		return nil, fmt.Errorf("not in a pearls directory: run 'pearls init' first")
	}

	paths := config.ResolvePaths(root)
	cfg, err := config.Load(paths.Config)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	return cfg, nil
}
