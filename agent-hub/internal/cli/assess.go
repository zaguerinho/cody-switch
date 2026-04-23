package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newAssessCmd() *cobra.Command {
	var alias string
	cmd := &cobra.Command{
		Use:   "assess <room>",
		Short: "Get a full room assessment (phase, dimensions, RFCs, unread)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			alias = resolveAlias(alias)
			path := "/api/v1/rooms/" + args[0] + "/assess"
			if alias != "" {
				path += "?as=" + alias
			}
			resp, err := apiCall("GET", path, nil)
			return handleResponse(resp, err, func(data any) {
				m, _ := data.(map[string]any)

				phase, _ := m["phase"].(float64)
				ready, _ := m["manifesto_ready"].(bool)
				unread, _ := m["your_unread"].(float64)
				total, _ := m["total_messages"].(float64)

				// Phase label
				phaseLabels := map[int]string{
					0: "Discovery", 1: "Setup", 2: "Investigation",
					3: "Evidence", 4: "Review", 5: "Sign-off",
				}
				label := phaseLabels[int(phase)]

				fmt.Printf("Room: %s\n", m["room"])
				fmt.Printf("Phase: %d (%s)\n", int(phase), label)
				fmt.Printf("Manifesto: %s\n", readyStr(ready))
				if alias != "" {
					fmt.Printf("Your unread: %d\n", int(unread))
				}
				fmt.Printf("Messages: %d\n", int(total))

				// Agents
				agents, _ := m["agents"].([]any)
				if len(agents) > 0 {
					names := make([]string, len(agents))
					for i, a := range agents {
						names[i], _ = a.(string)
					}
					fmt.Printf("Agents: %s\n", strings.Join(names, ", "))
				}

				// Dimensions
				dims, _ := m["dimensions"].(map[string]any)
				if len(dims) > 0 {
					fmt.Println("Dimensions:")
					for k, v := range dims {
						dm, _ := v.(map[string]any)
						status, _ := dm["status"].(string)
						owner, _ := dm["owner"].(string)
						if owner != "" {
							fmt.Printf("  %-12s %-8s (owner: %s)\n", k, status, owner)
						} else {
							fmt.Printf("  %-12s %-8s (unassigned)\n", k, status)
						}
					}
				}

				// Open RFCs
				rfcs, _ := m["open_rfcs"].([]any)
				if len(rfcs) > 0 {
					ids := make([]string, len(rfcs))
					for i, r := range rfcs {
						n, _ := r.(float64)
						ids[i] = fmt.Sprintf("#%d", int(n))
					}
					fmt.Printf("Open RFCs: %s\n", strings.Join(ids, ", "))
				}

				if act, ok := m["last_activity"].(string); ok && act != "" {
					fmt.Printf("Last activity: %s\n", act)
				}

				// Advance readiness
				readyAdv, _ := m["ready_to_advance"].(bool)
				blocker, _ := m["advance_blocker"].(string)
				needsHuman, _ := m["human_approval_needed"].(bool)
				if readyAdv {
					if needsHuman {
						fmt.Printf("Next phase: READY (needs human approval — run: agent-hub approve %s)\n", m["room"])
					} else {
						fmt.Printf("Next phase: READY to advance\n")
					}
				} else if blocker != "" {
					fmt.Printf("Advance blocked: %s\n", blocker)
				}

				// Warnings
				warnings, _ := m["warnings"].([]any)
				for _, w := range warnings {
					ws, _ := w.(string)
					if ws != "" {
						fmt.Printf("WARNING: %s\n", ws)
					}
				}
			})
		},
	}
	cmd.Flags().StringVar(&alias, "as", "", "Agent alias")
	return cmd
}

func readyStr(ready bool) string {
	if ready {
		return "locked"
	}
	return "placeholder (needs discovery)"
}
