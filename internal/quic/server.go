package quic

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/sirupsen/logrus"
)

// Server represents a QUIC server
type Server struct {
	listener  *quic.Listener
	tlsConfig *tls.Config
	logger    *logrus.Logger
	handlers  map[string]StreamHandler
}

// StreamHandler defines the interface for handling QUIC streams
type StreamHandler interface {
	HandleStream(ctx context.Context, stream *quic.Stream) error
}

// NewServer creates a new QUIC server
func NewServer(addr string, tlsConfig *tls.Config, logger *logrus.Logger) (*Server, error) {
	if tlsConfig == nil {
		tlsConfig = generateTLSConfig()
	}

	config := &quic.Config{
		MaxIdleTimeout:        30 * time.Second,
		MaxIncomingStreams:    100,
		MaxIncomingUniStreams: 100,
		KeepAlivePeriod:       10 * time.Second,
	}

	listener, err := quic.ListenAddr(addr, tlsConfig, config)
	if err != nil {
		return nil, fmt.Errorf("failed to start QUIC listener: %w", err)
	}

	return &Server{
		listener:  listener,
		tlsConfig: tlsConfig,
		logger:    logger,
		handlers:  make(map[string]StreamHandler),
	}, nil
}

// RegisterHandler registers a stream handler for a specific protocol
func (s *Server) RegisterHandler(protocol string, handler StreamHandler) {
	s.handlers[protocol] = handler
}

// Start starts the QUIC server
func (s *Server) Start(ctx context.Context) error {
	s.logger.Infof("QUIC server starting on %s", s.listener.Addr())

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			conn, err := s.listener.Accept(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				s.logger.WithError(err).Error("Failed to accept connection")
				continue
			}

			go s.handleConnection(ctx, conn)
		}
	}
}

// Stop stops the QUIC server
func (s *Server) Stop() error {
	return s.listener.Close()
}

// Addr returns the server's listening address
func (s *Server) Addr() net.Addr {
	return s.listener.Addr()
}

func (s *Server) handleConnection(ctx context.Context, conn *quic.Conn) {
	defer conn.CloseWithError(0, "")

	s.logger.Infof("New connection from %s", conn.RemoteAddr())

	for {
		select {
		case <-ctx.Done():
			return
		default:
			stream, err := conn.AcceptStream(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				s.logger.WithError(err).Error("Failed to accept stream")
				return
			}

			go s.handleStream(ctx, stream)
		}
	}
}

func (s *Server) handleStream(ctx context.Context, stream *quic.Stream) {
	defer stream.Close()

	// Read the protocol identifier (first 32 bytes)
	protocolBuf := make([]byte, 32)
	n, err := stream.Read(protocolBuf)
	if err != nil {
		s.logger.WithError(err).Error("Failed to read protocol identifier")
		return
	}

	protocol := string(protocolBuf[:n])
	s.logger.Infof("Handling stream for protocol: %s", protocol)

	handler, exists := s.handlers[protocol]
	if !exists {
		s.logger.Warnf("No handler registered for protocol: %s", protocol)
		return
	}

	if err := handler.HandleStream(ctx, stream); err != nil {
		s.logger.WithError(err).Errorf("Handler error for protocol %s", protocol)
	}
}

// generateTLSConfig creates a default TLS configuration for testing
func generateTLSConfig() *tls.Config {
	cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
	if err != nil {
		// Generate a self-signed certificate for testing
		cert = generateSelfSignedCert()
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"h3", "quic-iot", "quic-streaming"},
	}
}

// HTTP3Server creates an HTTP/3 server for video streaming
func (s *Server) CreateHTTP3Server(handler http.Handler) *http3.Server {
	return &http3.Server{
		Addr:      s.listener.Addr().String(),
		TLSConfig: s.tlsConfig,
		Handler:   handler,
	}
}