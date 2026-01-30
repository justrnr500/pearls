package pearl

import "testing"

func TestMatchPath(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		globs []string
		want  bool
	}{
		{
			name:  "single glob match",
			path:  "src/payments/checkout.ts",
			globs: []string{"src/payments/checkout.ts"},
			want:  true,
		},
		{
			name:  "multiple globs any match is true",
			path:  "src/payments/checkout.ts",
			globs: []string{"docs/**", "src/payments/**"},
			want:  true,
		},
		{
			name:  "doublestar matches single level",
			path:  "src/payments/checkout.ts",
			globs: []string{"src/payments/**"},
			want:  true,
		},
		{
			name:  "doublestar matches nested levels",
			path:  "src/payments/stripe/webhook.ts",
			globs: []string{"src/payments/**"},
			want:  true,
		},
		{
			name:  "extension pattern matches",
			path:  "src/payments/checkout.ts",
			globs: []string{"**/*.ts"},
			want:  true,
		},
		{
			name:  "extension pattern matches nested",
			path:  "src/payments/stripe/webhook.ts",
			globs: []string{"**/*.ts"},
			want:  true,
		},
		{
			name:  "no match returns false",
			path:  "src/payments/checkout.ts",
			globs: []string{"docs/**", "**/*.go"},
			want:  false,
		},
		{
			name:  "empty globs returns false",
			path:  "src/payments/checkout.ts",
			globs: []string{},
			want:  false,
		},
		{
			name:  "nil globs returns false",
			path:  "src/payments/checkout.ts",
			globs: nil,
			want:  false,
		},
		{
			name:  "empty path returns false",
			path:  "",
			globs: []string{"**/*.ts"},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchPath(tt.path, tt.globs)
			if got != tt.want {
				t.Errorf("MatchPath(%q, %v) = %v, want %v", tt.path, tt.globs, got, tt.want)
			}
		})
	}
}
