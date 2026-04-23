// Package manifest defines the domain types for a video-tutorial scene manifest.
// The types mirror the JSON schema produced by the Claude API content generator
// and consumed by the assembler and TTS synthesizer.
package manifest

// Manifest is the top-level scene manifest produced by the content generator.
type Manifest struct {
	Title     string    `json:"title"`
	Chapters  []Chapter `json:"chapters"`
	SourceDir string    `json:"_sourceDir,omitempty"`
}

// Chapter groups related segments under a titled section.
type Chapter struct {
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	Segments []Segment `json:"segments"`
}

// Segment is the atomic unit of a tutorial: one narration paired with one
// visual cue pointing at a specific region of source code.
type Segment struct {
	ID        string `json:"id"`
	Narration string `json:"narration"`
	Cue       Cue    `json:"cue"`
}

// Cue describes which source code to display during a segment.
type Cue struct {
	Type     string `json:"type"`
	File     string `json:"file"`
	Lines    [2]int `json:"lines"`
	ScrollTo int    `json:"scroll_to"`
}

// FlatSegment is a flattened segment with chapter context, used for synthesis.
// It decouples the TTS pipeline from the nested chapter/segment hierarchy.
type FlatSegment struct {
	ID        string
	Text      string
	ChapterID string
	Cue       Cue
}

// FlattenSegments walks the chapter tree and returns an ordered slice of
// FlatSegment values ready for sequential TTS synthesis. This is a direct
// port of flattenSegments from generate-manifest.js.
func (m *Manifest) FlattenSegments() []FlatSegment {
	var flat []FlatSegment
	for _, ch := range m.Chapters {
		for _, seg := range ch.Segments {
			flat = append(flat, FlatSegment{
				ID:        seg.ID,
				Text:      seg.Narration,
				ChapterID: ch.ID,
				Cue:       seg.Cue,
			})
		}
	}
	return flat
}
