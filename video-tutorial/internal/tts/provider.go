package tts

import (
	"context"
	"fmt"
	"time"
)

// Provider is the interface that TTS backends must implement.
// Both ElevenLabs (cloud) and Piper (local) implement this.
type Provider interface {
	// SynthesizeSegment synthesizes a single text segment and returns
	// audio as base64-encoded data with word-level timestamps.
	SynthesizeSegment(ctx context.Context, text string) (*SynthResult, error)

	// Name returns a human-readable name for the provider (e.g., "ElevenLabs", "Piper").
	Name() string
}

// SynthesizeAll synthesizes multiple segments sequentially using the given provider.
// Calls onProgress and onResult after each segment completes.
// segmentGapMS is the delay between segments (for rate limiting with cloud APIs;
// can be 0 for local providers).
func SynthesizeAll(
	ctx context.Context,
	provider Provider,
	segments []SegmentInput,
	segmentGapMS int,
	onProgress ProgressFunc,
	onResult ResultFunc,
) ([]SynthResult, error) {
	var results []SynthResult

	for i, seg := range segments {
		result, err := provider.SynthesizeSegment(ctx, seg.Text)
		if err != nil {
			return results, fmt.Errorf("segment %s: %w", seg.ID, err)
		}
		result.ID = seg.ID
		results = append(results, *result)

		if onProgress != nil {
			onProgress(i+1, len(segments), seg.ID)
		}
		if onResult != nil {
			onResult(*result)
		}

		// Gap between segments (rate limiting for cloud APIs)
		if segmentGapMS > 0 && i < len(segments)-1 {
			gap := time.Duration(segmentGapMS) * time.Millisecond
			select {
			case <-ctx.Done():
				return results, ctx.Err()
			case <-time.After(gap):
			}
		}
	}

	return results, nil
}
