package httpclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestPostJSON_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer ts.Close()

	c := New(10, 2, 100)
	body, err := c.PostJSON(context.Background(), ts.URL, nil, map[string]string{"q": "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(body) == 0 {
		t.Fatal("expected non-empty body")
	}
}

func TestPostJSON_Retries429(t *testing.T) {
	var calls atomic.Int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n <= 2 {
			w.WriteHeader(429)
			w.Write([]byte("rate limited"))
			return
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer ts.Close()

	c := New(10, 3, 10) // fast retries for test
	body, err := c.PostJSON(context.Background(), ts.URL, nil, nil)
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if body == nil {
		t.Fatal("expected body")
	}
	if calls.Load() != 3 {
		t.Errorf("expected 3 calls, got %d", calls.Load())
	}
}

func TestPostJSON_Retries5xx(t *testing.T) {
	var calls atomic.Int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			w.WriteHeader(503)
			w.Write([]byte("service unavailable"))
			return
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer ts.Close()

	c := New(10, 2, 10)
	_, err := c.PostJSON(context.Background(), ts.URL, nil, nil)
	if err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
}

func TestPostJSON_NoRetryOn4xx(t *testing.T) {
	var calls atomic.Int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(400)
		w.Write([]byte("bad request"))
	}))
	defer ts.Close()

	c := New(10, 3, 10)
	_, err := c.PostJSON(context.Background(), ts.URL, nil, nil)
	if err == nil {
		t.Fatal("expected error on 400")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 400 {
		t.Errorf("expected status 400, got %d", apiErr.StatusCode)
	}
	if calls.Load() != 1 {
		t.Errorf("should not retry on 400, got %d calls", calls.Load())
	}
}

func TestPostJSON_ContextCancellation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
		w.Write([]byte("rate limited"))
	}))
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	c := New(10, 3, 1000)
	_, err := c.PostJSON(ctx, ts.URL, nil, nil)
	if err == nil {
		t.Fatal("expected error on cancelled context")
	}
}

func TestPostJSON_CustomHeaders(t *testing.T) {
	var gotHeader string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("xi-api-key")
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	c := New(10, 0, 100)
	headers := map[string]string{"xi-api-key": "test-key-123"}
	_, err := c.PostJSON(context.Background(), ts.URL, headers, nil)
	if err != nil {
		t.Fatal(err)
	}
	if gotHeader != "test-key-123" {
		t.Errorf("expected header test-key-123, got %q", gotHeader)
	}
}
