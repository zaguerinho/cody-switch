package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zaguerinho/claude-switch/video-tutorial/internal/config"
	"github.com/zaguerinho/claude-switch/video-tutorial/internal/server"
)

func newServeCmd(version string) *cobra.Command {
	var port int

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the interactive Q&A server",
		Long: `Start a local HTTP server that proxies chatbot questions to the Claude CLI.
Run alongside an open tutorial HTML file to enable interactive Q&A without
needing an API key — uses your existing Claude Code authentication.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.ValidateCLI(); err != nil {
				return err
			}

			addr := fmt.Sprintf(":%d", port)
			srv := server.New(addr, version)
			return srv.ListenAndServe(cmd.Context())
		},
	}

	cmd.Flags().IntVar(&port, "port", 19191, "Port to listen on")

	return cmd
}
