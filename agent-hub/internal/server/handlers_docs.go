package server

import (
	"encoding/json"
	"net/http"
	"strings"
)

func (s *Server) handleListDocs(w http.ResponseWriter, r *http.Request) {
	room := r.PathValue("name")
	docs, err := s.store.ListDocs(room)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if docs == nil {
		writeOK(w, []struct{}{})
		return
	}
	writeOK(w, docs)
}

func (s *Server) handleReadDoc(w http.ResponseWriter, r *http.Request) {
	room := r.PathValue("name")
	doc := r.PathValue("doc")

	content, err := s.store.ReadDoc(room, doc)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeNotFound(w, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, map[string]string{"name": doc, "content": content})
}

func (s *Server) handleUpdateDoc(w http.ResponseWriter, r *http.Request) {
	room := r.PathValue("name")
	doc := r.PathValue("doc")

	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "invalid JSON body")
		return
	}
	if req.Content == "" {
		writeBadRequest(w, "content is required")
		return
	}

	if err := s.store.WriteDoc(room, doc, req.Content); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, map[string]string{"name": doc, "updated": "true"})
}
