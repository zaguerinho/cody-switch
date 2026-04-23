package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// FlattenSegments
// ---------------------------------------------------------------------------

func TestFlattenSegments(t *testing.T) {
	t.Run("flattens chapters into ordered segment array", func(t *testing.T) {
		m := &Manifest{
			Chapters: []Chapter{
				{ID: "ch1", Title: "A", Segments: []Segment{
					{ID: "s1", Narration: "First", Cue: Cue{File: "a.py", Lines: [2]int{1, 5}}},
					{ID: "s2", Narration: "Second", Cue: Cue{File: "a.py", Lines: [2]int{6, 10}}},
				}},
				{ID: "ch2", Title: "B", Segments: []Segment{
					{ID: "s3", Narration: "Third", Cue: Cue{File: "b.py", Lines: [2]int{1, 3}}},
				}},
			},
		}

		flat := m.FlattenSegments()

		if len(flat) != 3 {
			t.Fatalf("expected 3 segments, got %d", len(flat))
		}
		if flat[0].ID != "s1" || flat[0].Text != "First" || flat[0].ChapterID != "ch1" {
			t.Errorf("flat[0] = %+v", flat[0])
		}
		if flat[2].ID != "s3" || flat[2].ChapterID != "ch2" {
			t.Errorf("flat[2] = %+v", flat[2])
		}
	})

	t.Run("handles empty manifest", func(t *testing.T) {
		m := &Manifest{Chapters: []Chapter{}}
		flat := m.FlattenSegments()
		if len(flat) != 0 {
			t.Fatalf("expected 0 segments, got %d", len(flat))
		}
	})
}

// ---------------------------------------------------------------------------
// Validate
// ---------------------------------------------------------------------------

func TestValidate(t *testing.T) {
	// Helper: create a temp file with content and return its path.
	tmpFile := func(t *testing.T, name, content string) string {
		t.Helper()
		dir := t.TempDir()
		fp := filepath.Join(dir, name)
		if err := os.WriteFile(fp, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		return fp
	}

	t.Run("passes valid manifest", func(t *testing.T) {
		src := tmpFile(t, "test.py", "line1\nline2\nline3\nline4\nline5\n")

		m := &Manifest{
			Title: "Test Tutorial",
			Chapters: []Chapter{{
				ID: "ch1", Title: "Chapter 1",
				Segments: []Segment{{
					ID: "ch1-s1", Narration: "Hello",
					Cue: Cue{Type: "highlight", File: src, Lines: [2]int{1, 5}, ScrollTo: 1},
				}},
			}},
		}

		if err := Validate(m, []string{src}); err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})

	t.Run("fails on missing title", func(t *testing.T) {
		m := &Manifest{
			Chapters: []Chapter{{ID: "ch1", Title: "X", Segments: []Segment{}}},
		}
		err := Validate(m, nil)
		if err == nil || !strings.Contains(strings.ToLower(err.Error()), "title") {
			t.Errorf("expected error about title, got: %v", err)
		}
	})

	t.Run("fails on empty chapters", func(t *testing.T) {
		m := &Manifest{Title: "T", Chapters: []Chapter{}}
		err := Validate(m, nil)
		if err == nil || !strings.Contains(strings.ToLower(err.Error()), "chapters") {
			t.Errorf("expected error about chapters, got: %v", err)
		}
	})

	t.Run("fails on out-of-bounds line range", func(t *testing.T) {
		src := tmpFile(t, "small.py", "line1\nline2\n")

		m := &Manifest{
			Title: "Test",
			Chapters: []Chapter{{
				ID: "ch1", Title: "Ch1",
				Segments: []Segment{{
					ID: "s1", Narration: "Hi",
					Cue: Cue{Type: "highlight", File: src, Lines: [2]int{1, 99}, ScrollTo: 1},
				}},
			}},
		}

		err := Validate(m, []string{src})
		if err == nil || !strings.Contains(strings.ToLower(err.Error()), "exceeds file length") {
			t.Errorf("expected error about line exceeds, got: %v", err)
		}
	})

	t.Run("fails on nonexistent file reference", func(t *testing.T) {
		src := tmpFile(t, "real.py", "line1\nline2\nline3\n")

		m := &Manifest{
			Title: "Test",
			Chapters: []Chapter{{
				ID: "ch1", Title: "Ch1",
				Segments: []Segment{{
					ID: "s1", Narration: "Hi",
					Cue: Cue{Type: "highlight", File: "nonexistent.py", Lines: [2]int{1, 5}, ScrollTo: 1},
				}},
			}},
		}

		err := Validate(m, []string{src})
		if err == nil || !strings.Contains(strings.ToLower(err.Error()), "not found") {
			t.Errorf("expected error about not found, got: %v", err)
		}
	})

	t.Run("fails on invalid line range start > end", func(t *testing.T) {
		src := tmpFile(t, "test.py", "line1\nline2\nline3\n")

		m := &Manifest{
			Title: "Test",
			Chapters: []Chapter{{
				ID: "ch1", Title: "Ch1",
				Segments: []Segment{{
					ID: "s1", Narration: "Hi",
					Cue: Cue{Type: "highlight", File: src, Lines: [2]int{5, 2}, ScrollTo: 5},
				}},
			}},
		}

		err := Validate(m, []string{src})
		if err == nil || !strings.Contains(strings.ToLower(err.Error()), "invalid line range") {
			t.Errorf("expected error about invalid line range, got: %v", err)
		}
	})

	t.Run("resolves file by basename", func(t *testing.T) {
		src := tmpFile(t, "app.py", "line1\nline2\nline3\n")

		m := &Manifest{
			Title: "Test",
			Chapters: []Chapter{{
				ID: "ch1", Title: "Ch1",
				Segments: []Segment{{
					ID: "s1", Narration: "Hi",
					Cue: Cue{Type: "highlight", File: "app.py", Lines: [2]int{1, 3}, ScrollTo: 1},
				}},
			}},
		}

		if err := Validate(m, []string{src}); err != nil {
			t.Errorf("expected basename resolution to work, got: %v", err)
		}
	})
}
