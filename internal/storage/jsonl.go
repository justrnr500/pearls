package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/justrnr500/pearls/internal/pearl"
)

// JSONL handles reading and writing pearls to JSONL files.
type JSONL struct {
	path string
}

// NewJSONL creates a new JSONL handler for the given file path.
func NewJSONL(path string) *JSONL {
	return &JSONL{path: path}
}

// Path returns the JSONL file path.
func (j *JSONL) Path() string {
	return j.path
}

// ReadAll reads all pearls from the JSONL file.
func (j *JSONL) ReadAll() ([]*pearl.Pearl, error) {
	file, err := os.Open(j.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open jsonl file: %w", err)
	}
	defer file.Close()

	var pearls []*pearl.Pearl
	scanner := bufio.NewScanner(file)

	// Increase buffer size for long lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var p pearl.Pearl
		if err := json.Unmarshal(line, &p); err != nil {
			return nil, fmt.Errorf("parse line %d: %w", lineNum, err)
		}
		pearls = append(pearls, &p)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read jsonl file: %w", err)
	}

	return pearls, nil
}

// WriteAll writes all pearls to the JSONL file (overwrites existing).
func (j *JSONL) WriteAll(pearls []*pearl.Pearl) error {
	// Ensure directory exists
	dir := filepath.Dir(j.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Write to temp file first, then rename for atomicity
	tmpPath := j.path + ".tmp"
	file, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	encoder := json.NewEncoder(file)
	encoder.SetEscapeHTML(false)

	for _, p := range pearls {
		if err := encoder.Encode(p); err != nil {
			file.Close()
			os.Remove(tmpPath)
			return fmt.Errorf("encode pearl %s: %w", p.ID, err)
		}
	}

	if err := file.Sync(); err != nil {
		file.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("sync file: %w", err)
	}

	if err := file.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close file: %w", err)
	}

	if err := os.Rename(tmpPath, j.path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename temp file: %w", err)
	}

	return nil
}

// Append adds a pearl to the end of the JSONL file.
func (j *JSONL) Append(p *pearl.Pearl) error {
	// Ensure directory exists
	dir := filepath.Dir(j.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	file, err := os.OpenFile(j.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open file for append: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetEscapeHTML(false)

	if err := encoder.Encode(p); err != nil {
		return fmt.Errorf("encode pearl: %w", err)
	}

	return nil
}

// Exists returns true if the JSONL file exists.
func (j *JSONL) Exists() bool {
	_, err := os.Stat(j.path)
	return err == nil
}
