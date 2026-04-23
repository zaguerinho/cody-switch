package manifest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidationError holds one or more validation failures.
type ValidationError struct {
	Errors []string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("manifest validation failed:\n%s",
		strings.Join(prefixLines(e.Errors), "\n"))
}

func prefixLines(lines []string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = "  - " + l
	}
	return out
}

// Validate checks a manifest against the actual source files on disk.
// It verifies: required fields, file existence, and line range validity.
// Port of validateManifest from generate-manifest.js.
func Validate(m *Manifest, filePaths []string) error {
	var errs []string

	if m.Title == "" {
		errs = append(errs, `missing or invalid "title" field`)
	}
	if len(m.Chapters) == 0 {
		errs = append(errs, `missing or empty "chapters" array`)
	}

	// Build map of file paths → line counts
	fileLinesMap := make(map[string]int)
	for _, fp := range filePaths {
		content, err := os.ReadFile(fp)
		if err != nil {
			continue
		}
		lineCount := len(strings.Split(string(content), "\n"))
		fileLinesMap[fp] = lineCount
		fileLinesMap[filepath.Base(fp)] = lineCount
	}

	for _, ch := range m.Chapters {
		if ch.ID == "" || ch.Title == "" {
			errs = append(errs, fmt.Sprintf("chapter missing id or title: %s", truncateJSON(ch)))
			continue
		}
		for _, seg := range ch.Segments {
			if seg.ID == "" || seg.Narration == "" {
				errs = append(errs, fmt.Sprintf("segment missing id or narration in chapter %s", ch.ID))
				continue
			}
			cue := seg.Cue
			if cue.File == "" || (cue.Lines == [2]int{}) {
				errs = append(errs, fmt.Sprintf("segment %s: missing cue, file, or lines", seg.ID))
				continue
			}

			// Check file exists in the source file set
			lineCount, found := fileLinesMap[cue.File]
			if !found {
				lineCount, found = fileLinesMap[filepath.Base(cue.File)]
			}
			if !found {
				errs = append(errs, fmt.Sprintf("segment %s: file %q not found in source files", seg.ID, cue.File))
				continue
			}

			// Check line range
			start, end := cue.Lines[0], cue.Lines[1]
			if start < 1 || end < start {
				errs = append(errs, fmt.Sprintf("segment %s: invalid line range [%d, %d]", seg.ID, start, end))
			}
			if end > lineCount {
				errs = append(errs, fmt.Sprintf("segment %s: line %d exceeds file length (%d lines) in %q",
					seg.ID, end, lineCount, cue.File))
			}
		}
	}

	if len(errs) > 0 {
		return &ValidationError{Errors: errs}
	}
	return nil
}

func truncateJSON(v interface{}) string {
	s := fmt.Sprintf("%+v", v)
	if len(s) > 100 {
		return s[:100] + "..."
	}
	return s
}
