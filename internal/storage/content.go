package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/justrnr500/pearls/internal/pearl"
)

// Content manages markdown content files for pearls.
type Content struct {
	baseDir string
}

// NewContent creates a new content manager for the given base directory.
func NewContent(baseDir string) *Content {
	return &Content{baseDir: baseDir}
}

// BaseDir returns the content base directory.
func (c *Content) BaseDir() string {
	return c.baseDir
}

// PathForPearl returns the content file path for a pearl.
// Example: namespace "db.postgres", name "users" -> "db/postgres/users.md"
func (c *Content) PathForPearl(namespace, name string) string {
	parts := []string{}
	if namespace != "" {
		parts = append(parts, strings.Split(namespace, ".")...)
	}
	parts = append(parts, name+".md")
	return filepath.Join(parts...)
}

// FullPath returns the absolute path to a content file.
func (c *Content) FullPath(relativePath string) string {
	return filepath.Join(c.baseDir, relativePath)
}

// Read reads the content of a pearl's markdown file.
func (c *Content) Read(relativePath string) (string, error) {
	fullPath := c.FullPath(relativePath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("read content file: %w", err)
	}
	return string(data), nil
}

// Write writes content to a pearl's markdown file.
func (c *Content) Write(relativePath, content string) error {
	fullPath := c.FullPath(relativePath)

	// Ensure directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create content directory: %w", err)
	}

	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write content file: %w", err)
	}

	return nil
}

// Delete removes a pearl's content file.
func (c *Content) Delete(relativePath string) error {
	fullPath := c.FullPath(relativePath)
	err := os.Remove(fullPath)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// Exists checks if a content file exists.
func (c *Content) Exists(relativePath string) bool {
	fullPath := c.FullPath(relativePath)
	_, err := os.Stat(fullPath)
	return err == nil
}

// Hash computes the SHA256 hash of a content file.
func (c *Content) Hash(relativePath string) (string, error) {
	fullPath := c.FullPath(relativePath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("read file for hash: %w", err)
	}
	return HashContent(data), nil
}

// HashContent computes the SHA256 hash of content bytes.
func HashContent(content []byte) string {
	h := sha256.Sum256(content)
	return hex.EncodeToString(h[:])
}

// HashString computes the SHA256 hash of a string.
func HashString(content string) string {
	return HashContent([]byte(content))
}

// Template returns a default markdown template for a pearl.
func (c *Content) Template(p *pearl.Pearl) string {
	var sb strings.Builder

	sb.WriteString("# ")
	sb.WriteString(p.Name)
	sb.WriteString("\n\n")

	if p.Description != "" {
		sb.WriteString(p.Description)
		sb.WriteString("\n\n")
	}

	switch p.Type {
	case pearl.TypeTable:
		sb.WriteString("## Schema\n\n")
		sb.WriteString("| Column | Type | Nullable | Description |\n")
		sb.WriteString("|--------|------|----------|-------------|\n")
		sb.WriteString("| id | | | |\n\n")
		sb.WriteString("## Relationships\n\n")
		sb.WriteString("## Access Patterns\n\n")
		sb.WriteString("```sql\n-- Example query\n```\n\n")
		sb.WriteString("## Notes\n\n")

	case pearl.TypeAPI, pearl.TypeEndpoint:
		sb.WriteString("## Endpoints\n\n")
		sb.WriteString("## Authentication\n\n")
		sb.WriteString("## Examples\n\n")
		sb.WriteString("```bash\n# Example request\n```\n\n")
		sb.WriteString("## Notes\n\n")

	case pearl.TypeDatabase, pearl.TypeSchema:
		sb.WriteString("## Overview\n\n")
		sb.WriteString("## Tables\n\n")
		sb.WriteString("## Access\n\n")
		sb.WriteString("## Notes\n\n")

	default:
		sb.WriteString("## Overview\n\n")
		sb.WriteString("## Details\n\n")
		sb.WriteString("## Notes\n\n")
	}

	return sb.String()
}

// ListFiles returns all markdown files in the content directory.
func (c *Content) ListFiles() ([]string, error) {
	var files []string

	err := filepath.WalkDir(c.baseDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".md") {
			rel, err := filepath.Rel(c.baseDir, path)
			if err != nil {
				return err
			}
			files = append(files, rel)
		}
		return nil
	})

	if os.IsNotExist(err) {
		return nil, nil
	}

	return files, err
}
