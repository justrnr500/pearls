package pearl

import (
	"errors"
	"regexp"
	"strings"
)

var (
	// ErrEmptyNamespace is returned when the namespace is empty.
	ErrEmptyNamespace = errors.New("namespace cannot be empty")

	// ErrInvalidNamespace is returned when the namespace format is invalid.
	ErrInvalidNamespace = errors.New("invalid namespace format")

	// ErrInvalidSegment is returned when a namespace segment is invalid.
	ErrInvalidSegment = errors.New("invalid namespace segment")

	// namespaceSegmentPattern matches valid namespace segments.
	// Segments must start with a letter, contain only lowercase alphanumeric and underscores.
	namespaceSegmentPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
)

// ParseNamespace parses a dot-separated namespace string into segments.
// Example: "db.postgres.users" -> ["db", "postgres", "users"]
func ParseNamespace(ns string) ([]string, error) {
	if ns == "" {
		return nil, ErrEmptyNamespace
	}

	segments := strings.Split(ns, ".")
	for _, seg := range segments {
		if !isValidSegment(seg) {
			return nil, ErrInvalidSegment
		}
	}

	return segments, nil
}

// ValidateNamespace checks if a namespace string is valid.
func ValidateNamespace(ns string) error {
	_, err := ParseNamespace(ns)
	return err
}

// isValidSegment checks if a single namespace segment is valid.
func isValidSegment(seg string) bool {
	if seg == "" {
		return false
	}
	return namespaceSegmentPattern.MatchString(seg)
}

// JoinNamespace joins namespace segments with dots.
func JoinNamespace(segments ...string) string {
	return strings.Join(segments, ".")
}

// ParentNamespace returns the parent namespace.
// Example: "db.postgres.users" -> "db.postgres"
// Returns empty string if there's no parent.
func ParentNamespace(ns string) string {
	segments, err := ParseNamespace(ns)
	if err != nil || len(segments) <= 1 {
		return ""
	}
	return JoinNamespace(segments[:len(segments)-1]...)
}

// NamespaceDepth returns the depth of a namespace.
// Example: "db.postgres.users" -> 3
func NamespaceDepth(ns string) int {
	segments, err := ParseNamespace(ns)
	if err != nil {
		return 0
	}
	return len(segments)
}

// LastSegment returns the last segment of a namespace.
// Example: "db.postgres.users" -> "users"
func LastSegment(ns string) string {
	segments, err := ParseNamespace(ns)
	if err != nil || len(segments) == 0 {
		return ""
	}
	return segments[len(segments)-1]
}

// IsChildOf returns true if child is a direct or indirect child of parent.
// Example: IsChildOf("db.postgres.users", "db") -> true
func IsChildOf(child, parent string) bool {
	if parent == "" {
		return true
	}
	return strings.HasPrefix(child, parent+".")
}
