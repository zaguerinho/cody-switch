package tts

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Voice is a curated ElevenLabs voice option.
type Voice struct {
	ID          string
	Name        string
	Gender      string
	Accent      string
	Description string
}

// CuratedVoices is a list of high-quality ElevenLabs premade voices
// available to all users (including free tier).
var CuratedVoices = []Voice{
	{ID: "pNInz6obpgDQGcFmaJgB", Name: "Adam", Gender: "Male", Accent: "American", Description: "Deep, authoritative — great for technical narration"},
	{ID: "ErXwobaYiN019PkySvjV", Name: "Antoni", Gender: "Male", Accent: "American", Description: "Warm, conversational — friendly and approachable"},
	{ID: "VR6AewLTigWG4xSOukaG", Name: "Arnold", Gender: "Male", Accent: "American", Description: "Strong, confident — clear and commanding"},
	{ID: "21m00Tcm4TlvDq8ikWAM", Name: "Rachel", Gender: "Female", Accent: "American", Description: "Calm, clear — professional and polished"},
	{ID: "AZnzlk1XvdvUeBnXmlld", Name: "Domi", Gender: "Female", Accent: "American", Description: "Energetic, expressive — engaging delivery"},
	{ID: "EXAVITQu4vr4xnSDxMaL", Name: "Bella", Gender: "Female", Accent: "American", Description: "Soft, gentle — soothing and articulate"},
	{ID: "MF3mGyEYCl7XYWbV9V6O", Name: "Elli", Gender: "Female", Accent: "American", Description: "Young, bright — upbeat and clear"},
	{ID: "jBpfuIE2acCO8z3wKNLl", Name: "Gigi", Gender: "Female", Accent: "American", Description: "Animated, youthful — lively and fun"},
	{ID: "onwK4e9ZLuTAKqWW03F9", Name: "Daniel", Gender: "Male", Accent: "British", Description: "Refined, articulate — BBC-style delivery"},
	{ID: "N2lVS1w4EtoT3dr4eOWO", Name: "Callum", Gender: "Male", Accent: "British", Description: "Smooth, measured — calm and professional"},
}

// PromptVoiceSelection shows a numbered list of voices and lets the user pick.
// Returns the selected voice ID. If skipPrompt is true, returns defaultVoiceID.
func PromptVoiceSelection(defaultVoiceID string, skipPrompt bool) string {
	if skipPrompt {
		if defaultVoiceID != "" {
			return defaultVoiceID
		}
		return CuratedVoices[0].ID
	}

	// Find which voice is currently the default (if any)
	defaultIdx := -1
	for i, v := range CuratedVoices {
		if v.ID == defaultVoiceID {
			defaultIdx = i
			break
		}
	}

	fmt.Fprintf(os.Stderr, "\n┌─ Select a Voice ──────────────────────────────────────────────────────┐\n")
	for i, v := range CuratedVoices {
		marker := "   "
		if i == defaultIdx {
			marker = " ▸ "
		}
		fmt.Fprintf(os.Stderr, "│%s%2d. %-10s %-7s %-9s %s│\n",
			marker, i+1, v.Name, v.Gender, v.Accent,
			padRight(v.Description, 33))
	}
	fmt.Fprintf(os.Stderr, "└───────────────────────────────────────────────────────────────────────┘\n")

	defaultLabel := "1 Adam"
	if defaultIdx >= 0 {
		defaultLabel = fmt.Sprintf("%d %s", defaultIdx+1, CuratedVoices[defaultIdx].Name)
	}
	fmt.Fprintf(os.Stderr, "\nChoose a voice [1-%d, default=%s]: ", len(CuratedVoices), defaultLabel)

	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(answer)

	if answer == "" {
		if defaultIdx >= 0 {
			selected := CuratedVoices[defaultIdx]
			fmt.Fprintf(os.Stderr, "Using: %s — %s\n", selected.Name, selected.Description)
			return selected.ID
		}
		fmt.Fprintf(os.Stderr, "Using: %s — %s\n", CuratedVoices[0].Name, CuratedVoices[0].Description)
		return CuratedVoices[0].ID
	}

	n, err := strconv.Atoi(answer)
	if err != nil || n < 1 || n > len(CuratedVoices) {
		fmt.Fprintf(os.Stderr, "Invalid choice, using default.\n")
		if defaultIdx >= 0 {
			return CuratedVoices[defaultIdx].ID
		}
		return CuratedVoices[0].ID
	}

	selected := CuratedVoices[n-1]
	fmt.Fprintf(os.Stderr, "Using: %s — %s\n", selected.Name, selected.Description)
	return selected.ID
}

func padRight(s string, n int) string {
	if len(s) >= n {
		return s[:n]
	}
	return s + strings.Repeat(" ", n-len(s))
}
