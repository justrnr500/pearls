// Package pearl provides core types for the pearls data catalog.
package pearl

import (
	"fmt"
	"regexp"
	"time"

	"github.com/bmatcuk/doublestar/v4"
)

// AssetType represents the type of a pearl. Free-form string validated as
// non-empty, lowercase alphanumeric + hyphens.
type AssetType string

// Common asset type constants for convenience. The type system is open —
// any string matching the validation pattern is accepted.
const (
	TypeTable     AssetType = "table"
	TypeSchema    AssetType = "schema"
	TypeDatabase  AssetType = "database"
	TypeAPI       AssetType = "api"
	TypeEndpoint  AssetType = "endpoint"
	TypeFile      AssetType = "file"
	TypeBucket    AssetType = "bucket"
	TypePipeline  AssetType = "pipeline"
	TypeDashboard AssetType = "dashboard"
	TypeQuery     AssetType = "query"
	TypeCustom    AssetType = "custom"
)

// assetTypePattern matches valid asset types: lowercase alphanumeric + hyphens,
// must start with a letter.
var assetTypePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// IsValid returns true if the asset type matches the format: non-empty,
// lowercase alphanumeric + hyphens, starting with a letter.
func (t AssetType) IsValid() bool {
	return assetTypePattern.MatchString(string(t))
}

// scopePattern matches valid scope strings: lowercase alphanumeric + hyphens,
// must start with a letter.
var scopePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// ValidateScopes checks that all scope strings are valid.
func ValidateScopes(scopes []string) error {
	for _, s := range scopes {
		if !scopePattern.MatchString(s) {
			return fmt.Errorf("invalid scope %q: must be lowercase alphanumeric + hyphens, starting with a letter", s)
		}
	}
	return nil
}

// ValidateGlobs checks that all glob patterns have valid syntax.
func ValidateGlobs(globs []string) error {
	for _, g := range globs {
		if g == "" {
			return fmt.Errorf("glob pattern cannot be empty")
		}
		if !doublestar.ValidatePattern(g) {
			return fmt.Errorf("invalid glob pattern %q", g)
		}
	}
	return nil
}

// Status represents the lifecycle status of a pearl.
type Status string

const (
	StatusActive     Status = "active"
	StatusDeprecated Status = "deprecated"
	StatusArchived   Status = "archived"
)

// ValidStatuses returns all valid statuses.
func ValidStatuses() []Status {
	return []Status{StatusActive, StatusDeprecated, StatusArchived}
}

// IsValid returns true if the status is valid.
func (s Status) IsValid() bool {
	for _, valid := range ValidStatuses() {
		if s == valid {
			return true
		}
	}
	return false
}

// ConnectionInfo holds connection details for database/API assets.
type ConnectionInfo struct {
	Type     string            `json:"type"`               // postgres, mysql, snowflake, bigquery, s3, etc.
	Host     string            `json:"host,omitempty"`     // Can be env var reference: ${DB_HOST}
	Port     int               `json:"port,omitempty"`     // Port number
	Database string            `json:"database,omitempty"` // Database name
	Schema   string            `json:"schema,omitempty"`   // Schema name
	Extras   map[string]string `json:"extras,omitempty"`   // Additional connection params
}

// Pearl represents a documented data asset.
type Pearl struct {
	// Identity
	ID        string `json:"id"`        // e.g., "prl-a3f8" or "db.postgres.users"
	Name      string `json:"name"`      // Human-readable name
	Namespace string `json:"namespace"` // Dot-separated path: "db.postgres"

	// Classification
	Type   AssetType `json:"type"`   // Free-form: table, api, convention, brainstorm, etc.
	Tags   []string  `json:"tags"`   // Freeform tags: ["pii", "analytics", "deprecated"]
	Globs  []string  `json:"globs,omitempty"`  // File-path glob patterns for push-based context injection
	Scopes []string  `json:"scopes,omitempty"` // Contextual groupings for scope-based injection

	// Content
	Description string `json:"description"`  // Brief one-liner
	ContentPath string `json:"content_path"` // Path to markdown file
	ContentHash string `json:"content_hash"` // SHA256 of content for change detection

	// Relationships
	References []string `json:"references,omitempty"` // IDs of related pearls
	Parent     string   `json:"parent,omitempty"`     // Parent pearl ID (for hierarchical assets)

	// Connection (optional, for databases/APIs)
	Connection *ConnectionInfo `json:"connection,omitempty"`

	// Metadata
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedBy string    `json:"created_by"`
	Status    Status    `json:"status"`
}

// FullID returns the fully-qualified ID (namespace + name).
func (p *Pearl) FullID() string {
	if p.Namespace == "" {
		return p.Name
	}
	return p.Namespace + "." + p.Name
}
