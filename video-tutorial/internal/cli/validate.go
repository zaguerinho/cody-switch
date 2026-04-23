package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/zaguerinho/claude-switch/video-tutorial/internal/manifest"
)

func newValidateCmd() *cobra.Command {
	var sourceDir string

	cmd := &cobra.Command{
		Use:   "validate <manifest.json>",
		Short: "Validate a tutorial manifest",
		Long:  "Validate a tutorial manifest JSON file for correctness, checking\nscene structure, timing, and source file references.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			manifestPath := args[0]

			// 1. Read and parse manifest.json.
			data, err := os.ReadFile(manifestPath)
			if err != nil {
				return fmt.Errorf("reading manifest: %w", err)
			}
			var m manifest.Manifest
			if err := json.Unmarshal(data, &m); err != nil {
				return fmt.Errorf("parsing manifest: %w", err)
			}

			// 2. Resolve source files directory.
			dir := sourceDir
			if dir == "" {
				dir = m.SourceDir
			}
			if dir == "" {
				dir, err = os.Getwd()
				if err != nil {
					return fmt.Errorf("resolving working directory: %w", err)
				}
			}

			// 3. Collect all files referenced in the manifest.
			fileSet := make(map[string]struct{})
			for _, ch := range m.Chapters {
				for _, seg := range ch.Segments {
					if seg.Cue.File != "" {
						fileSet[seg.Cue.File] = struct{}{}
					}
				}
			}

			// Resolve referenced files to absolute paths.
			var filePaths []string
			for f := range fileSet {
				absPath := f
				if !filepath.IsAbs(f) {
					absPath = filepath.Join(dir, f)
				}
				filePaths = append(filePaths, absPath)
			}

			// 4. Validate.
			if err := manifest.Validate(&m, filePaths); err != nil {
				// 5. Print validation errors.
				fmt.Fprintf(os.Stderr, "%s\n", err)
				return fmt.Errorf("validation failed")
			}

			// 5. Print success.
			segCount := 0
			for _, ch := range m.Chapters {
				segCount += len(ch.Segments)
			}
			fmt.Fprintf(os.Stderr, "Manifest is valid: %q\n", m.Title)
			fmt.Fprintf(os.Stderr, "  %d chapters, %d segments, %d source files\n",
				len(m.Chapters), segCount, len(filePaths))

			return nil
		},
	}

	cmd.Flags().StringVar(&sourceDir, "source-dir", "", "Directory containing source files referenced in the manifest")

	return cmd
}
