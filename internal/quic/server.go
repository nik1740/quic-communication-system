package quic

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"time"

	"github.com/quic-go/quic-go"
	"go.uber.org/zap"
)

// Server represents a QUIC server instance
type Server struct {
	listener *quic.Listener
	logger   *zap.Logger
	handlers map[string]StreamHandler
}

// StreamHandler defines the interface for handling QUIC streams
type StreamHandler interface {
	HandleStream(ctx context.Context, stream quic.Stream) error
}

// NewServer creates a new QUIC server
func NewServer(addr string, logger *zap.Logger) (*Server, error) {
	tlsConfig, err := generateTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to generate TLS config: %w", err)
	}

	quicConfig := &quic.Config{
		MaxIdleTimeout:  30 * time.Second,
		KeepAlivePeriod: 10 * time.Second,
	}

	listener, err := quic.ListenAddr(addr, tlsConfig, quicConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create QUIC listener: %w", err)
	}

	return &Server{
		listener: listener,
		logger:   logger,
		handlers: make(map[string]StreamHandler),
	}, nil
}

// RegisterHandler registers a stream handler for a specific protocol
func (s *Server) RegisterHandler(protocol string, handler StreamHandler) {
	s.handlers[protocol] = handler
}

// Start starts the QUIC server
func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("Starting QUIC server", zap.String("addr", s.listener.Addr().String()))

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			conn, err := s.listener.Accept(ctx)
			if err != nil {
				s.logger.Error("Failed to accept connection", zap.Error(err))
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

// handleConnection handles a new QUIC connection
func (s *Server) handleConnection(ctx context.Context, conn quic.Connection) {
	defer conn.CloseWithError(0, "")

	s.logger.Info("New QUIC connection", 
		zap.String("remote_addr", conn.RemoteAddr().String()),
		zap.String("connection_id", conn.ConnectionState().TLS.ServerName),
	)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			stream, err := conn.AcceptStream(ctx)
			if err != nil {
				s.logger.Error("Failed to accept stream", zap.Error(err))
				return
			}

			go s.handleStream(ctx, stream)
		}
	}
}

// handleStream handles a new QUIC stream
func (s *Server) handleStream(ctx context.Context, stream quic.Stream) {
	defer stream.Close()

	// Read protocol header (first 4 bytes indicate protocol type)
	protocolBytes := make([]byte, 4)
	n, err := stream.Read(protocolBytes)
	if err != nil || n != 4 {
		s.logger.Error("Failed to read protocol header", zap.Error(err))
		return
	}

	protocol := string(protocolBytes)
	handler, exists := s.handlers[protocol]
	if !exists {
		s.logger.Warn("Unknown protocol", zap.String("protocol", protocol))
		return
	}

	if err := handler.HandleStream(ctx, stream); err != nil {
		s.logger.Error("Stream handler error", 
			zap.String("protocol", protocol),
			zap.Error(err),
		)
	}
}

// generateTLSConfig generates a self-signed certificate for testing
func generateTLSConfig() (*tls.Config, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"QUIC Communication System"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"San Francisco"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		DNSNames:     []string{"localhost"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"quic-communication-system"},
	}, nil
}