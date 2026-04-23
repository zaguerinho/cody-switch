package server

import (
	"encoding/json"
	"net/http"
)

// handleHealth returns a handler that reports server status and version.
// The browser uses this to detect whether the local server is running.
func handleHealth(version string) http.HandlerFunc {
	resp, _ := json.Marshal(map[string]string{
		"status":  "ok",
		"version": version,
	})

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(resp)
	}
}
