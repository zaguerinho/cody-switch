package tts

// SynthResult holds the output of synthesizing a single segment.
type SynthResult struct {
	ID             string         `json:"id"`
	AudioBase64    string         `json:"audioBase64"`
	WordTimestamps []WordTimestamp `json:"wordTimestamps"`
	Duration       float64        `json:"duration"`
}

// SegmentInput is a segment to be synthesized.
type SegmentInput struct {
	ID   string
	Text string
}

// ProgressFunc is called after each segment completes.
type ProgressFunc func(completed, total int, segmentID string)

// ResultFunc is called with each result for incremental saving.
type ResultFunc func(result SynthResult)

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
