// Package assembler builds self-contained tutorial HTML files from a manifest,
// TTS synthesis results, source code, and embedded browser assets. It is a Go
// port of assemble.js from the video-tutorial Node.js implementation.
package assembler

import (
	"fmt"

	"github.com/zaguerinho/claude-switch/video-tutorial/internal/manifest"
	"github.com/zaguerinho/claude-switch/video-tutorial/internal/tts"
)

// SyncWord is a word with globally-offset timestamps.
type SyncWord struct {
	Word  string  `json:"word"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

// SyncChapter has the chapter metadata plus the index of its first word.
type SyncChapter struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	StartWord int    `json:"startWord"`
}

// SyncData holds the globally-aligned word timestamps and chapter mapping.
type SyncData struct {
	Words         []SyncWord    `json:"words"`
	WordToCue     []string      `json:"wordToCue"`
	Chapters      []SyncChapter `json:"chapters"`
	TotalDuration float64       `json:"totalDuration"`
}

// AudioSegment is the data for a single audio segment in the output.
type AudioSegment struct {
	ID          string  `json:"id"`
	AudioBase64 string  `json:"audioBase64"`
	Duration    float64 `json:"duration"`
}

// BuildSyncData builds the global word timestamps array from per-segment synthesis results.
// Offsets each segment's timestamps so they're globally monotonic.
// segmentGapSec is the gap in seconds between segments (typically 0.3).
func BuildSyncData(m *manifest.Manifest, synthResults []tts.SynthResult, segmentGapSec float64) (*SyncData, error) {
	// Build a map of segment ID -> synth result for fast lookup.
	resultMap := make(map[string]*tts.SynthResult, len(synthResults))
	for i := range synthResults {
		resultMap[synthResults[i].ID] = &synthResults[i]
	}

	var words []SyncWord
	var wordToCue []string
	var chapters []SyncChapter
	var timeOffset float64

	for _, ch := range m.Chapters {
		chapters = append(chapters, SyncChapter{
			ID:        ch.ID,
			Title:     ch.Title,
			StartWord: len(words),
		})

		for _, seg := range ch.Segments {
			result, ok := resultMap[seg.ID]
			if !ok {
				return nil, fmt.Errorf("assembler: no synthesis result for segment %q", seg.ID)
			}

			for _, w := range result.WordTimestamps {
				words = append(words, SyncWord{
					Word:  w.Word,
					Start: w.Start + timeOffset,
					End:   w.End + timeOffset,
				})
				wordToCue = append(wordToCue, seg.ID)
			}

			timeOffset += result.Duration + segmentGapSec
		}
	}

	// Ensure non-nil slices for JSON marshalling.
	if words == nil {
		words = []SyncWord{}
	}
	if wordToCue == nil {
		wordToCue = []string{}
	}
	if chapters == nil {
		chapters = []SyncChapter{}
	}

	return &SyncData{
		Words:         words,
		WordToCue:     wordToCue,
		Chapters:      chapters,
		TotalDuration: timeOffset,
	}, nil
}

// BuildCueMap maps segment IDs to their visual cues.
func BuildCueMap(m *manifest.Manifest) map[string]manifest.Cue {
	cueMap := make(map[string]manifest.Cue)
	for _, ch := range m.Chapters {
		for _, seg := range ch.Segments {
			cueMap[seg.ID] = seg.Cue
		}
	}
	return cueMap
}

// BuildAudioSegments extracts audio data ordered by manifest segment order.
func BuildAudioSegments(m *manifest.Manifest, synthResults []tts.SynthResult) ([]AudioSegment, error) {
	resultMap := make(map[string]*tts.SynthResult, len(synthResults))
	for i := range synthResults {
		resultMap[synthResults[i].ID] = &synthResults[i]
	}

	var segments []AudioSegment
	for _, ch := range m.Chapters {
		for _, seg := range ch.Segments {
			result, ok := resultMap[seg.ID]
			if !ok {
				return nil, fmt.Errorf("assembler: no audio for segment %q", seg.ID)
			}
			segments = append(segments, AudioSegment{
				ID:          seg.ID,
				AudioBase64: result.AudioBase64,
				Duration:    result.Duration,
			})
		}
	}
	return segments, nil
}
