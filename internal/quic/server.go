package quic

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"

	"github.com/quic-go/quic-go/http3"
	"github.com/sirupsen/logrus"
)

// Server represents a QUIC server
type Server struct {
	addr     string
	certFile string
	keyFile  string
	logger   *logrus.Logger
	server   *http3.Server
	mux      *http.ServeMux
	mu       sync.RWMutex
	handlers map[string]http.HandlerFunc
}

// NewServer creates a new QUIC server
func NewServer(addr, certFile, keyFile string, logger *logrus.Logger) *Server {
	return &Server{
		addr:     addr,
		certFile: certFile,
		keyFile:  keyFile,
		logger:   logger,
		mux:      http.NewServeMux(),
		handlers: make(map[string]http.HandlerFunc),
	}
}

// RegisterHandler registers a new HTTP handler
func (s *Server) RegisterHandler(pattern string, handler http.HandlerFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.handlers[pattern] = handler
	s.mux.HandleFunc(pattern, handler)
	s.logger.WithFields(logrus.Fields{
		"pattern": pattern,
		"component": "quic-server",
	}).Info("Registered handler")
}

// Start starts the QUIC server
func (s *Server) Start(ctx context.Context) error {
	s.logger.WithFields(logrus.Fields{
		"addr": s.addr,
		"component": "quic-server",
	}).Info("Starting QUIC server")

	// Load TLS certificate
	cert, err := tls.LoadX509KeyPair(s.certFile, s.keyFile)
	if err != nil {
		return fmt.Errorf("failed to load TLS certificate: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"h3", "h3-29", "h3-28", "h3-27"},
	}

	// Create HTTP/3 server
	s.server = &http3.Server{
		Addr:      s.addr,
		Handler:   s.mux,
		TLSConfig: tlsConfig,
	}

	// Start server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- s.server.ListenAndServe()
	}()

	s.logger.WithFields(logrus.Fields{
		"addr": s.addr,
		"component": "quic-server",
	}).Info("QUIC server started")

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

// Stop stops the QUIC server
func (s *Server) Stop() error {
	s.logger.WithField("component", "quic-server").Info("Stopping QUIC server")
	
	if s.server != nil {
		if err := s.server.Close(); err != nil {
			return fmt.Errorf("failed to stop HTTP/3 server: %w", err)
		}
	}

	s.logger.WithField("component", "quic-server").Info("QUIC server stopped")
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