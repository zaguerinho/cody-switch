package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/zaguerinho/claude-switch/video-tutorial/internal/assembler"
	"github.com/zaguerinho/claude-switch/video-tutorial/internal/config"
	"github.com/zaguerinho/claude-switch/video-tutorial/internal/manifest"
	"github.com/zaguerinho/claude-switch/video-tutorial/internal/tts"
)

func newAssembleCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "assemble <slug>",
		Short: "Re-assemble HTML from a cached manifest",
		Long:  "Re-assemble the final HTML tutorial from a previously generated manifest\nand cached audio files, without re-running the build pipeline.",
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

			// Load synth-results.json.
			synthPath := filepath.Join(tutorialDir, "synth-results.json")
			synthData, err := os.ReadFile(synthPath)
			if err != nil {
				return fmt.Errorf("reading synth results: %w\nExpected at: %s", err, synthPath)
			}
			var synthResults []tts.SynthResult
			if err := json.Unmarshal(synthData, &synthResults); err != nil {
				return fmt.Errorf("parsing synth results: %w", err)
			}

			// 2. Resolve sourceDir from manifest.SourceDir.
			sourceDir := m.SourceDir
			if sourceDir == "" {
				// Fall back to current working directory.
				sourceDir, err = os.Getwd()
				if err != nil {
					return fmt.Errorf("resolving working directory: %w", err)
				}
				fmt.Fprintf(os.Stderr, "Warning: manifest has no _sourceDir; using current directory: %s\n", sourceDir)
			}

			// 3. Assemble.
			cfg := config.New()
			outputPath := filepath.Join(tutorialDir, slug+".html")
			if err := assembler.Assemble(assembler.AssembleParams{
				Manifest:     &m,
				SynthResults: synthResults,
				SourceDir:    sourceDir,
				OutputPath:   outputPath,
				Slug:         slug,
				Config:       cfg,
			}); err != nil {
				return fmt.Errorf("assembly: %w", err)
			}

			// 4. Print results.
			fi, _ := os.Stat(outputPath)
			var sizeMB float64
			if fi != nil {
				sizeMB = float64(fi.Size()) / 1024 / 1024
			}
			fmt.Fprintf(os.Stderr, "Re-assembled: %s (%.1f MB)\n", outputPath, sizeMB)

			return nil
		},
	}
}
