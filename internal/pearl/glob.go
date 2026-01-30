package pearl

import (
	"github.com/bmatcuk/doublestar/v4"
)

// MatchPath checks if a file path matches any of the given glob patterns.
// Patterns use doublestar syntax (** matches any depth).
// Paths are relative to repo root.
func MatchPath(path string, globs []string) bool {
	if path == "" || len(globs) == 0 {
		return false
	}

	for _, g := range globs {
		matched, err := doublestar.Match(g, path)
		if err == nil && matched {
			return true
		}
	}
	return false
}
