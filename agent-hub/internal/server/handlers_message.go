package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/zaguerinho/claude-switch/agent-hub/internal/model"
)

func (s *Server) handlePostMessage(w http.ResponseWriter, r *http.Request) {
	room := r.PathValue("name")
	var req struct {
		From    string `json:"from"`
		Type    string `json:"type"`
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "invalid JSON body")
		return
	}
	if req.From == "" {
		writeBadRequest(w, "from is required")
		return
	}
	if req.Body == "" {
		writeBadRequest(w, "body cannot be empty")
		return
	}

	msgType := model.MessageType(req.Type)
	if req.Type == "" {
		msgType = model.MessageNote
	}
	if !msgType.IsValid() {
		writeBadRequest(w, "invalid message type: "+req.Type+"; use: question, answer, rfc, note, status-update")
		return
	}

	msg, err := s.store.PostMessage(room, req.From, msgType, req.Subject, req.Body)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeNotFound(w, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeCreated(w, msg)
}

func (s *Server) handleReadMessages(w http.ResponseWriter, r *http.Request) {
	room := r.PathValue("name")

	// Check for unread mode
	alias := r.URL.Query().Get("as")
	unread := r.URL.Query().Get("unread") == "true"

	if unread && alias != "" {
		msgs, err := s.store.ReadUnread(room, alias)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if msgs == nil {
			msgs = []model.Message{}
		}
		writeOK(w, msgs)
		return
	}

	last := 0
	if v := r.URL.Query().Get("last"); v != "" {
		last, _ = strconv.Atoi(v)
	}

	msgs, err := s.store.ReadMessages(room, last)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if msgs == nil {
		msgs = []model.Message{}
	}
	writeOK(w, msgs)
}

func (s *Server) handleCheckMessages(w http.ResponseWriter, r *http.Request) {
	room := r.PathValue("name")
	alias := r.URL.Query().Get("as")
	if alias == "" {
		writeBadRequest(w, "as query parameter is required")
		return
	}

	count, latest, err := s.store.UnreadCount(room, alias)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := model.CheckResult{
		Room: room,
		Unread:  count,
	}
	if latest != nil {
		result.Latest = latest.Subject
		if result.Latest == "" {
			// Truncate body for summary
			body := latest.Body
			if len(body) > 80 {
				body = body[:80] + "..."
			}
			result.Latest = body
		}
		result.LatestID = latest.ID
	}
	writeOK(w, result)
}

func (s *Server) handleAckMessages(w http.ResponseWriter, r *http.Request) {
	room := r.PathValue("name")
	var req struct {
		Alias string `json:"alias"`
		UpTo  int    `json:"up_to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "invalid JSON body")
		return
	}
	if req.Alias == "" {
		writeBadRequest(w, "alias is required")
		return
	}

	if err := s.store.Acknowledge(room, req.Alias, req.UpTo); err != nil {
		if strings.Contains(err.Error(), "cannot ack") {
			writeBadRequest(w, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, map[string]any{"acknowledged": req.UpTo, "alias": req.Alias})
}

func (s *Server) handleCheckAll(w http.ResponseWriter, r *http.Request) {
	alias := r.URL.Query().Get("as")
	if alias == "" {
		writeBadRequest(w, "as query parameter is required")
		return
	}

	roomNames, err := s.store.GetAgentRooms(alias)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var results []model.CheckResult
	for _, name := range roomNames {
		count, latest, err := s.store.UnreadCount(name, alias)
		if err != nil {
			continue
		}
		cr := model.CheckResult{Room: name, Unread: count}
		if latest != nil {
			cr.Latest = latest.Subject
			if cr.Latest == "" {
				body := latest.Body
				if len(body) > 80 {
					body = body[:80] + "..."
				}
				cr.Latest = body
			}
			cr.LatestID = latest.ID
		}
		results = append(results, cr)
	}

	writeOK(w, model.CheckAllResult{Rooms: results})
}
