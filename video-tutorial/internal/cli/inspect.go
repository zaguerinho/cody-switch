package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/zaguerinho/claude-switch/video-tutorial/internal/manifest"
	"github.com/zaguerinho/claude-switch/video-tutorial/internal/tts"
)

func newInspectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "inspect <slug>",
		Short: "Inspect a cached tutorial build",
		Long:  "Inspect the cached build artifacts for a tutorial slug, showing\nmanifest details, audio file status, and output info.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			outputBase := resolveOutputBase()
			tutorialDir := filepath.Join(outputBase, slug)

			// 1. Load manifest.json.
			manifestPath := filepath.Join(tutorialDir, "manifest.json")
			manifestData, err := os.ReadFile(manifestPath)
			if err != nil {
				return fmt.Errorf("reading manifest: %w\nExpected at: %s", err, manifestPath)
			}
			var m manifest.Manifest
			if err := json.Unmarshal(manifestData, &m); err != nil {
				return fmt.Errorf("parsing manifest: %w", err)
			}

			// Count segments.
			segCount := 0
			for _, ch := range m.Chapters {
				segCount += len(ch.Segments)
			}

			// Estimate duration: ~150 words per minute, ~2 words per sentence segment.
			// Use word count in narration as a rough estimate.
			totalWords := 0
			for _, ch := range m.Chapters {
				for _, seg := range ch.Segments {
					totalWords += wordCount(seg.Narration)
				}
			}
			estimatedMinutes := float64(totalWords) / 150.0

			// 3. Print metadata.
			fmt.Fprintf(os.Stderr, "Title:      %s\n", m.Title)
			fmt.Fprintf(os.Stderr, "Slug:       %s\n", slug)
			fmt.Fprintf(os.Stderr, "Chapters:   %d\n", len(m.Chapters))
			fmt.Fprintf(os.Stderr, "Segments:   %d\n", segCount)
			fmt.Fprintf(os.Stderr, "Est. duration: %.1f min (~%d words)\n", estimatedMinutes, totalWords)
			if m.SourceDir != "" {
				fmt.Fprintf(os.Stderr, "Source dir: %s\n", m.SourceDir)
			}

			// 4. List chapters with titles.
			fmt.Fprintf(os.Stderr, "\nChapters:\n")
			for i, ch := range m.Chapters {
				fmt.Fprintf(os.Stderr, "  %d. [%s] %s (%d segments)\n",
					i+1, ch.ID, ch.Title, len(ch.Segments))
			}

			// 2 & 5. Check if synth-results.json exists.
			synthPath := filepath.Join(tutorialDir, "synth-results.json")
			synthData, err := os.ReadFile(synthPath)
			if err == nil {
				var results []tts.SynthResult
				if err := json.Unmarshal(synthData, &results); err == nil && len(results) > 0 {
					// Calculate total audio duration and file size.
					var totalDuration float64
					var totalAudioSize int
					for _, r := range results {
						totalDuration += r.Duration
						totalAudioSize += len(r.AudioBase64)
					}

					fmt.Fprintf(os.Stderr, "\nAudio:\n")
					fmt.Fprintf(os.Stderr, "  Segments synthesized: %d/%d\n", len(results), segCount)
					fmt.Fprintf(os.Stderr, "  Total duration:       %.1f sec (%.1f min)\n",
						totalDuration, totalDuration/60.0)
					fmt.Fprintf(os.Stderr, "  Audio data size:      %.1f MB (base64)\n",
						float64(totalAudioSize)/1024/1024)
				}
			} else {
				fmt.Fprintf(os.Stderr, "\nAudio: no synth-results.json found\n")
			}

			// Check for the assembled HTML file.
			htmlPath := filepath.Join(tutorialDir, slug+".html")
			if fi, err := os.Stat(htmlPath); err == nil {
				sizeMB := float64(fi.Size()) / 1024 / 1024
				fmt.Fprintf(os.Stderr, "\nOutput:     %s (%.1f MB)\n", htmlPath, sizeMB)
			} else {
				fmt.Fprintf(os.Stderr, "\nOutput:     not yet assembled\n")
			}

			return nil
		},
	}
}

// wordCount returns a rough word count by splitting on whitespace.
func wordCount(s string) int {
	count := 0
	inWord := false
	for _, r := range s {
		if r == ' ' || r == '\n' || r == '\t' || r == '\r' {
			inWord = false
		} else if !inWord {
			inWord = true
			count++
		}
	}
	return count
}
