package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newApproveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "approve <room>",
		Short: "Human approval gate — advance to next phase",
		Long:  "Checks if the room is ready to advance, then posts an approval message and updates the phase. Used at human-gated transitions (Phase 1→2 and Phase 4→5).",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			room := args[0]

			// Get current assessment
			resp, err := apiCall("GET", "/api/v1/rooms/"+room+"/assess", nil)
			if err != nil {
				return printError(err.Error())
			}
			if !resp.OK {
				return printError(resp.Error)
			}

			// Parse assessment
			data, _ := json.Marshal(resp.Data)
			var assess struct {
				Phase         int    `json:"phase"`
				PhaseLabel    string `json:"phase_label"`
				ReadyToAdvance bool  `json:"ready_to_advance"`
				AdvanceBlocker string `json:"advance_blocker"`
				HumanApproval bool   `json:"human_approval_needed"`
			}
			json.Unmarshal(data, &assess)

			if !assess.ReadyToAdvance {
				return printError(fmt.Sprintf("not ready to advance: %s", assess.AdvanceBlocker))
			}

			nextPhase := assess.Phase + 1
			phaseLabels := map[int]string{
				0: "Discovery", 1: "Setup", 2: "Investigation",
				3: "Evidence", 4: "Review", 5: "Sign-off",
			}

			// Post approval message
			body := map[string]string{
				"from":    "human",
				"type":    "note",
				"subject": fmt.Sprintf("APPROVED: Phase %d → %d (%s)", assess.Phase, nextPhase, phaseLabels[nextPhase]),
				"body":    fmt.Sprintf("Human approval granted. Advancing from Phase %d (%s) to Phase %d (%s).", assess.Phase, assess.PhaseLabel, nextPhase, phaseLabels[nextPhase]),
			}
			apiCall("POST", "/api/v1/rooms/"+room+"/messages", body)

			// Advance phase
			statusBody := map[string]string{
				"key":        "phase",
				"value":      fmt.Sprintf("%d", nextPhase),
				"updated_by": "human",
			}
			apiCall("PUT", "/api/v1/rooms/"+room+"/status", statusBody)

			if jsonOutput {
				resp, _ := apiCall("GET", "/api/v1/rooms/"+room+"/assess", nil)
				if resp != nil {
					printJSON(resp)
				}
			} else {
				fmt.Printf("Approved. Phase %d (%s) → Phase %d (%s)\n", assess.Phase, assess.PhaseLabel, nextPhase, phaseLabels[nextPhase])
			}
			return nil
		},
	}
	return cmd
}
