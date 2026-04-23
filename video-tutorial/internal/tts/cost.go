package tts

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/zaguerinho/claude-switch/video-tutorial/internal/config"
)

// ElevenLabs pricing tiers (USD per character).
var pricingPerChar = map[string]float64{
	"free":    0,          // 10,000 chars/month included
	"starter": 5.0 / 30000,
	"creator": 22.0 / 100000,
	"pro":     99.0 / 500000,
}

// subscription holds the relevant fields from the ElevenLabs subscription API.
type subscription struct {
	Tier           string `json:"tier"`
	CharacterLimit int    `json:"character_limit"`
	CharacterCount int    `json:"character_count"`
}

// ConfirmCostEstimate shows the USD cost estimate and asks the user to confirm.
// Returns true if the user confirms, false if they cancel.
// If skipConfirm is true, shows the estimate but doesn't prompt.
func ConfirmCostEstimate(segments []SegmentInput, cfg *config.Config, skipConfirm bool) bool {
	totalChars := 0
	for _, seg := range segments {
		totalChars += len(seg.Text)
	}

	fmt.Fprintf(os.Stderr, "\n┌─ ElevenLabs Cost Estimate ─────────────────────┐\n")
	fmt.Fprintf(os.Stderr, "│  Segments:    %-6d                             │\n", len(segments))
	fmt.Fprintf(os.Stderr, "│  Characters:  %-6d                             │\n", totalChars)

	// Try to fetch remaining credits.
	remaining, limit, tier := fetchRemainingCredits(cfg)

	if limit > 0 {
		afterUse := remaining - totalChars
		rate := pricingPerChar[strings.ToLower(tier)]
		cost := float64(totalChars) * rate

		fmt.Fprintf(os.Stderr, "│                                                │\n")
		fmt.Fprintf(os.Stderr, "│  Plan:        %-10s (%d/month)       │\n", tier, limit)
		fmt.Fprintf(os.Stderr, "│  Remaining:   %-6d credits                    │\n", remaining)
		fmt.Fprintf(os.Stderr, "│  After build: %-6d credits                    │\n", afterUse)
		if cost == 0 {
			fmt.Fprintf(os.Stderr, "│  Est. cost:   $0.00 (included in free tier)     │\n")
		} else {
			fmt.Fprintf(os.Stderr, "│  Est. cost:   $%-6.2f                           │\n", cost)
		}

		if afterUse < 0 {
			fmt.Fprintf(os.Stderr, "│                                                │\n")
			fmt.Fprintf(os.Stderr, "│  ⚠  Not enough credits! Need %d more.        │\n", -afterUse)
		}
	} else {
		// Couldn't fetch plan — show estimates for each tier.
		fmt.Fprintf(os.Stderr, "│                                                │\n")
		fmt.Fprintf(os.Stderr, "│  Estimated cost by plan:                       │\n")
		fmt.Fprintf(os.Stderr, "│    Free tier:   $0.00  (10k chars/month)       │\n")
		fmt.Fprintf(os.Stderr, "│    Starter:     $%.2f  ($5/30k chars)          │\n",
			float64(totalChars)*pricingPerChar["starter"])
		fmt.Fprintf(os.Stderr, "│    Creator:     $%.2f  ($22/100k chars)        │\n",
			float64(totalChars)*pricingPerChar["creator"])
		fmt.Fprintf(os.Stderr, "│    Pro:         $%.2f  ($99/500k chars)        │\n",
			float64(totalChars)*pricingPerChar["pro"])
	}

	fmt.Fprintf(os.Stderr, "└────────────────────────────────────────────────┘\n")

	if skipConfirm {
		return true
	}

	fmt.Fprintf(os.Stderr, "\nProceed with synthesis? [Y/n] ")
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	return answer == "" || answer == "y" || answer == "yes"
}

// fetchRemainingCredits calls the ElevenLabs subscription API.
// Returns (remaining, limit, tier). Returns (0, 0, "") on any failure.
func fetchRemainingCredits(cfg *config.Config) (int, int, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := cfg.ElevenLabsBaseURL + "/v1/user/subscription"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, 0, ""
	}
	req.Header.Set("xi-api-key", cfg.ElevenLabsAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return 0, 0, ""
	}
	defer resp.Body.Close()

	var sub subscription
	if err := json.NewDecoder(resp.Body).Decode(&sub); err != nil {
		return 0, 0, ""
	}

	remaining := sub.CharacterLimit - sub.CharacterCount
	return remaining, sub.CharacterLimit, sub.Tier
}
