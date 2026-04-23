package store

import (
	"os"
	"path/filepath"
	"strings"
)

// Assessment is the pre-computed state of a room for a specific agent.
type Assessment struct {
	Room           string                `json:"room"`
	Phase          int                   `json:"phase"`
	PhaseLabel     string                `json:"phase_label"`
	ManifestoReady bool                  `json:"manifesto_ready"`
	Agents         []string              `json:"agents"`
	Dimensions     map[string]DimStatus  `json:"dimensions"`
	OpenRFCs       []int                 `json:"open_rfcs"`
	ApprovedRFCs   []int                 `json:"approved_rfcs"`
	YourUnread     int                   `json:"your_unread"`
	TotalMessages  int                   `json:"total_messages"`
	LastActivity   string                `json:"last_activity,omitempty"`
	ReadyToAdvance bool                  `json:"ready_to_advance"`
	AdvanceBlocker string                `json:"advance_blocker,omitempty"`
	HumanApproval  bool                  `json:"human_approval_needed"`
	Warnings       []string              `json:"warnings,omitempty"`
}

// DimStatus tracks a single quality dimension.
type DimStatus struct {
	Status string `json:"status"`
	Owner  string `json:"owner,omitempty"`
}

// Assess computes a full room assessment for a given agent.
func (fs *FileStore) Assess(room, alias string) (*Assessment, error) {
	mu := fs.mu.get(room)
	mu.Lock()
	defer mu.Unlock()

	if _, err := fs.getRoomUnlocked(room); err != nil {
		return nil, err
	}

	a := &Assessment{
		Room:         room,
		Dimensions:   map[string]DimStatus{},
		OpenRFCs:     []int{},
		ApprovedRFCs: []int{},
		Agents:       []string{},
		Warnings:     []string{},
	}

	// Manifesto readiness
	a.ManifestoReady = fs.isManifestoReady(room)

	// Agents
	agents, _ := fs.getAgentsUnlocked(room)
	for _, ag := range agents {
		a.Agents = append(a.Agents, ag.Alias)
	}

	// Status board → phase, dimensions, open RFCs
	board, _ := fs.getStatusUnlocked(room)
	for _, e := range board.Entries {
		switch {
		case e.Key == "phase":
			switch e.Value {
			case "0":
				a.Phase = 0
			case "1":
				a.Phase = 1
			case "2":
				a.Phase = 2
			case "3":
				a.Phase = 3
			case "4":
				a.Phase = 4
			case "5":
				a.Phase = 5
			}
		case strings.HasPrefix(e.Key, "dim-"):
			a.Dimensions[e.Key] = DimStatus{Status: e.Value, Owner: e.UpdatedBy}
		case strings.HasPrefix(e.Key, "rfc-"):
			numStr := strings.TrimPrefix(e.Key, "rfc-")
			var num int
			for _, c := range numStr {
				if c >= '0' && c <= '9' {
					num = num*10 + int(c-'0')
				}
			}
			if num > 0 {
				if e.Value == "open" {
					a.OpenRFCs = append(a.OpenRFCs, num)
				} else if e.Value == "approved" || e.Value == "closed" {
					a.ApprovedRFCs = append(a.ApprovedRFCs, num)
				}
			}
		}
	}

	// If no phase set but manifesto not ready, it's phase 0
	if !a.ManifestoReady {
		a.Phase = 0
	}

	// Warnings: detect inconsistencies
	if !a.ManifestoReady && len(a.ApprovedRFCs) > 0 {
		a.Warnings = append(a.Warnings, "MANIFESTO.md still has placeholder but RFCs have been approved — someone forgot to update the doc")
	}
	if a.Phase > 0 && !a.ManifestoReady {
		a.Warnings = append(a.Warnings, "phase advanced past Discovery but MANIFESTO.md was never updated")
	}
	if len(a.Dimensions) > 0 {
		for k, d := range a.Dimensions {
			if d.Status == "RED" && a.Phase >= 3 {
				a.Warnings = append(a.Warnings, k+" is still RED in Phase 3+ (Evidence) — should be at least YELLOW")
			}
		}
	}
	if len(a.Dimensions) == 0 && a.ManifestoReady {
		// Check if manifesto mentions dimensions but none are on the board
		if fs.manifestoHasDimensions(room) {
			a.Warnings = append(a.Warnings, "MANIFESTO.md defines quality dimensions but none are tracked on the status board — run: agent-hub status <room> --update \"dim-NAME=RED\" --from <alias>")
		}
	}
	if len(a.Dimensions) == 0 && a.Phase >= 1 {
		a.Warnings = append(a.Warnings, "no dimensions registered — Phase 1+ requires dimension owners")
	}

	// Unread count for this agent
	if alias != "" {
		cursor, _ := fs.getCursorUnlocked(room, alias)
		dir := fs.messagesDir(room)
		ids, _ := fs.listMessageIDsUnlocked(dir)
		for _, id := range ids {
			if id > cursor {
				a.YourUnread++
			}
		}
		a.TotalMessages = len(ids)
	} else {
		msgCount, _ := fs.messageCountUnlocked(room)
		a.TotalMessages = msgCount
	}

	// Last activity
	if t := fs.lastActivityUnlocked(room); t != nil {
		a.LastActivity = t.Format("2006-01-02T15:04:05Z")
	}

	// Phase label
	phaseLabels := map[int]string{
		0: "Discovery", 1: "Setup", 2: "Investigation",
		3: "Evidence", 4: "Review", 5: "Sign-off",
	}
	a.PhaseLabel = phaseLabels[a.Phase]

	// Evaluate readiness to advance
	a.evaluateAdvance()

	return a, nil
}

