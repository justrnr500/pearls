package pearl

import (
	"testing"
)

func TestParseNamespace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		want     []string
		wantErr  error
	}{
		{
			name:  "simple namespace",
			input: "db",
			want:  []string{"db"},
		},
		{
			name:  "two segments",
			input: "db.postgres",
			want:  []string{"db", "postgres"},
		},
		{
			name:  "three segments",
			input: "db.postgres.users",
			want:  []string{"db", "postgres", "users"},
		},
		{
			name:  "with underscore",
			input: "db.my_database.user_accounts",
			want:  []string{"db", "my_database", "user_accounts"},
		},
		{
			name:  "with numbers",
			input: "db.postgres15.v2_users",
			want:  []string{"db", "postgres15", "v2_users"},
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: ErrEmptyNamespace,
		},
		{
			name:    "starts with number",
			input:   "1db.postgres",
			wantErr: ErrInvalidSegment,
		},
		{
			name:    "uppercase",
			input:   "DB.Postgres",
			wantErr: ErrInvalidSegment,
		},
		{
			name:    "empty segment",
			input:   "db..postgres",
			wantErr: ErrInvalidSegment,
		},
		{
			name:    "trailing dot",
			input:   "db.postgres.",
			wantErr: ErrInvalidSegment,
		},
		{
			name:    "leading dot",
			input:   ".db.postgres",
			wantErr: ErrInvalidSegment,
		},
		{
			name:    "special characters",
			input:   "db.postgres-users",
			wantErr: ErrInvalidSegment,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseNamespace(tt.input)
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("ParseNamespace(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseNamespace(%q) unexpected error = %v", tt.input, err)
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("ParseNamespace(%q) = %v, want %v", tt.input, got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ParseNamespace(%q) = %v, want %v", tt.input, got, tt.want)
					return
				}
			}
		})
	}
}

func TestParentNamespace(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"db.postgres.users", "db.postgres"},
		{"db.postgres", "db"},
		{"db", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParentNamespace(tt.input)
			if got != tt.want {
				t.Errorf("ParentNamespace(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNamespaceDepth(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"db.postgres.users", 3},
		{"db.postgres", 2},
		{"db", 1},
		{"", 0},
		{"Invalid", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NamespaceDepth(tt.input)
			if got != tt.want {
				t.Errorf("NamespaceDepth(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestLastSegment(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"db.postgres.users", "users"},
		{"db.postgres", "postgres"},
		{"db", "db"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := LastSegment(tt.input)
			if got != tt.want {
				t.Errorf("LastSegment(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsChildOf(t *testing.T) {
	tests := []struct {
		child  string
		parent string
		want   bool
	}{
		{"db.postgres.users", "db", true},
		{"db.postgres.users", "db.postgres", true},
		{"db.postgres.users", "db.postgres.users", false},
		{"db.postgres", "db.postgres.users", false},
		{"db.postgres", "db", true},
		{"db", "db", false},
		{"db.postgres", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.child+"_"+tt.parent, func(t *testing.T) {
			got := IsChildOf(tt.child, tt.parent)
			if got != tt.want {
				t.Errorf("IsChildOf(%q, %q) = %v, want %v", tt.child, tt.parent, got, tt.want)
			}
		})
	}
}
