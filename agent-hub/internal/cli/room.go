package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newRoomCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "room",
		Short: "Manage coordination rooms",
	}
	cmd.AddCommand(newRoomCreateCmd())
	cmd.AddCommand(newRoomListCmd())
	cmd.AddCommand(newRoomInfoCmd())
	cmd.AddCommand(newRoomArchiveCmd())
	return cmd
}

func newRoomCreateCmd() *cobra.Command {
	var description string
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new room",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]string{"name": args[0]}
			if description != "" {
				body["description"] = description
			}
			resp, err := apiCall("POST", "/api/v1/rooms", body)
			return handleResponse(resp, err, func(data any) {
				fmt.Printf("Room %q created\n", args[0])
			})
		},
	}
	cmd.Flags().StringVar(&description, "description", "", "Room description")
	return cmd
}

func newRoomListCmd() *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List rooms",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "/api/v1/rooms"
			if all {
				path += "?all=true"
			}
			resp, err := apiCall("GET", path, nil)
			return handleResponse(resp, err, func(data any) {
				rooms, ok := data.([]any)
				if !ok || len(rooms) == 0 {
					fmt.Println("No rooms")
					return
				}
				for _, s := range rooms {
					m, _ := s.(map[string]any)
					name := m["name"]
					archived, _ := m["archived"].(bool)
					agentCount, _ := m["agent_count"].(float64)
					msgCount, _ := m["message_count"].(float64)

					status := ""
					if archived {
						status = " [archived]"
					}
					fmt.Printf("  %-25s %d agents  %d messages%s\n", name, int(agentCount), int(msgCount), status)
				}
			})
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "Include archived rooms")
	return cmd
}

func newRoomInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info <name>",
		Short: "Show room details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := apiCall("GET", "/api/v1/rooms/"+args[0], nil)
			return handleResponse(resp, err, func(data any) {
				m, _ := data.(map[string]any)
				fmt.Printf("Room: %s\n", m["name"])
				if desc, ok := m["description"].(string); ok && desc != "" {
					fmt.Printf("Description: %s\n", desc)
				}
				fmt.Printf("Created: %s\n", m["created_at"])
				fmt.Printf("Agents: %.0f\n", m["agent_count"])
				fmt.Printf("Messages: %.0f\n", m["message_count"])
				if act, ok := m["last_activity"].(string); ok {
					fmt.Printf("Last Activity: %s\n", act)
				}
			})
		},
	}
}

func newRoomArchiveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "archive <name>",
		Short: "Archive a room",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := apiCall("POST", "/api/v1/rooms/"+args[0]+"/archive", nil)
			return handleResponse(resp, err, func(data any) {
				fmt.Printf("Room %q archived\n", args[0])
			})
		},
	}
}

// marshalData re-marshals the generic data for structured access.
func marshalData(data any, v any) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}
