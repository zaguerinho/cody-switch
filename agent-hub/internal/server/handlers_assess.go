package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleAssess(w http.ResponseWriter, r *http.Request) {
	room := r.PathValue("name")
	alias := r.URL.Query().Get("as")

	assessment, err := s.store.Assess(room, alias)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeNotFound(w, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, assessment)
}
