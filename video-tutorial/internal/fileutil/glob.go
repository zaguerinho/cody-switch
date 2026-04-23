// Package fileutil provides filesystem helpers for the video-tutorial pipeline:
// safe glob expansion (no shell exec), slugification, and language detection.
package fileutil

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// ExpandGlob expands a glob pattern relative to baseDir and returns absolute
// paths sorted alphabetically. It uses doublestar for recursive patterns
// (e.g. "**/*.py") and falls back to filepath.Glob for simple patterns.
//
// Returns an error (not an empty slice) if the pattern is syntactically invalid.
func ExpandGlob(pattern, baseDir string) ([]string, error) {
	// Ensure baseDir is absolute so results are absolute.
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("resolving base dir: %w", err)
	}

	fullPattern := filepath.Join(absBase, pattern)

	var matches []string
	if strings.Contains(pattern, "**") {
		matches, err = doublestar.FilepathGlob(fullPattern)
	} else {
		matches, err = filepath.Glob(fullPattern)
	}
	if err != nil {
		return nil, fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
	}

	// Resolve every match to an absolute path (filepath.Glob and doublestar
	// already return paths relative to the pattern root, which is absolute
	// here, but this ensures consistency).
	abs := make([]string, 0, len(matches))
	for _, m := range matches {
		a, err := filepath.Abs(m)
		if err != nil {
			continue
		}
		abs = append(abs, a)
	}

	sort.Strings(abs)
	return abs, nil
}

// slugifyRe matches runs of non-alphanumeric characters.
var slugifyRe = regexp.MustCompile(`[^a-z0-9]+`)

// trimHyphens strips leading and trailing hyphens.
var trimHyphens = regexp.MustCompile(`^-+|-+$`)

// Slugify converts arbitrary text into a URL/filesystem-safe slug.
// It lowercases, replaces non-alphanumeric runs with hyphens, trims leading
// and trailing hyphens, and truncates to 50 characters.
//
// This is a direct port of the slugify function from build-tutorial.js.
func Slugify(text string) string {
	s := strings.ToLower(text)
	s = slugifyRe.ReplaceAllString(s, "-")
	s = trimHyphens.ReplaceAllString(s, "")
	if len(s) > 50 {
		s = s[:50]
		// Avoid trailing hyphen after truncation.
		s = strings.TrimRight(s, "-")
	}
	return s
}
