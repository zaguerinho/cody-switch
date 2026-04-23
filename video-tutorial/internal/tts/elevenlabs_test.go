package tts

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/zaguerinho/claude-switch/video-tutorial/internal/config"
)

func fakeElevenLabsResponse() elevenLabsResponse {
	return elevenLabsResponse{
		AudioBase64: "AAAA", // fake base64
		Alignment: Alignment{
			Characters: []string{"H", "i"},
			Starts:     []float64{0, 0.1},
			Ends:       []float64{0.1, 0.2},
		},
	}
}

func testConfig(serverURL string) *config.Config {
	cfg := config.New()
	cfg.ElevenLabsBaseURL = serverURL
	cfg.ElevenLabsAPIKey = "test-key"
	cfg.ElevenLabsVoiceID = "test-voice"
	cfg.HTTPTimeoutSeconds = 10
	cfg.RetryMax = 2
	cfg.RetryBaseDelayMS = 10
	cfg.SegmentGapMS = 10
	return cfg
}

func TestElevenLabs_SynthesizeSegment_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("xi-api-key") != "test-key" {
			t.Errorf("missing xi-api-key header")
		}
		json.NewEncoder(w).Encode(fakeElevenLabsResponse())
	}))
	defer ts.Close()

	s := NewElevenLabs(testConfig(ts.URL))
	result, err := s.SynthesizeSegment(context.Background(), "Hi there")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.AudioBase64 != "AAAA" {
		t.Errorf("audio = %q, want AAAA", result.AudioBase64)
	}
	if len(result.WordTimestamps) != 1 {
		t.Errorf("words = %d, want 1", len(result.WordTimestamps))
	}
	if result.WordTimestamps[0].Word != "Hi" {
		t.Errorf("word = %q, want Hi", result.WordTimestamps[0].Word)
	}
}

func TestElevenLabs_SynthesizeSegment_MissingAudio(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"audio_base64": ""})
	}))
	defer ts.Close()

	s := NewElevenLabs(testConfig(ts.URL))
	_, err := s.SynthesizeSegment(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for missing audio")
	}
}

func TestElevenLabs_Name(t *testing.T) {
	s := NewElevenLabs(config.New())
	if s.Name() != "ElevenLabs" {
		t.Errorf("Name() = %q, want ElevenLabs", s.Name())
	}
}

func TestSynthesizeAll_SequentialWithProgress(t *testing.T) {
	var calls atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		json.NewEncoder(w).Encode(fakeElevenLabsResponse())
	}))
	defer ts.Close()

	provider := NewElevenLabs(testConfig(ts.URL))
	segments := []SegmentInput{
		{ID: "s1", Text: "Hello"},
		{ID: "s2", Text: "World"},
		{ID: "s3", Text: "Test"},
	}

	var progressCalls int
	var resultIDs []string

	results, err := SynthesizeAll(
		context.Background(),
		provider,
		segments,
		10, // fast gap for test
		func(completed, total int, id string) { progressCalls++ },
		func(r SynthResult) { resultIDs = append(resultIDs, r.ID) },
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("results = %d, want 3", len(results))
	}
	if progressCalls != 3 {
		t.Errorf("progressCalls = %d, want 3", progressCalls)
	}
	if len(resultIDs) != 3 || resultIDs[0] != "s1" || resultIDs[2] != "s3" {
		t.Errorf("resultIDs = %v", resultIDs)
	}
	if calls.Load() != 3 {
		t.Errorf("API calls = %d, want 3", calls.Load())
	}
}

func TestSynthesizeAll_RetriesOnRateLimit(t *testing.T) {
	var calls atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			w.WriteHeader(429)
			w.Write([]byte("rate limited"))
			return
		}
		json.NewEncoder(w).Encode(fakeElevenLabsResponse())
	}))
	defer ts.Close()

	provider := NewElevenLabs(testConfig(ts.URL))
	results, err := SynthesizeAll(
		context.Background(),
		provider,
		[]SegmentInput{{ID: "s1", Text: "Hi"}},
		10, nil, nil,
	)
	if err != nil {
		t.Fatalf("expected retry to succeed, got: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results = %d, want 1", len(results))
	}
}

func TestSynthesizeAll_ContextCancel(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(fakeElevenLabsResponse())
	}))
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	provider := NewElevenLabs(testConfig(ts.URL))
	segments := []SegmentInput{
		{ID: "s1", Text: "Hello"},
		{ID: "s2", Text: "World"},
	}

	_, err := SynthesizeAll(ctx, provider, segments, 1000,
		func(completed, total int, id string) {
			if completed == 1 {
				cancel()
			}
		}, nil,
	)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}
