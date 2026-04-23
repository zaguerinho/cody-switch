package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/zaguerinho/claude-switch/agent-hub/embed"
)

func (s *Server) handleCreateRoom(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "invalid JSON body")
		return
	}
	if req.Name == "" {
		writeBadRequest(w, "name is required")
		return
	}

	meta, err := s.store.CreateRoom(req.Name, req.Description)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			writeConflict(w, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Scaffold governance docs
	s.store.ScaffoldDocs(req.Name, embed.GovernanceTemplates())

	writeCreated(w, meta)
}

func (s *Server) handleListRooms(w http.ResponseWriter, r *http.Request) {
	all := r.URL.Query().Get("all") == "true"

	var rooms any
	var err error
	if all {
		rooms, err = s.store.ListAllRooms()
	} else {
		rooms, err = s.store.ListRooms()
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, rooms)
}

func (s *Server) handleGetRoom(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	info, err := s.store.GetRoomInfo(name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeNotFound(w, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, info)
}

func (s *Server) handleArchiveRoom(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := s.store.ArchiveRoom(name); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeNotFound(w, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, map[string]string{"archived": name})
}
