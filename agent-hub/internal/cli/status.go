package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	var update, from string
	cmd := &cobra.Command{
		Use:   "status <room>",
		Short: "Show or update the status board",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if update != "" {
				from = resolveAlias(from)
				return runUpdateStatus(args[0], update, from)
			}
			return runShowStatus(args[0])
		},
	}
	cmd.Flags().StringVar(&update, "update", "", "Update status: key=value")
	cmd.Flags().StringVar(&from, "from", "", "Who is updating")
	return cmd
}

func runShowStatus(room string) error {
	resp, err := apiCall("GET", "/api/v1/rooms/"+room+"/status", nil)
	return handleResponse(resp, err, func(data any) {
		m, _ := data.(map[string]any)
		entries, _ := m["entries"].([]any)
		if len(entries) == 0 {
			fmt.Println("Status board is empty")
			return
		}
		fmt.Printf("Status board for %q:\n", room)
		for _, e := range entries {
			em, _ := e.(map[string]any)
			key, _ := em["key"].(string)
			value, _ := em["value"].(string)
			by, _ := em["updated_by"].(string)
			fmt.Printf("  %-20s %s  (by %s)\n", key+":", value, by)
		}
	})
}

func runUpdateStatus(room, kv, from string) error {
	if from == "" {
		return printError("no identity set -- use `agent-hub identity set --alias=<name>` or pass --from")
	}
	parts := strings.SplitN(kv, "=", 2)
	if len(parts) != 2 {
		return printError("--update must be in key=value format")
	}

	body := map[string]string{
		"key":        parts[0],
		"value":      parts[1],
		"updated_by": from,
	}
	resp, err := apiCall("PUT", "/api/v1/rooms/"+room+"/status", body)
	return handleResponse(resp, err, func(data any) {
		fmt.Printf("Status updated: %s = %s\n", parts[0], parts[1])
	})
}
