package cli

import (
	"github.com/spf13/cobra"
)

var (
	jsonOutput bool
	apiPort    int
	uiPort     int
)

func newRootCmd(version string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "agent-hub",
		Short:   "Local multi-agent coordination server for Claude Code",
		Long:    "A lightweight coordination server that enables Claude Code agents\nacross different projects to communicate via structured messages,\nstatus boards, and read tracking.",
		Version: version,
	}

	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	rootCmd.PersistentFlags().IntVar(&apiPort, "api-port", 7777, "API server port")
	rootCmd.PersistentFlags().IntVar(&uiPort, "ui-port", 9093, "Dashboard UI port")

	// Server lifecycle
	rootCmd.AddCommand(newServeCmd(version))
	rootCmd.AddCommand(newStopCmd())
	rootCmd.AddCommand(newHealthCmd())

	// Room management
	rootCmd.AddCommand(newRoomCmd())

	// Identity management
	rootCmd.AddCommand(newIdentityCmd())

	// Agent commands
	rootCmd.AddCommand(newJoinCmd())
	rootCmd.AddCommand(newLeaveCmd())
	rootCmd.AddCommand(newWhoCmd())

	// Message commands
	rootCmd.AddCommand(newPostCmd())
	rootCmd.AddCommand(newCheckCmd())
	rootCmd.AddCommand(newCheckAllCmd())
	rootCmd.AddCommand(newReadCmd())
	rootCmd.AddCommand(newAckCmd())

	// Status board
	rootCmd.AddCommand(newStatusCmd())

	// Assessment + approval
	rootCmd.AddCommand(newAssessCmd())
	rootCmd.AddCommand(newApproveCmd())

	// Documents
	rootCmd.AddCommand(newDocCmd())

	return rootCmd
}

// Execute runs the root command with the given version string.
func Execute(version string) error {
	return newRootCmd(version).Execute()
}
