package server

import (
	"encoding/json"
	"net/http"
	"strings"
)

func (s *Server) handleJoinRoom(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var req struct {
		Alias string `json:"alias"`
		Role  string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "invalid JSON body")
		return
	}
	if req.Alias == "" {
		writeBadRequest(w, "alias is required")
		return
	}

	agent, err := s.store.JoinRoom(name, req.Alias, req.Role)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeNotFound(w, err.Error())
			return
		}
		if strings.Contains(err.Error(), "already joined") {
			writeConflict(w, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeCreated(w, agent)
}

func (s *Server) handleLeaveRoom(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	alias := r.PathValue("alias")

	if err := s.store.LeaveRoom(name, alias); err != nil {
		if strings.Contains(err.Error(), "not a member") {
			writeNotFound(w, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, map[string]string{"removed": alias})
}

func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	agents, err := s.store.GetAgents(name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if agents == nil {
		writeOK(w, []struct{}{})
		return
	}
	writeOK(w, agents)
}
