package server

import (
	"encoding/json"
	"net/http"
	"strings"
)

func (s *Server) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	room := r.PathValue("name")
	board, err := s.store.GetStatus(room)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeNotFound(w, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, board)
}

func (s *Server) handleUpdateStatus(w http.ResponseWriter, r *http.Request) {
	room := r.PathValue("name")
	var req struct {
		Key       string `json:"key"`
		Value     string `json:"value"`
		UpdatedBy string `json:"updated_by"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "invalid JSON body")
		return
	}
	if req.Key == "" {
		writeBadRequest(w, "key is required")
		return
	}
	if req.UpdatedBy == "" {
		writeBadRequest(w, "updated_by is required")
		return
	}

	if err := s.store.UpdateStatus(room, req.Key, req.Value, req.UpdatedBy); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Auto-append to STATUS.md activity log
	s.store.AppendToStatusDoc(room, req.Key, req.Value, req.UpdatedBy)

	writeOK(w, map[string]string{"key": req.Key, "value": req.Value})
}
