// Package pearl provides core types for the pearls data catalog.
package pearl

import (
	"time"
)

// AssetType represents the type of data asset.
type AssetType string

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

// ValidAssetTypes returns all valid asset types.
func ValidAssetTypes() []AssetType {
	return []AssetType{
		TypeTable, TypeSchema, TypeDatabase, TypeAPI, TypeEndpoint,
		TypeFile, TypeBucket, TypePipeline, TypeDashboard, TypeQuery, TypeCustom,
	}
}

// IsValid returns true if the asset type is valid.
func (t AssetType) IsValid() bool {
	for _, valid := range ValidAssetTypes() {
		if t == valid {
			return true
		}
	}
	return false
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
	Type AssetType `json:"type"` // table, schema, database, api, file, bucket, etc.
	Tags []string  `json:"tags"` // Freeform tags: ["pii", "analytics", "deprecated"]

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