// evaluateAdvance checks if the current phase transition criteria are met.
func (a *Assessment) evaluateAdvance() {
	switch a.Phase {
	case 0: // Discovery → Setup
		if !a.ManifestoReady {
			a.AdvanceBlocker = "manifesto goal is still a placeholder"
		} else if len(a.OpenRFCs) > 0 {
			a.AdvanceBlocker = "open RFCs must be resolved"
		} else {
			a.ReadyToAdvance = true
		}
	case 1: // Setup → Investigation (needs human GO)
		allAssigned := len(a.Dimensions) > 0
		for _, d := range a.Dimensions {
			if d.Owner == "" {
				allAssigned = false
				break
			}
		}
		if len(a.OpenRFCs) > 0 {
			a.AdvanceBlocker = "open RFCs must be resolved"
		} else if !allAssigned {
			a.AdvanceBlocker = "all dimensions need owners"
		} else {
			a.ReadyToAdvance = true
			a.HumanApproval = true // Phase 1→2 requires human GO
		}
	case 2: // Investigation → Evidence
		allYellowPlus := len(a.Dimensions) > 0
		for _, d := range a.Dimensions {
			if d.Status == "RED" {
				allYellowPlus = false
				break
			}
		}
		if len(a.OpenRFCs) > 0 {
			a.AdvanceBlocker = "open RFCs must be resolved"
		} else if !allYellowPlus {
			a.AdvanceBlocker = "all dimensions must be at least YELLOW"
		} else {
			a.ReadyToAdvance = true
		}
	case 3: // Evidence → Review
		allGreen := len(a.Dimensions) > 0
		for _, d := range a.Dimensions {
			if d.Status != "GREEN" {
				allGreen = false
				break
			}
		}
		if !allGreen {
			a.AdvanceBlocker = "all dimensions must be GREEN"
		} else {
			a.ReadyToAdvance = true
		}
	case 4: // Review → Sign-off
		if len(a.OpenRFCs) > 0 {
			a.AdvanceBlocker = "open RFCs must be resolved"
		} else {
			a.ReadyToAdvance = true
			a.HumanApproval = true // Phase 4→5 requires human sign-off
		}
	}
}

// isManifestoReady checks if the manifesto goal has been filled in.
func (fs *FileStore) isManifestoReady(room string) bool {
	path := filepath.Join(fs.roomDir(room), "docs", "MANIFESTO.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	content := string(data)
	// Default template has "<!-- Describe the shared objective for this room -->"
	if strings.Contains(content, "<!-- Describe the shared objective") {
		return false
	}
	// Check if goal section is empty (just "## Goal" followed by blank or next section)
	idx := strings.Index(content, "## Goal")
	if idx < 0 {
		return false
	}
	after := content[idx+len("## Goal"):]
	nextSection := strings.Index(after, "\n## ")
	if nextSection >= 0 {
		after = after[:nextSection]
	}
	trimmed := strings.TrimSpace(after)
	return trimmed != "" && !strings.HasPrefix(trimmed, "<!--")
}

// manifestoHasDimensions checks if the manifesto contains a dimensions table.
func (fs *FileStore) manifestoHasDimensions(room string) bool {
	path := filepath.Join(fs.roomDir(room), "docs", "MANIFESTO.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	content := strings.ToLower(string(data))
	return strings.Contains(content, "dimension") && strings.Contains(content, "|")
}
