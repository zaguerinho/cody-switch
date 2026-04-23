package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/spf13/cobra"
	"github.com/zaguerinho/claude-switch/video-tutorial/internal/assembler"
	"github.com/zaguerinho/claude-switch/video-tutorial/internal/config"
	"github.com/zaguerinho/claude-switch/video-tutorial/internal/fileutil"
	"github.com/zaguerinho/claude-switch/video-tutorial/internal/manifest"
	"github.com/zaguerinho/claude-switch/video-tutorial/internal/notify"
	"github.com/zaguerinho/claude-switch/video-tutorial/internal/tts"
)

func newBuildCmd() *cobra.Command {
	var (
		topic      string
		files      string
		voice      string
		ttsBackend string
		piperModel string
		yes        bool
	)

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build a narrated code tutorial",
		Long:  "Build a narrated code tutorial from source files and a topic description.\nGenerates a manifest, synthesizes audio, and assembles the final HTML.",
		RunE: func(cmd *cobra.Command, args []string) error {
			startTime := time.Now()

			// 1. Create config, load env, apply flag overrides.
			cfg := config.New()
			cfg.LoadEnv()

			if ttsBackend != "" {
				cfg.TTSBackend = ttsBackend
			}
			if voice != "" {
				cfg.ElevenLabsVoiceID = voice
			}
			if piperModel != "" {
				cfg.PiperModel = piperModel
			}

			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("invalid config: %w", err)
			}

			// 2. Check claude CLI is available.
			if err := config.ValidateCLI(); err != nil {
				return err
			}

			// 3. Resolve source files using fileutil.ExpandGlob.
			sourceDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("resolving working directory: %w", err)
			}

			pattern := files
			if pattern == "" {
				pattern = "**/*"
			}

			filePaths, err := fileutil.ExpandGlob(pattern, sourceDir)
			if err != nil {
				return fmt.Errorf("expanding file glob: %w", err)
			}
			if len(filePaths) == 0 {
				return fmt.Errorf("no files matched pattern %q in %s", pattern, sourceDir)
			}
			fmt.Fprintf(os.Stderr, "Matched %d source files\n", len(filePaths))

			// Determine output base directory.
			outputBase := resolveOutputBase()
			slug := fileutil.Slugify(topic)
			tutorialDir := filepath.Join(outputBase, slug)
			if err := os.MkdirAll(tutorialDir, 0o755); err != nil {
				return fmt.Errorf("creating tutorial dir: %w", err)
			}

			// 4. Generate manifest via Claude CLI.
			ctx := context.Background()
			fmt.Fprintf(os.Stderr, "Generating manifest...\n")
			m, err := manifest.Generate(ctx, topic, filePaths, sourceDir)
			if err != nil {
				return fmt.Errorf("manifest generation: %w", err)
			}
			m.SourceDir = sourceDir

			// 5. Save manifest.json.
			manifestPath := filepath.Join(tutorialDir, "manifest.json")
			if err := writeJSON(manifestPath, m); err != nil {
				return fmt.Errorf("saving manifest: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Manifest saved: %s\n", manifestPath)

			// 6. Flatten segments, load existing synth results for resume.
			flat := m.FlattenSegments()
			synthResultsPath := filepath.Join(tutorialDir, "synth-results.json")
			existingResults := loadSynthResults(synthResultsPath)
			existingMap := make(map[string]tts.SynthResult, len(existingResults))
			for _, r := range existingResults {
				existingMap[r.ID] = r
			}

			// Determine which segments still need synthesis.
			var toSynthesize []tts.SegmentInput
			var reusedResults []tts.SynthResult
			for _, seg := range flat {
				if existing, ok := existingMap[seg.ID]; ok {
					reusedResults = append(reusedResults, existing)
					if verbose {
						fmt.Fprintf(os.Stderr, "  Reusing cached audio for %s\n", seg.ID)
					}
				} else {
					toSynthesize = append(toSynthesize, tts.SegmentInput{
						ID:   seg.ID,
						Text: seg.Text,
					})
				}
			}

			if len(reusedResults) > 0 {
				fmt.Fprintf(os.Stderr, "Reusing %d cached segments, synthesizing %d remaining\n",
					len(reusedResults), len(toSynthesize))
			}

			// 7. Synthesize remaining segments with incremental save.
			allResults := make([]tts.SynthResult, 0, len(flat))
			allResults = append(allResults, reusedResults...)

			if len(toSynthesize) > 0 {
				// Auto-detect or use specified TTS provider.
				provider, err := tts.NewProvider(tts.Backend(cfg.TTSBackend), cfg)
				if err != nil {
					return err
				}

				// For ElevenLabs: interactive voice selection + cost confirmation.
				if provider.Name() == "ElevenLabs" && !yes {
					// 1. Voice selection (unless --voice was explicitly passed).
					if voice == "" {
						cfg.ElevenLabsVoiceID = tts.PromptVoiceSelection(cfg.ElevenLabsVoiceID, false)
						// Recreate provider with updated voice ID.
						provider, err = tts.NewProvider(tts.Backend(cfg.TTSBackend), cfg)
						if err != nil {
							return err
						}
					}

					// 2. Cost estimate + confirmation.
					if !tts.ConfirmCostEstimate(toSynthesize, cfg, false) {
						fmt.Fprintf(os.Stderr, "Cancelled. Manifest saved — re-run to continue.\n")
						return nil
					}
				}
				fmt.Fprintf(os.Stderr, "Synthesizing %d segments via %s...\n", len(toSynthesize), provider.Name())

				// Use rate-limit gap for cloud APIs, none for local.
				gapMS := cfg.SegmentGapMS
				if provider.Name() == "Piper" {
					gapMS = 0 // no rate limiting needed for local TTS
				}

				onProgress := func(completed, total int, segmentID string) {
					fmt.Fprintf(os.Stderr, "  [%d/%d] %s\n", completed, total, segmentID)
				}

				onResult := func(result tts.SynthResult) {
					allResults = append(allResults, result)
					// Incremental save after each segment.
					_ = writeJSON(synthResultsPath, allResults)
				}

				newResults, err := tts.SynthesizeAll(ctx, provider, toSynthesize, gapMS, onProgress, onResult)
				if err != nil {
					// Save partial results before returning the error.
					_ = writeJSON(synthResultsPath, allResults)
					return fmt.Errorf("synthesis failed (partial results saved): %w", err)
				}
				_ = newResults // results already appended via onResult callback
			}

			// Build a properly ordered results slice matching manifest segment order.
			orderedResults := orderResultsByManifest(flat, allResults)

			// Final save of complete results.
			if err := writeJSON(synthResultsPath, orderedResults); err != nil {
				return fmt.Errorf("saving synth results: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Synth results saved: %s\n", synthResultsPath)

			// 8. Assemble via assembler.Assemble.
			outputPath := filepath.Join(tutorialDir, slug+".html")
			cfg.OutputDir = tutorialDir
			if err := assembler.Assemble(assembler.AssembleParams{
				Manifest:     m,
				SynthResults: orderedResults,
				SourceDir:    sourceDir,
				OutputPath:   outputPath,
				Slug:         slug,
				Config:       cfg,
			}); err != nil {
				return fmt.Errorf("assembly: %w", err)
			}

			// 9. Print results to stderr.
			elapsed := time.Since(startTime)
			segCount := countSegments(m)
			fi, _ := os.Stat(outputPath)
			var sizeMB float64
			if fi != nil {
				sizeMB = float64(fi.Size()) / 1024 / 1024
			}

			fmt.Fprintf(os.Stderr, "\nBuild complete in %s\n", elapsed.Round(time.Second))
			fmt.Fprintf(os.Stderr, "  Title:    %s\n", m.Title)
			fmt.Fprintf(os.Stderr, "  Chapters: %d\n", len(m.Chapters))
			fmt.Fprintf(os.Stderr, "  Segments: %d\n", segCount)
			fmt.Fprintf(os.Stderr, "  Size:     %.1f MB\n", sizeMB)
			fmt.Fprintf(os.Stderr, "  Output:   %s\n", outputPath)

			// 10. Send macOS notification.
			if runtime.GOOS == "darwin" {
				notify.Send(
					"Tutorial Ready",
					fmt.Sprintf("%s (%d chapters)", m.Title, len(m.Chapters)),
					outputPath,
				)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&topic, "topic", "", "Tutorial topic description (required)")
	cmd.Flags().StringVar(&files, "files", "", "Glob pattern for source files to include")
	cmd.Flags().StringVar(&ttsBackend, "tts", "auto", "TTS backend: auto, piper (free/local), elevenlabs (cloud)")
	cmd.Flags().StringVar(&voice, "voice", "", "ElevenLabs voice ID (skip voice picker)")
	cmd.Flags().StringVar(&piperModel, "piper-model", "", "Piper voice model (e.g., en_US-lessac-medium)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompts")
	_ = cmd.MarkFlagRequired("topic")

	return cmd
}

// resolveOutputBase returns the output base directory.
// If the --output flag is set, use that. Otherwise, use the video-tutorial's
// own output/tutorials/ directory (resolved from the executable path).
func resolveOutputBase() string {
	if output != "" {
		return output
	}
	// Default: output/tutorials/ relative to the executable's directory.
	exe, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exe)
		candidate := filepath.Join(exeDir, "output", "tutorials")
		if _, err := os.Stat(filepath.Dir(candidate)); err == nil {
			return candidate
		}
	}
	// Fallback: output/tutorials/ in the current working directory.
	cwd, _ := os.Getwd()
	return filepath.Join(cwd, "output", "tutorials")
}

// writeJSON marshals v to JSON with indentation and writes it to path.
func writeJSON(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// loadSynthResults reads synth-results.json and returns the parsed results.
// Returns nil on any error (file missing, parse error, etc.).
func loadSynthResults(path string) []tts.SynthResult {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var results []tts.SynthResult
	if err := json.Unmarshal(data, &results); err != nil {
		return nil
	}
	return results
}

// orderResultsByManifest returns synth results in the order segments appear in the manifest.
func orderResultsByManifest(flat []manifest.FlatSegment, results []tts.SynthResult) []tts.SynthResult {
	resultMap := make(map[string]tts.SynthResult, len(results))
	for _, r := range results {
		resultMap[r.ID] = r
	}
	ordered := make([]tts.SynthResult, 0, len(flat))
	for _, seg := range flat {
		if r, ok := resultMap[seg.ID]; ok {
			ordered = append(ordered, r)
		}
	}
	return ordered
}

// countSegments returns the total number of segments across all chapters.
func countSegments(m *manifest.Manifest) int {
	count := 0
	for _, ch := range m.Chapters {
		count += len(ch.Segments)
	}
	return count
}
