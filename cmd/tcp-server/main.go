package main

import (
	"context"
	"crypto/tls"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	quiclib "github.com/nik1740/quic-communication-system/internal/quic"
	"github.com/nik1740/quic-communication-system/internal/tcp"
)

func main() {
	var (
		addr     = flag.String("addr", ":8080", "Server address")
		protocol = flag.String("protocol", "tcp", "Protocol (tcp or quic)")
		certFile = flag.String("cert", "", "TLS certificate file")
		keyFile  = flag.String("key", "", "TLS key file")
	)
	flag.Parse()

	log.Printf("Starting %s server on %s", *protocol, *addr)

	// Generate TLS certificate if not provided
	var tlsConfig *tls.Config
	if *certFile == "" || *keyFile == "" {
		cert, err := quiclib.GenerateSelfSignedCert()
		if err != nil {
			log.Fatal("Failed to generate certificate:", err)
		}
		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
	} else {
		cert, err := tls.LoadX509KeyPair(*certFile, *keyFile)
		if err != nil {
			log.Fatal("Failed to load certificate:", err)
		}
		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
	}

	// Create and start server
	server := tcp.NewServer(*addr, tlsConfig)

	// Start server in a goroutine
	go func() {
		if err := server.Start(); err != nil {
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

	if err := server.Stop(); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
	
	// Wait for context timeout
	<-ctx.Done()
}