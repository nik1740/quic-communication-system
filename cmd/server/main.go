package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nik1740/quic-communication-system/internal/iot"
	quiclib "github.com/nik1740/quic-communication-system/internal/quic"
	"github.com/nik1740/quic-communication-system/internal/streaming"
	"github.com/quic-go/quic-go/http3"
)

func main() {
	// Create TLS certificate for QUIC
	cert, err := quiclib.GenerateSelfSignedCert()
	if err != nil {
		log.Fatal("Failed to generate certificate:", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"h3"},
	}

	// Create HTTP/3 server
	server := &http3.Server{
		Addr:      ":8443",
		TLSConfig: tlsConfig,
	}

	// Set up HTTP handlers
	mux := http.NewServeMux()
	
	// IoT endpoints
	mux.HandleFunc("/iot/", iot.Handler)
	
	// Video streaming endpoints
	mux.HandleFunc("/stream/", streaming.Handler)
	
	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "QUIC server is running")
	})

	server.Handler = mux

	// Start server in a goroutine
	go func() {
		log.Printf("Starting QUIC server on :8443")
		if err := server.ListenAndServe(); err != nil {
			log.Fatal("Server failed:", err)
		}
	}()

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := server.Close(); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
	
	// Wait for context timeout
	<-ctx.Done()
}