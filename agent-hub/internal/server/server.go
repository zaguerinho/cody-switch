package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/zaguerinho/claude-switch/agent-hub/internal/store"
)

// Server is the agent-hub HTTP API server.
type Server struct {
	store   *store.FileStore
	apiPort int
	uiPort  int
	version string
}

// New creates a new server backed by the given store.
func New(s *store.FileStore, apiPort, uiPort int, version string) *Server {
	return &Server{store: s, apiPort: apiPort, uiPort: uiPort, version: version}
}

// Run starts the API and dashboard servers, blocking until ctx is cancelled.
func (s *Server) Run(ctx context.Context) error {
	apiMux := s.apiRoutes()
	apiHandler := withRecovery(withLogging(apiMux))

	apiServer := &http.Server{
		Addr:              fmt.Sprintf("127.0.0.1:%d", s.apiPort),
		Handler:           apiHandler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Start API server
	apiLn, err := net.Listen("tcp", apiServer.Addr)
	if err != nil {
		return fmt.Errorf("API listen: %w", err)
	}
	go func() {
		log.Printf("API server listening on http://127.0.0.1:%d", s.apiPort)
		if err := apiServer.Serve(apiLn); err != nil && err != http.ErrServerClosed {
			log.Printf("API server error: %v", err)
		}
	}()

	// Start dashboard server (shares same store, serves both API + UI)
	dashMux := s.dashboardRoutes()
	dashHandler := withRecovery(withLogging(dashMux))

	dashServer := &http.Server{
		Addr:              fmt.Sprintf("127.0.0.1:%d", s.uiPort),
		Handler:           dashHandler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	dashLn, err := net.Listen("tcp", dashServer.Addr)
	if err != nil {
		apiServer.Close()
		return fmt.Errorf("dashboard listen: %w", err)
	}
	go func() {
		log.Printf("Dashboard listening on http://127.0.0.1:%d", s.uiPort)
		if err := dashServer.Serve(dashLn); err != nil && err != http.ErrServerClosed {
			log.Printf("dashboard server error: %v", err)
		}
	}()

	<-ctx.Done()

	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	apiServer.Shutdown(shutCtx)
	dashServer.Shutdown(shutCtx)
	return nil
}
