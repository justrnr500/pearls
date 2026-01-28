// Package config handles pearls configuration.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	// DirName is the name of the pearls directory.
	DirName = ".pearls"
	// ConfigFile is the name of the config file.
	ConfigFile = "config.yaml"
	// DBFile is the name of the SQLite database file.
	DBFile = "pearls.db"
	// JSONLFile is the name of the JSONL metadata file.
	JSONLFile = "pearls.jsonl"
	// ContentDir is the name of the content directory.
	ContentDir = "content"
	// GitIgnoreFile is the name of the gitignore file.
	GitIgnoreFile = ".gitignore"
)

// Config represents the pearls configuration.
type Config struct {
	Project ProjectConfig `yaml:"project"`
	Storage StorageConfig `yaml:"storage"`
	Defaults DefaultsConfig `yaml:"defaults"`
	Aliases map[string]string `yaml:"aliases,omitempty"`
}

// ProjectConfig holds project identification settings.
type ProjectConfig struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
}

// StorageConfig holds storage settings.
type StorageConfig struct {
	ContentDir string `yaml:"content_dir"`
}

// DefaultsConfig holds default values for new pearls.
type DefaultsConfig struct {
	Status    string `yaml:"status"`
	CreatedBy string `yaml:"created_by"`
}

// Default returns a default configuration.
func Default() *Config {
	user := os.Getenv("USER")
	if user == "" {
		user = "unknown"
	}

	return &Config{
		Project: ProjectConfig{
			Name:        "my-data-catalog",
			Description: "Data asset catalog",
		},
		Storage: StorageConfig{
			ContentDir: ContentDir,
		},
		Defaults: DefaultsConfig{
			Status:    "active",
			CreatedBy: "${USER}",
		},
		Aliases: map[string]string{},
	}
}

// Load reads the configuration from a file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}

// Save writes the configuration to a file.
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

// Paths holds the resolved paths for a pearls installation.
type Paths struct {
	Root    string // .pearls directory
	Config  string // config.yaml
	DB      string // pearls.db
	JSONL   string // pearls.jsonl
	Content string // content/
}

// ResolvePaths returns the paths for a pearls installation rooted at the given directory.
func ResolvePaths(root string) *Paths {
	pearlsDir := filepath.Join(root, DirName)
	return &Paths{
		Root:    pearlsDir,
		Config:  filepath.Join(pearlsDir, ConfigFile),
		DB:      filepath.Join(pearlsDir, DBFile),
		JSONL:   filepath.Join(pearlsDir, JSONLFile),
		Content: filepath.Join(pearlsDir, ContentDir),
	}
}

// FindRoot searches for a .pearls directory starting from the given path
// and walking up the directory tree.
func FindRoot(startPath string) (string, error) {
	absPath, err := filepath.Abs(startPath)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}

	current := absPath
	for {
		pearlsDir := filepath.Join(current, DirName)
		if info, err := os.Stat(pearlsDir); err == nil && info.IsDir() {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Reached root
			return "", fmt.Errorf("not a pearls directory (or any parent): %s", startPath)
		}
		current = parent
	}
}

// Exists checks if a pearls installation exists at the given path.
func Exists(path string) bool {
	pearlsDir := filepath.Join(path, DirName)
	info, err := os.Stat(pearlsDir)
	return err == nil && info.IsDir()
}
