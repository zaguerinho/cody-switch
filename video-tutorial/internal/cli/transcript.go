package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zaguerinho/claude-switch/video-tutorial/internal/manifest"
	"github.com/zaguerinho/claude-switch/video-tutorial/internal/tts"
)

func newExportTranscriptCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "export-transcript <slug>",
		Short: "Export tutorial transcript",
		Long:  "Export the narration transcript for a tutorial in the specified format.\nSupported formats: text (plain text), srt (SubRip), vtt (WebVTT).",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			validFormats := map[string]bool{"text": true, "srt": true, "vtt": true}
			if !validFormats[format] {
				return fmt.Errorf("invalid format %q: must be one of text, srt, vtt", format)
			}

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

			// 2. Load synth-results.json for timestamps (needed for srt/vtt).
			var synthResults []tts.SynthResult
			synthPath := filepath.Join(tutorialDir, "synth-results.json")
			if format == "srt" || format == "vtt" {
				synthData, err := os.ReadFile(synthPath)
				if err != nil {
					return fmt.Errorf("reading synth results (needed for %s format): %w", format, err)
				}
				if err := json.Unmarshal(synthData, &synthResults); err != nil {
					return fmt.Errorf("parsing synth results: %w", err)
				}
			}

			switch format {
			case "text":
				return exportText(&m)
			case "srt":
				return exportSRT(&m, synthResults)
			case "vtt":
				return exportVTT(&m, synthResults)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", "text", "Output format: text, srt, or vtt")

	return cmd
}

// exportText outputs plain narration text per chapter to stdout.
func exportText(m *manifest.Manifest) error {
	for i, ch := range m.Chapters {
		if i > 0 {
			fmt.Println()
		}
		fmt.Printf("## %s\n\n", ch.Title)
		for _, seg := range ch.Segments {
			fmt.Println(seg.Narration)
		}
	}
	return nil
}

// exportSRT outputs numbered subtitle entries with SRT timestamps to stdout.
func exportSRT(m *manifest.Manifest, synthResults []tts.SynthResult) error {
	// Build a map from segment ID to synth result for timing data.
	resultMap := make(map[string]*tts.SynthResult, len(synthResults))
	for i := range synthResults {
		resultMap[synthResults[i].ID] = &synthResults[i]
	}

	entryNum := 1
	var timeOffset float64

	for _, ch := range m.Chapters {
		for _, seg := range ch.Segments {
			result, ok := resultMap[seg.ID]
			if !ok {
				return fmt.Errorf("no synth result for segment %q", seg.ID)
			}

			startTime := timeOffset
			endTime := timeOffset + result.Duration

			fmt.Printf("%d\n", entryNum)
			fmt.Printf("%s --> %s\n", formatSRTTime(startTime), formatSRTTime(endTime))
			fmt.Printf("%s\n\n", seg.Narration)

			timeOffset = endTime + 0.3 // segment gap
			entryNum++
		}
	}

	return nil
}

// exportVTT outputs WebVTT format to stdout.
func exportVTT(m *manifest.Manifest, synthResults []tts.SynthResult) error {
	// Build a map from segment ID to synth result for timing data.
	resultMap := make(map[string]*tts.SynthResult, len(synthResults))
	for i := range synthResults {
		resultMap[synthResults[i].ID] = &synthResults[i]
	}

	fmt.Println("WEBVTT")
	fmt.Println()

	var timeOffset float64

	for _, ch := range m.Chapters {
		for _, seg := range ch.Segments {
			result, ok := resultMap[seg.ID]
			if !ok {
				return fmt.Errorf("no synth result for segment %q", seg.ID)
			}

			startTime := timeOffset
			endTime := timeOffset + result.Duration

			fmt.Printf("%s\n", seg.ID)
			fmt.Printf("%s --> %s\n", formatVTTTime(startTime), formatVTTTime(endTime))
			fmt.Printf("%s\n\n", seg.Narration)

			timeOffset = endTime + 0.3 // segment gap
		}
	}

	return nil
}

// formatSRTTime formats seconds as HH:MM:SS,mmm (SRT format uses comma).
func formatSRTTime(seconds float64) string {
	return formatTimestamp(seconds, ",")
}

// formatVTTTime formats seconds as HH:MM:SS.mmm (VTT format uses period).
func formatVTTTime(seconds float64) string {
	return formatTimestamp(seconds, ".")
}

// formatTimestamp formats seconds as HH:MM:SS{sep}mmm.
func formatTimestamp(seconds float64, sep string) string {
	if seconds < 0 {
		seconds = 0
	}
	totalMS := int(seconds * 1000)
	h := totalMS / 3600000
	totalMS %= 3600000
	min := totalMS / 60000
	totalMS %= 60000
	sec := totalMS / 1000
	ms := totalMS % 1000

	var b strings.Builder
	fmt.Fprintf(&b, "%02d:%02d:%02d%s%03d", h, min, sec, sep, ms)
	return b.String()
}
