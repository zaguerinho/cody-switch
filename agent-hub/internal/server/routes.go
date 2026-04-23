package server

import (
	"io/fs"
	"net/http"

	"github.com/zaguerinho/claude-switch/agent-hub/embed"
)

// registerAPIRoutes adds all JSON API routes to the given mux.
func (s *Server) registerAPIRoutes(mux *http.ServeMux) {
	// Health + version
	mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		writeOK(w, map[string]string{"status": "ok", "version": s.version})
	})

	// Rooms
	mux.HandleFunc("POST /api/v1/rooms", s.handleCreateRoom)
	mux.HandleFunc("GET /api/v1/rooms", s.handleListRooms)
	mux.HandleFunc("GET /api/v1/rooms/check-all", s.handleCheckAll)
	mux.HandleFunc("GET /api/v1/rooms/{name}", s.handleGetRoom)
	mux.HandleFunc("POST /api/v1/rooms/{name}/archive", s.handleArchiveRoom)

	// Agents
	mux.HandleFunc("POST /api/v1/rooms/{name}/agents", s.handleJoinRoom)
	mux.HandleFunc("DELETE /api/v1/rooms/{name}/agents/{alias}", s.handleLeaveRoom)
	mux.HandleFunc("GET /api/v1/rooms/{name}/agents", s.handleListAgents)

	// Messages
	mux.HandleFunc("POST /api/v1/rooms/{name}/messages", s.handlePostMessage)
	mux.HandleFunc("GET /api/v1/rooms/{name}/messages", s.handleReadMessages)
	mux.HandleFunc("GET /api/v1/rooms/{name}/messages/check", s.handleCheckMessages)
	mux.HandleFunc("POST /api/v1/rooms/{name}/messages/ack", s.handleAckMessages)

	// Status
	mux.HandleFunc("GET /api/v1/rooms/{name}/status", s.handleGetStatus)
	mux.HandleFunc("PUT /api/v1/rooms/{name}/status", s.handleUpdateStatus)

	// Assess
	mux.HandleFunc("GET /api/v1/rooms/{name}/assess", s.handleAssess)

	// Docs
	mux.HandleFunc("GET /api/v1/rooms/{name}/docs", s.handleListDocs)
	mux.HandleFunc("GET /api/v1/rooms/{name}/docs/{doc}", s.handleReadDoc)
	mux.HandleFunc("PUT /api/v1/rooms/{name}/docs/{doc}", s.handleUpdateDoc)
}

// apiRoutes returns a mux with only API routes.
func (s *Server) apiRoutes() *http.ServeMux {
	mux := http.NewServeMux()
	s.registerAPIRoutes(mux)
	return mux
}

// dashboardRoutes returns a mux with API + dashboard UI routes.
func (s *Server) dashboardRoutes() *http.ServeMux {
	mux := http.NewServeMux()
	s.registerAPIRoutes(mux)

	// Serve embedded static assets
	staticFS, _ := fs.Sub(embed.Assets, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Dashboard SPA — serve index.html for all non-API, non-static routes
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		html, err := embed.ReadAsset("templates/index.html")
		if err != nil {
			http.Error(w, "internal error", 500)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(html)
	})

	return mux
}
