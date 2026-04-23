// Package config provides central configuration for the video-tutorial binary.
// It replaces all hardcoded constants from the Node.js implementation with a
// single Config struct, populated with sensible defaults and overridable via
// environment variables.
package config

import (
	"errors"
	"os"
	"os/exec"
	"strings"
)

// Config holds every tunable knob for the video-tutorial pipeline.
// Manifest generation uses the Claude CLI (`claude -p`), so no Anthropic API
// key or endpoint configuration is needed.
type Config struct {
	// TTS backend selection: "auto", "piper", "elevenlabs"
	TTSBackend string

	// ElevenLabs (cloud TTS)
	ElevenLabsBaseURL string
	ElevenLabsModel   string
	ElevenLabsAPIKey  string
	ElevenLabsVoiceID string

	// Piper (local TTS — free, no API key needed)
	PiperModel      string // e.g. "en_US-lessac-medium" (auto-downloads on first use)
	PiperDataDir    string // override model directory
	PiperSampleRate int    // model sample rate, default 22050

	// Tunable constants
	SegmentGapMS       int
	RetryMax           int
	RetryBaseDelayMS   int
	HTTPTimeoutSeconds int

	// Voice settings (ElevenLabs only)
	VoiceStability       float64
	VoiceSimilarityBoost float64

	// Paths
	OutputDir string
}

// New returns a Config populated with production defaults.
// Call LoadEnv afterwards to pull secrets from the environment.
func New() *Config {
	return &Config{
		TTSBackend: "auto",

		ElevenLabsBaseURL: "https://api.elevenlabs.io",
		ElevenLabsModel:   "eleven_multilingual_v2",

		PiperModel:      "en_US-lessac-medium",
		PiperSampleRate: 22050,

		SegmentGapMS:       300,
		RetryMax:           3,
		RetryBaseDelayMS:   1000,
		HTTPTimeoutSeconds: 60,

		VoiceStability:       0.5,
		VoiceSimilarityBoost: 0.75,
	}
}

// LoadEnv reads API keys and the voice ID from environment variables.
// It does not return an error — call Validate to check for missing values.
func (c *Config) LoadEnv() {
	if v := os.Getenv("ELEVENLABS_API_KEY"); v != "" {
		c.ElevenLabsAPIKey = v
	}
	if v := os.Getenv("ELEVENLABS_VOICE_ID"); v != "" {
		c.ElevenLabsVoiceID = v
	}
	if v := os.Getenv("PIPER_MODEL"); v != "" {
		c.PiperModel = v
	}
	if v := os.Getenv("PIPER_DATA_DIR"); v != "" {
		c.PiperDataDir = v
	}
}

// Validate checks that required fields are present based on the TTS backend.
// For "auto" mode, at least one backend must be available (checked at runtime).
// Note: no Anthropic API key needed — manifest generation uses the Claude CLI.
func (c *Config) Validate() error {
	switch c.TTSBackend {
	case "elevenlabs":
		var missing []string
		if c.ElevenLabsAPIKey == "" {
			missing = append(missing, "ELEVENLABS_API_KEY")
		}
		if c.ElevenLabsVoiceID == "" {
			missing = append(missing, "ELEVENLABS_VOICE_ID")
		}
		if len(missing) > 0 {
			return errors.New("missing required environment variables: " + strings.Join(missing, ", "))
		}
	case "piper":
		// Piper needs no API keys; binary availability checked at runtime.
	case "auto", "":
		// Auto-detect at runtime; no upfront validation needed.
	default:
		return errors.New("unknown TTS backend: " + c.TTSBackend + " (use: auto, piper, elevenlabs)")
	}
	return nil
}

// ValidateCLI checks that the Claude CLI is available on PATH.
func ValidateCLI() error {
	_, err := exec.LookPath("claude")
	if err != nil {
		return errors.New("claude CLI not found on PATH — install Claude Code first: https://claude.ai/code")
	}
	return nil
}
