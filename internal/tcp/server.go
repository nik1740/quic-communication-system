package tcp

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/nik1740/quic-communication-system/internal/iot"
	"github.com/nik1740/quic-communication-system/internal/streaming"
)

// Server represents a TCP/TLS server for comparison
type Server struct {
	server   *http.Server
	tlsConfig *tls.Config
}

// NewServer creates a new TCP/TLS server
func NewServer(addr string, tlsConfig *tls.Config) *Server {
	mux := http.NewServeMux()
	
	// IoT endpoints (same as QUIC)
	mux.HandleFunc("/iot/", iot.Handler)
	
	// Video streaming endpoints (same as QUIC)
	mux.HandleFunc("/stream/", streaming.Handler)
	
	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "TCP/TLS server is running")
	})

	// Benchmark endpoint
	mux.HandleFunc("/benchmark/", handleBenchmark)

	return &Server{
		server: &http.Server{
			Addr:         addr,
			Handler:      mux,
			TLSConfig:    tlsConfig,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		tlsConfig: tlsConfig,
	}
}

// Start starts the TCP/TLS server
func (s *Server) Start() error {
	log.Printf("Starting TCP/TLS server on %s", s.server.Addr)
	if s.tlsConfig != nil {
		return s.server.ListenAndServeTLS("", "")
	}
	return s.server.ListenAndServe()
}

// Stop stops the TCP/TLS server
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}

func handleBenchmark(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Return connection info for benchmarking
		info := map[string]interface{}{
			"protocol":   "TCP/TLS",
			"connection": "HTTP/1.1 or HTTP/2",
			"timestamp":  time.Now().Unix(),
			"server":     "tcp-comparison",
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Protocol", "TCP")
		
		if err := writeJSON(w, info); err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
		}
		
	case http.MethodPost:
		// Echo test for latency measurement
		start := time.Now()
		
		// Read and echo the request body
		body := make([]byte, r.ContentLength)
		n, err := r.Body.Read(body)
		if err != nil && err.Error() != "EOF" {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}
		
		latency := time.Since(start)
		
		response := map[string]interface{}{
			"protocol":     "TCP/TLS",
			"bytes_read":   n,
			"latency_ns":   latency.Nanoseconds(),
			"latency_ms":   float64(latency.Nanoseconds()) / 1e6,
			"timestamp":    time.Now().Unix(),
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Protocol", "TCP")
		w.Header().Set("X-Latency-Ms", fmt.Sprintf("%.2f", response["latency_ms"]))
		
		if err := writeJSON(w, response); err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
		}
		
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func writeJSON(w http.ResponseWriter, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(data)
}