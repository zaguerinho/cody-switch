package assembler

import (
	"math"
	"testing"

	"github.com/zaguerinho/claude-switch/video-tutorial/internal/manifest"
	"github.com/zaguerinho/claude-switch/video-tutorial/internal/tts"
)

const floatTolerance = 1e-9

func approxEqual(a, b float64) bool {
	return math.Abs(a-b) < floatTolerance
}

func TestBuildSyncData_CorrectOffsets(t *testing.T) {
	m := &manifest.Manifest{
		Chapters: []manifest.Chapter{
			{
				ID:    "ch1",
				Title: "One",
				Segments: []manifest.Segment{
					{
						ID:        "s1",
						Narration: "Hello",
						Cue:       manifest.Cue{Type: "highlight", File: "a.py", Lines: [2]int{1, 5}, ScrollTo: 1},
					},
				},
			},
			{
				ID:    "ch2",
				Title: "Two",
				Segments: []manifest.Segment{
					{
						ID:        "s2",
						Narration: "World",
						Cue:       manifest.Cue{Type: "highlight", File: "a.py", Lines: [2]int{6, 10}, ScrollTo: 6},
					},
				},
			},
		},
	}
	synthResults := []tts.SynthResult{
		{
			ID:             "s1",
			WordTimestamps: []tts.WordTimestamp{{Word: "Hello", Start: 0, End: 0.5}},
			Duration:       0.6,
		},
		{
			ID:             "s2",
			WordTimestamps: []tts.WordTimestamp{{Word: "World", Start: 0, End: 0.5}},
			Duration:       0.6,
		},
	}

	result, err := BuildSyncData(m, synthResults, 0.3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Words) != 2 {
		t.Fatalf("expected 2 words, got %d", len(result.Words))
	}

	if result.Words[0].Word != "Hello" {
		t.Errorf("expected first word 'Hello', got %q", result.Words[0].Word)
	}
	if result.Words[0].Start != 0 {
		t.Errorf("expected first word start=0, got %f", result.Words[0].Start)
	}

	if result.Words[1].Word != "World" {
		t.Errorf("expected second word 'World', got %q", result.Words[1].Word)
	}
	if result.Words[1].Start <= 0.5 {
		t.Errorf("second word start should be > 0.5 (offset), got %f", result.Words[1].Start)
	}

	// Second word offset: segment 1 duration (0.6) + gap (0.3) = 0.9
	expectedStart := 0.6 + 0.3 // timeOffset after s1
	if !approxEqual(result.Words[1].Start, expectedStart) {
		t.Errorf("expected second word start~=%f, got %f", expectedStart, result.Words[1].Start)
	}

	if len(result.Chapters) != 2 {
		t.Fatalf("expected 2 chapters, got %d", len(result.Chapters))
	}
	if result.Chapters[0].StartWord != 0 {
		t.Errorf("expected chapter 0 startWord=0, got %d", result.Chapters[0].StartWord)
	}
	if result.Chapters[1].StartWord != 1 {
		t.Errorf("expected chapter 1 startWord=1, got %d", result.Chapters[1].StartWord)
	}

	if result.WordToCue[0] != "s1" {
		t.Errorf("expected wordToCue[0]='s1', got %q", result.WordToCue[0])
	}
	if result.WordToCue[1] != "s2" {
		t.Errorf("expected wordToCue[1]='s2', got %q", result.WordToCue[1])
	}

	// totalDuration = sum of (duration + gap) for each segment
	// s1: 0.6 + 0.3 = 0.9, s2: 0.6 + 0.3 = 0.9, total = 1.8
	if result.TotalDuration <= 1.0 {
		t.Errorf("totalDuration should account for gaps, got %f", result.TotalDuration)
	}
}

func TestBuildSyncData_EmptyManifest(t *testing.T) {
	m := &manifest.Manifest{
		Chapters: []manifest.Chapter{},
	}

	result, err := BuildSyncData(m, nil, 0.3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Words) != 0 {
		t.Errorf("expected 0 words, got %d", len(result.Words))
	}
	if len(result.Chapters) != 0 {
		t.Errorf("expected 0 chapters, got %d", len(result.Chapters))
	}
	if result.TotalDuration != 0 {
		t.Errorf("expected totalDuration=0, got %f", result.TotalDuration)
	}
}

func TestBuildSyncData_MissingResult(t *testing.T) {
	m := &manifest.Manifest{
		Chapters: []manifest.Chapter{
			{
				ID:    "ch1",
				Title: "One",
				Segments: []manifest.Segment{
					{ID: "s1", Narration: "Hello"},
				},
			},
		},
	}

	_, err := BuildSyncData(m, nil, 0.3)
	if err == nil {
		t.Fatal("expected error for missing synth result, got nil")
	}
}

func TestBuildCueMap(t *testing.T) {
	m := &manifest.Manifest{
		Chapters: []manifest.Chapter{
			{
				ID:    "ch1",
				Title: "Ch1",
				Segments: []manifest.Segment{
					{
						ID:        "s1",
						Narration: "A",
						Cue:       manifest.Cue{Type: "highlight", File: "a.py", Lines: [2]int{1, 5}, ScrollTo: 1},
					},
					{
						ID:        "s2",
						Narration: "B",
						Cue:       manifest.Cue{Type: "reveal", File: "b.py", Lines: [2]int{10, 20}, ScrollTo: 10},
					},
				},
			},
		},
	}

	cueMap := BuildCueMap(m)

	if len(cueMap) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(cueMap))
	}

	s1Cue, ok := cueMap["s1"]
	if !ok {
		t.Fatal("missing cue for s1")
	}
	if s1Cue.File != "a.py" {
		t.Errorf("expected s1 file='a.py', got %q", s1Cue.File)
	}

	s2Cue, ok := cueMap["s2"]
	if !ok {
		t.Fatal("missing cue for s2")
	}
	if s2Cue.Type != "reveal" {
		t.Errorf("expected s2 type='reveal', got %q", s2Cue.Type)
	}
	if s2Cue.Lines != [2]int{10, 20} {
		t.Errorf("expected s2 lines=[10,20], got %v", s2Cue.Lines)
	}
}

func TestBuildAudioSegments(t *testing.T) {
	m := &manifest.Manifest{
		Chapters: []manifest.Chapter{
			{
				ID:    "ch1",
				Title: "Ch1",
				Segments: []manifest.Segment{
					{ID: "s1", Narration: "A"},
					{ID: "s2", Narration: "B"},
				},
			},
		},
	}
	synthResults := []tts.SynthResult{
		{ID: "s1", AudioBase64: "AAAA", Duration: 1.0},
		{ID: "s2", AudioBase64: "BBBB", Duration: 2.0},
	}

	segments, err := BuildAudioSegments(m, synthResults)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(segments) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(segments))
	}
	if segments[0].ID != "s1" || segments[0].AudioBase64 != "AAAA" {
		t.Errorf("segment 0 mismatch: %+v", segments[0])
	}
	if segments[1].ID != "s2" || segments[1].Duration != 2.0 {
		t.Errorf("segment 1 mismatch: %+v", segments[1])
	}
}

func TestBuildAudioSegments_MissingResult(t *testing.T) {
	m := &manifest.Manifest{
		Chapters: []manifest.Chapter{
			{
				ID:    "ch1",
				Title: "Ch1",
				Segments: []manifest.Segment{
					{ID: "s1", Narration: "A"},
				},
			},
		},
	}

	_, err := BuildAudioSegments(m, nil)
	if err == nil {
		t.Fatal("expected error for missing audio, got nil")
	}
}
