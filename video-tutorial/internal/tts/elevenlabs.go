package tts

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zaguerinho/claude-switch/video-tutorial/internal/config"
	"github.com/zaguerinho/claude-switch/video-tutorial/internal/httpclient"
)

// elevenLabsResponse matches the JSON returned by the ElevenLabs TTS endpoint.
type elevenLabsResponse struct {
	AudioBase64 string    `json:"audio_base64"`
	Alignment   Alignment `json:"alignment"`
}

// ElevenLabs implements the Provider interface using the ElevenLabs cloud API.
type ElevenLabs struct {
	client *httpclient.Client
	cfg    *config.Config
}

// NewElevenLabs creates an ElevenLabs provider using the given config.
func NewElevenLabs(cfg *config.Config) *ElevenLabs {
	return &ElevenLabs{
		client: httpclient.New(cfg.HTTPTimeoutSeconds, cfg.RetryMax, cfg.RetryBaseDelayMS),
		cfg:    cfg,
	}
}

// Name returns "ElevenLabs".
func (e *ElevenLabs) Name() string { return "ElevenLabs" }

// SynthesizeSegment synthesizes a single text segment via ElevenLabs TTS.
func (e *ElevenLabs) SynthesizeSegment(ctx context.Context, text string) (*SynthResult, error) {
	url := fmt.Sprintf("%s/v1/text-to-speech/%s/with-timestamps",
		e.cfg.ElevenLabsBaseURL, e.cfg.ElevenLabsVoiceID)

	headers := map[string]string{
		"xi-api-key": e.cfg.ElevenLabsAPIKey,
	}

	body := map[string]interface{}{
		"text":     text,
		"model_id": e.cfg.ElevenLabsModel,
		"voice_settings": map[string]float64{
			"stability":        e.cfg.VoiceStability,
			"similarity_boost": e.cfg.VoiceSimilarityBoost,
		},
	}

	raw, err := e.client.PostJSON(ctx, url, headers, body)
	if err != nil {
		return nil, fmt.Errorf("ElevenLabs synthesis: %w", err)
	}

	var resp elevenLabsResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("parse ElevenLabs response: %w", err)
	}

	if resp.AudioBase64 == "" {
		return nil, fmt.Errorf("response missing audio_base64 field")
	}

	words := CharacterTimestampsToWords(resp.Alignment)
	var duration float64
	if len(words) > 0 {
		duration = words[len(words)-1].End
	}

	return &SynthResult{
		AudioBase64:    resp.AudioBase64,
		WordTimestamps: words,
		Duration:       duration,
	}, nil
}
