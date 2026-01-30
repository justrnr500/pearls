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
		// Built-in constants
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
		// Free-form types (new)
		{AssetType("convention"), true},
		{AssetType("brainstorm"), true},
		{AssetType("runbook"), true},
		{AssetType("decision"), true},
		{AssetType("my-script"), true},
		// Invalid
		{AssetType(""), false},
		{AssetType("UPPER"), false},
		{AssetType("has spaces"), false},
		{AssetType("has_underscore"), false},
		{AssetType("123start"), false},
		{AssetType("-leading-hyphen"), false},
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

func TestValidateScopes(t *testing.T) {
	tests := []struct {
		name    string
		scopes  []string
		wantErr bool
	}{
		{"valid scopes", []string{"payments", "auth", "stripe"}, false},
		{"valid with hyphens", []string{"error-handling", "api-v2"}, false},
		{"empty list", []string{}, false},
		{"nil list", nil, false},
		{"invalid uppercase", []string{"Payments"}, true},
		{"invalid spaces", []string{"my scope"}, true},
		{"invalid empty string", []string{""}, true},
		{"invalid underscore", []string{"my_scope"}, true},
		{"mixed valid and invalid", []string{"payments", "BAD"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateScopes(tt.scopes)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateScopes(%v) error = %v, wantErr %v", tt.scopes, err, tt.wantErr)
			}
		})
	}
}

func TestValidateGlobs(t *testing.T) {
	tests := []struct {
		name    string
		globs   []string
		wantErr bool
	}{
		{"valid single glob", []string{"src/payments/**"}, false},
		{"valid multiple globs", []string{"src/payments/**/*.ts", "src/billing/**"}, false},
		{"valid simple pattern", []string{"*.go"}, false},
		{"empty list", []string{}, false},
		{"nil list", nil, false},
		{"invalid empty string", []string{""}, true},
		{"invalid bad pattern", []string{"["}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGlobs(tt.globs)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateGlobs(%v) error = %v, wantErr %v", tt.globs, err, tt.wantErr)
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
		Globs:       []string{"src/models/user/**", "src/db/users/**"},
		Scopes:      []string{"users", "auth"},
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
	if len(decoded.Globs) != 2 {
		t.Errorf("len(Globs) = %d, want 2", len(decoded.Globs))
	}
	if len(decoded.Scopes) != 2 {
		t.Errorf("len(Scopes) = %d, want 2", len(decoded.Scopes))
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
	if _, ok := raw["globs"]; ok {
		t.Error("globs should be omitted when empty")
	}
	if _, ok := raw["scopes"]; ok {
		t.Error("scopes should be omitted when empty")
	}
}
