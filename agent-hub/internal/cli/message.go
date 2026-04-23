package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func newPostCmd() *cobra.Command {
	var from, msgType, subject, file string
	cmd := &cobra.Command{
		Use:   "post <room> [body]",
		Short: "Post a message to a room",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			from = resolveAlias(from)
			if from == "" {
				return printError("no identity set -- use `agent-hub identity set --alias=<name>` or pass --from")
			}

			var body string
			if file != "" {
				data, err := os.ReadFile(file)
				if err != nil {
					return printError("read file: " + err.Error())
				}
				body = string(data)
			} else if len(args) > 1 {
				body = strings.Join(args[1:], " ")
			} else {
				return printError("message body required (provide as argument or --file)")
			}

			if msgType == "" {
				msgType = "note"
			}

			reqBody := map[string]string{
				"from":    from,
				"type":    msgType,
				"body":    body,
				"subject": subject,
			}
			resp, err := apiCall("POST", "/api/v1/rooms/"+args[0]+"/messages", reqBody)
			return handleResponse(resp, err, func(data any) {
				m, _ := data.(map[string]any)
				id, _ := m["id"].(float64)
				fmt.Printf("Message #%d posted to %q\n", int(id), args[0])
			})
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "Sender alias")
	cmd.Flags().StringVar(&msgType, "type", "note", "Message type: question, answer, rfc, note, status-update")
	cmd.Flags().StringVar(&subject, "subject", "", "Message subject")
	cmd.Flags().StringVar(&file, "file", "", "Read body from file")
	return cmd
}

func newCheckCmd() *cobra.Command {
	var alias string
	cmd := &cobra.Command{
		Use:   "check <room>",
		Short: "Check for unread messages",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			alias = resolveAlias(alias)
			if alias == "" {
				return printError("no identity set -- use `agent-hub identity set --alias=<name>` or pass --as")
			}
			resp, err := apiCall("GET", "/api/v1/rooms/"+args[0]+"/messages/check?as="+alias, nil)
			return handleResponse(resp, err, func(data any) {
				m, _ := data.(map[string]any)
				unread, _ := m["unread"].(float64)
				if unread == 0 {
					fmt.Println("No unread messages")
					return
				}
				fmt.Printf("%d unread message(s)\n", int(unread))
				if latest, ok := m["latest"].(string); ok && latest != "" {
					fmt.Printf("  Latest: %s\n", latest)
				}
			})
		},
	}
	cmd.Flags().StringVar(&alias, "as", "", "Agent alias")
	return cmd
}

func newCheckAllCmd() *cobra.Command {
	var alias string
	cmd := &cobra.Command{
		Use:    "check-all",
		Short:  "Check all rooms for unread messages",
		Hidden: true, // Used by session-start hook
		RunE: func(cmd *cobra.Command, args []string) error {
			alias = resolveAlias(alias)
			if alias == "" {
				return printError("no identity set -- use `agent-hub identity set --alias=<name>` or pass --as")
			}
			resp, err := apiCall("GET", "/api/v1/rooms/check-all?as="+alias, nil)
			return handleResponse(resp, err, func(data any) {
				m, _ := data.(map[string]any)
				rooms, _ := m["rooms"].([]any)
				hasUnread := false
				for _, s := range rooms {
					sm, _ := s.(map[string]any)
					unread, _ := sm["unread"].(float64)
					if unread > 0 {
						hasUnread = true
						fmt.Printf("  [%s] %d unread\n", sm["room"], int(unread))
					}
				}
				if !hasUnread {
					fmt.Println("No unread messages")
				}
			})
		},
	}
	cmd.Flags().StringVar(&alias, "as", "", "Agent alias")
	return cmd
}

func newReadCmd() *cobra.Command {
	var alias string
	var last int
	var unread bool
	cmd := &cobra.Command{
		Use:   "read <room>",
		Short: "Read messages from a room",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "/api/v1/rooms/" + args[0] + "/messages"
			params := []string{}
			if unread {
				resolved := resolveAlias(alias)
				if resolved == "" {
					return printError("no identity set -- use `agent-hub identity set --alias=<name>` or pass --as")
				}
				params = append(params, "unread=true", "as="+resolved)
			} else if last > 0 {
				params = append(params, fmt.Sprintf("last=%d", last))
			}
			if len(params) > 0 {
				path += "?" + strings.Join(params, "&")
			}

			resp, err := apiCall("GET", path, nil)
			return handleResponse(resp, err, func(data any) {
				msgs, ok := data.([]any)
				if !ok || len(msgs) == 0 {
					fmt.Println("No messages")
					return
				}
				for _, msg := range msgs {
					m, _ := msg.(map[string]any)
					id, _ := m["id"].(float64)
					from, _ := m["from"].(string)
					typ, _ := m["type"].(string)
					subj, _ := m["subject"].(string)
					body, _ := m["body"].(string)
					ts, _ := m["timestamp"].(string)

					// Format type badge
					badge := strings.ToUpper(string(typ[0:1]))
					fmt.Printf("\n--- #%d [%s] %s (%s) ---\n", int(id), badge, from, ts)
					if subj != "" {
						fmt.Printf("Subject: %s\n", subj)
					}
					fmt.Println(body)
				}
			})
		},
	}
	cmd.Flags().StringVar(&alias, "as", "", "Agent alias (for --unread)")
	cmd.Flags().IntVar(&last, "last", 0, "Read last N messages")
	cmd.Flags().BoolVar(&unread, "unread", false, "Only show unread messages")
	return cmd
}

func newAckCmd() *cobra.Command {
	var alias string
	var upTo int
	cmd := &cobra.Command{
		Use:   "ack <room>",
		Short: "Acknowledge messages as read",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			alias = resolveAlias(alias)
			if alias == "" {
				return printError("no identity set -- use `agent-hub identity set --alias=<name>` or pass --as")
			}
			body := map[string]any{"alias": alias, "up_to": upTo}
			resp, err := apiCall("POST", "/api/v1/rooms/"+args[0]+"/messages/ack", body)
			return handleResponse(resp, err, func(data any) {
				fmt.Println("Messages acknowledged")
			})
		},
	}
	cmd.Flags().StringVar(&alias, "as", "", "Agent alias")
	cmd.Flags().IntVar(&upTo, "up-to", 0, "Acknowledge up to message ID (0 = all)")
	return cmd
}
