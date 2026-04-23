package cli

import (
	"github.com/spf13/cobra"
)

var (
	verbose bool
	output  string
)

func newRootCmd(version string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "video-tutorial",
		Short:   "Generate narrated code tutorial screencasts",
		Long:    "Generate narrated code tutorial screencasts as self-contained HTML files.\nCombines syntax-highlighted code, step-by-step narration, and text-to-speech\naudio into a single portable HTML document.",
		Version: version,
	}

	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "", "Output file path")

	rootCmd.AddCommand(newBuildCmd())
	rootCmd.AddCommand(newAssembleCmd())
	rootCmd.AddCommand(newValidateCmd())
	rootCmd.AddCommand(newInspectCmd())
	rootCmd.AddCommand(newExportTranscriptCmd())
	rootCmd.AddCommand(newServeCmd(version))

	return rootCmd
}

// Execute runs the root command with the given version string.
func Execute(version string) error {
	return newRootCmd(version).Execute()
}
