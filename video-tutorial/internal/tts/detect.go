package tts

import (
	"fmt"

	"github.com/zaguerinho/claude-switch/video-tutorial/internal/config"
)

// Backend represents a TTS backend choice.
type Backend string

const (
	BackendAuto       Backend = "auto"
	BackendPiper      Backend = "piper"
	BackendElevenLabs Backend = "elevenlabs"
)

// DetectProvider auto-detects the best available TTS provider based on
// environment and installed tools. Priority:
//  1. If ElevenLabs keys are set → use ElevenLabs (higher quality)
//  2. If Piper is installed → use Piper (free, local)
//  3. Error with instructions for both options
func DetectProvider(cfg *config.Config) (Provider, error) {
	// ElevenLabs available?
	if cfg.ElevenLabsAPIKey != "" && cfg.ElevenLabsVoiceID != "" {
		return NewElevenLabs(cfg), nil
	}

	// Piper available?
	if IsPiperInstalled() {
		EnsurePiperModel(cfg.PiperModel)
		return NewPiper(cfg), nil
	}

	return nil, fmt.Errorf("no TTS provider available\n\n" +
		"Option 1 — Piper (free, local, no API key needed):\n" +
		"  pip install piper-tts\n\n" +
		"Option 2 — ElevenLabs (cloud, higher quality):\n" +
		"  export ELEVENLABS_API_KEY=your-key\n" +
		"  export ELEVENLABS_VOICE_ID=your-voice-id\n")
}

// NewProvider creates a specific TTS provider by name.
func NewProvider(backend Backend, cfg *config.Config) (Provider, error) {
	switch backend {
	case BackendAuto:
		return DetectProvider(cfg)
	case BackendPiper:
		if !IsPiperInstalled() {
			return nil, fmt.Errorf("piper not found on PATH\n\n%s", InstallPiperHint())
		}
		EnsurePiperModel(cfg.PiperModel)
		return NewPiper(cfg), nil
	case BackendElevenLabs:
		if cfg.ElevenLabsAPIKey == "" || cfg.ElevenLabsVoiceID == "" {
			return nil, fmt.Errorf("ElevenLabs requires ELEVENLABS_API_KEY and ELEVENLABS_VOICE_ID environment variables")
		}
		return NewElevenLabs(cfg), nil
	default:
		return nil, fmt.Errorf("unknown TTS backend: %q (use: auto, piper, elevenlabs)", backend)
	}
}
