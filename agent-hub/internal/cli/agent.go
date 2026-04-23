package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newJoinCmd() *cobra.Command {
	var alias, role string
	cmd := &cobra.Command{
		Use:   "join <room>",
		Short: "Join a room as an agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			alias = resolveAlias(alias)
			if alias == "" {
				return printError("no identity set -- use `agent-hub identity set --alias=<name>` or pass --as")
			}
			body := map[string]string{"alias": alias}
			if role != "" {
				body["role"] = role
			}
			resp, err := apiCall("POST", "/api/v1/rooms/"+args[0]+"/agents", body)
			return handleResponse(resp, err, func(data any) {
				fmt.Printf("Joined room %q as %q\n", args[0], alias)
			})
		},
	}
	cmd.Flags().StringVar(&alias, "as", "", "Agent alias")
	cmd.Flags().StringVar(&role, "role", "", "Agent role")
	return cmd
}

func newLeaveCmd() *cobra.Command {
	var alias string
	cmd := &cobra.Command{
		Use:   "leave <room>",
		Short: "Leave a room",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			alias = resolveAlias(alias)
			if alias == "" {
				return printError("no identity set -- use `agent-hub identity set --alias=<name>` or pass --as")
			}
			resp, err := apiCall("DELETE", "/api/v1/rooms/"+args[0]+"/agents/"+alias, nil)
			return handleResponse(resp, err, func(data any) {
				fmt.Printf("Left room %q\n", args[0])
			})
		},
	}
	cmd.Flags().StringVar(&alias, "as", "", "Agent alias")
	return cmd
}

func newWhoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "who <room>",
		Short: "List agents in a room",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := apiCall("GET", "/api/v1/rooms/"+args[0]+"/agents", nil)
			return handleResponse(resp, err, func(data any) {
				agents, ok := data.([]any)
				if !ok || len(agents) == 0 {
					fmt.Println("No agents in room")
					return
				}
				for _, a := range agents {
					m, _ := a.(map[string]any)
					alias := m["alias"]
					role, _ := m["role"].(string)
					if role != "" {
						fmt.Printf("  %s (%s)\n", alias, role)
					} else {
						fmt.Printf("  %s\n", alias)
					}
				}
			})
		},
	}
}
