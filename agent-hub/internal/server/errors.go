package server

import (
	"encoding/json"
	"net/http"

	"github.com/zaguerinho/claude-switch/agent-hub/internal/model"
)

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeOK writes a success response.
func writeOK(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, model.APIResponse{OK: true, Data: data})
}

// writeCreated writes a 201 response.
func writeCreated(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusCreated, model.APIResponse{OK: true, Data: data})
}

// writeError writes an error response with appropriate status code.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, model.APIResponse{OK: false, Error: msg})
}

// writeNotFound writes a 404 error.
func writeNotFound(w http.ResponseWriter, msg string) {
	writeError(w, http.StatusNotFound, msg)
}

// writeBadRequest writes a 400 error.
func writeBadRequest(w http.ResponseWriter, msg string) {
	writeError(w, http.StatusBadRequest, msg)
}

// writeConflict writes a 409 error.
func writeConflict(w http.ResponseWriter, msg string) {
	writeError(w, http.StatusConflict, msg)
}
