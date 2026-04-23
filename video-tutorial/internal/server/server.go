// Package server provides a local HTTP server for interactive Q&A.
// It proxies chatbot questions from the tutorial HTML to the Claude CLI
// (`claude -p`), streaming responses back as Server-Sent Events.
package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
)

// Server is the interactive Q&A HTTP server.
type Server struct {
	Addr    string
	Version string
}

// New creates a server listening on the given address.
func New(addr, version string) *Server {
	return &Server{Addr: addr, Version: version}
}

// ListenAndServe starts the HTTP server and blocks until the context
// is cancelled or an error occurs.
func (s *Server) ListenAndServe(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.Handle("GET /health", handleHealth(s.Version))
	mux.Handle("POST /ask", handleAsk())
	mux.Handle("OPTIONS /ask", handleHealth(s.Version)) // CORS preflight

	handler := corsMiddleware(mux)

	srv := &http.Server{
		Addr:    s.Addr,
		Handler: handler,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}

	// Shut down gracefully when context is cancelled.
	go func() {
		<-ctx.Done()
		srv.Close()
	}()

	fmt.Fprintf(os.Stderr, "Q&A server listening on http://localhost%s\n", s.Addr)
	fmt.Fprintf(os.Stderr, "Press Ctrl+C to stop\n")

	err := srv.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

// corsMiddleware adds CORS headers for localhost browser access.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
