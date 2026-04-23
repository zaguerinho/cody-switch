// Package httpclient provides a shared HTTP client with timeout, retry, and
// rate-limit handling. Used exclusively for ElevenLabs API calls — manifest
// generation uses the Claude CLI instead.
package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"
)

// APIError is returned when the remote API responds with a non-2xx status.
type APIError struct {
	StatusCode int
	Body       string
	Retryable  bool
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d: %s", e.StatusCode, truncate(e.Body, 300))
}

// Client wraps net/http.Client with retry and timeout logic.
type Client struct {
	HTTPClient   *http.Client
	RetryMax     int
	BaseDelayMS  int
}

// New creates a Client with the given timeout and retry settings.
func New(timeoutSeconds, retryMax, baseDelayMS int) *Client {
	return &Client{
		HTTPClient: &http.Client{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
		},
		RetryMax:    retryMax,
		BaseDelayMS: baseDelayMS,
	}
}

// PostJSON sends a JSON POST request and returns the raw response body.
// Retries on 429 (rate limit) and 5xx errors with exponential backoff.
func (c *Client) PostJSON(ctx context.Context, url string, headers map[string]string, body interface{}) ([]byte, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request body: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= c.RetryMax; attempt++ {
		if attempt > 0 {
			delay := time.Duration(float64(c.BaseDelayMS)*math.Pow(2, float64(attempt-1))) * time.Millisecond
			fmt.Fprintf(io.Discard, "  Retry %d/%d after %v...\n", attempt, c.RetryMax, delay)

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("read response: %w", err)
			continue
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return respBody, nil
		}

		apiErr := &APIError{
			StatusCode: resp.StatusCode,
			Body:       string(respBody),
			Retryable:  resp.StatusCode == 429 || resp.StatusCode >= 500,
		}

		if apiErr.Retryable && attempt < c.RetryMax {
			lastErr = apiErr
			continue
		}

		return nil, apiErr
	}

	return nil, fmt.Errorf("exhausted %d retries: %w", c.RetryMax, lastErr)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
