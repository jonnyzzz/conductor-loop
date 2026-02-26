package runner

import (
	"os"
	"path/filepath"
	"strings"
)

// projectRootMarkers are file/directory names that heuristically identify a
// project root. The list is checked in order; the first match wins.
var projectRootMarkers = []string{
	".git",
	"go.mod",
	"package.json",
	"pyproject.toml",
	"Cargo.toml",
	"build.gradle",
	"pom.xml",
	"Makefile",
	"README.md",
	"README.rst",
	"README",
}

// FindProjectRoot walks upward from dir (or CWD when dir is empty) and returns
// the nearest ancestor directory (inclusive) that contains a known project root
// marker. Returns empty string when no marker is found.
func FindProjectRoot(dir string) string {
	start := strings.TrimSpace(dir)
	if start == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return ""
		}
		start = cwd
	}
	current := filepath.Clean(start)
	for {
		for _, marker := range projectRootMarkers {
			if _, err := os.Lstat(filepath.Join(current, marker)); err == nil {
				return current
			}
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return ""
}

// SuggestProjectID converts an arbitrary name (typically a directory base name)
// into a valid project ID: lowercase, alphanumeric and hyphens only, no leading
// or trailing hyphens, consecutive separators collapsed.
func SuggestProjectID(name string) string {
	b := make([]byte, 0, len(name))
	prevHyphen := false
	for _, r := range name {
		var c byte
		switch {
		case r >= 'a' && r <= 'z':
			c = byte(r)
		case r >= 'A' && r <= 'Z':
			c = byte(r) + 32 // toLower
		case r >= '0' && r <= '9':
			c = byte(r)
		default:
			c = '-'
		}
		if c == '-' {
			if prevHyphen || len(b) == 0 {
				continue
			}
			prevHyphen = true
		} else {
			prevHyphen = false
		}
		b = append(b, c)
	}
	result := strings.TrimRight(string(b), "-")
	return result
}
