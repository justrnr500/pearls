package pearl

import (
	"encoding/json"
	"testing"
	"time"
)

func TestAssetTypeIsValid(t *testing.T) {
	tests := []struct {
		input AssetType
		want  bool
	}{
		{TypeTable, true},
		{TypeSchema, true},
		{TypeDatabase, true},
		{TypeAPI, true},
		{TypeEndpoint, true},
		{TypeFile, true},
		{TypeBucket, true},
		{TypePipeline, true},
		{TypeDashboard, true},
		{TypeQuery, true},
		{TypeCustom, true},
		{AssetType("invalid"), false},
		{AssetType(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			got := tt.input.IsValid()
			if got != tt.want {
				t.Errorf("AssetType(%q).IsValid() = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestStatusIsValid(t *testing.T) {
	tests := []struct {
		input Status
		want  bool
	}{
		{StatusActive, true},
		{StatusDeprecated, true},
		{StatusArchived, true},
		{Status("invalid"), false},
		{Status(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			got := tt.input.IsValid()
			if got != tt.want {
				t.Errorf("Status(%q).IsValid() = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestPearlFullID(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		pearlName string
		want      string
	}{
		{"with namespace", "db.postgres", "users", "db.postgres.users"},
		{"no namespace", "", "standalone", "standalone"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pearl{
				Namespace: tt.namespace,
				Name:      tt.pearlName,
			}
			got := p.FullID()
			if got != tt.want {
				t.Errorf("Pearl.FullID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPearlJSONMarshal(t *testing.T) {
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	p := &Pearl{
		ID:          "db.postgres.users",
		Name:        "users",
		Namespace:   "db.postgres",
		Type:        TypeTable,
		Tags:        []string{"pii", "core"},
		Description: "Core user account information",
		ContentPath: "content/db/postgres/users.md",
		ContentHash: "abc123",
		References:  []string{"db.postgres.organizations"},
		Connection: &ConnectionInfo{
			Type:     "postgres",
			Host:     "${DB_HOST}",
			Port:     5432,
			Database: "myapp",
			Schema:   "public",
		},
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: "agent",
		Status:    StatusActive,
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded Pearl
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.ID != p.ID {
		t.Errorf("ID = %q, want %q", decoded.ID, p.ID)
	}
	if decoded.Type != p.Type {
		t.Errorf("Type = %q, want %q", decoded.Type, p.Type)
	}
	if decoded.Status != p.Status {
		t.Errorf("Status = %q, want %q", decoded.Status, p.Status)
	}
	if decoded.Connection == nil {
		t.Fatal("Connection is nil")
	}
	if decoded.Connection.Port != 5432 {
		t.Errorf("Connection.Port = %d, want 5432", decoded.Connection.Port)
	}
	if len(decoded.Tags) != 2 {
		t.Errorf("len(Tags) = %d, want 2", len(decoded.Tags))
	}
}

func TestPearlJSONOmitEmpty(t *testing.T) {
	p := &Pearl{
		ID:          "standalone",
		Name:        "standalone",
		Type:        TypeQuery,
		Description: "A standalone query",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Status:      StatusActive,
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Check that omitempty fields are not present
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if _, ok := raw["connection"]; ok {
		t.Error("connection should be omitted when nil")
	}
	if _, ok := raw["references"]; ok {
		t.Error("references should be omitted when empty")
	}
	if _, ok := raw["parent"]; ok {
		t.Error("parent should be omitted when empty")
	}
}
