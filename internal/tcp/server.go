package tcp

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// Server represents a TCP/TLS server for comparison
type Server struct {
	addr     string
	certFile string
	keyFile  string
	logger   *logrus.Logger
	server   *http.Server
	router   *mux.Router
	mu       sync.RWMutex
	handlers map[string]http.HandlerFunc
}

// NewServer creates a new TCP/TLS server
func NewServer(addr, certFile, keyFile string, logger *logrus.Logger) *Server {
	router := mux.NewRouter()
	
	return &Server{
		addr:     addr,
		certFile: certFile,
		keyFile:  keyFile,
		logger:   logger,
		router:   router,
		handlers: make(map[string]http.HandlerFunc),
		server: &http.Server{
			Addr:         addr,
			Handler:      router,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}
}

// RegisterHandler registers a new HTTP handler
func (s *Server) RegisterHandler(pattern string, handler http.HandlerFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.handlers[pattern] = handler
	s.router.HandleFunc(pattern, handler)
	s.logger.WithFields(logrus.Fields{
		"pattern":   pattern,
		"component": "tcp-server",
	}).Info("Registered handler")
}

// Start starts the TCP/TLS server
func (s *Server) Start(ctx context.Context) error {
	s.logger.WithFields(logrus.Fields{
		"addr":      s.addr,
		"component": "tcp-server",
	}).Info("Starting TCP/TLS server")

	// Start server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		if s.certFile != "" && s.keyFile != "" {
			errChan <- s.server.ListenAndServeTLS(s.certFile, s.keyFile)
		} else {
			errChan <- s.server.ListenAndServe()
		}
	}()

	s.logger.WithFields(logrus.Fields{
		"addr":      s.addr,
		"component": "tcp-server",
	}).Info("TCP/TLS server started")

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		return s.Stop()
	case err := <-errChan:
		if err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}
		return nil
	}
}

// Stop stops the TCP/TLS server
func (s *Server) Stop() error {
	s.logger.WithField("component", "tcp-server").Info("Stopping TCP/TLS server")
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to stop server: %w", err)
	}

	s.logger.WithField("component", "tcp-server").Info("TCP/TLS server stopped")
	return nil
}

// GetHandlers returns registered handlers
func (s *Server) GetHandlers() map[string]http.HandlerFunc {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	handlers := make(map[string]http.HandlerFunc)
	for k, v := range s.handlers {
		handlers[k] = v
	}
	return handlers
}