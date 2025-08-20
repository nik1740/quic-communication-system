package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/nik1740/quic-communication-system/internal/iot"
	"github.com/nik1740/quic-communication-system/internal/quic"
	"github.com/nik1740/quic-communication-system/internal/streaming"
	"github.com/nik1740/quic-communication-system/pkg/config"
	"github.com/nik1740/quic-communication-system/pkg/logging"
	"github.com/spf13/cobra"
)

var (
	configPath string
	certDir    string
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "server",
		Short: "QUIC Communication System Server",
		Long:  "A QUIC-based communication server for IoT devices and video streaming",
		Run:   runServer,
	}

	rootCmd.Flags().StringVarP(&configPath, "config", "c", ".", "Configuration file path")
	rootCmd.Flags().StringVarP(&certDir, "certs", "d", "./certs", "Certificate directory")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runServer(cmd *cobra.Command, args []string) {
	// Load configuration
	config, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger, err := logging.NewLogger(config.Logging.Level, config.Logging.Format, config.Logging.File)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	logger.WithComponent("main").Info("Starting QUIC Communication System Server")

	// Ensure certificate directory exists
	if err := os.MkdirAll(certDir, 0755); err != nil {
		logger.WithComponent("main").WithError(err).Fatal("Failed to create certificate directory")
	}

	// Generate self-signed certificates if they don't exist
	certFile := filepath.Join(certDir, "server.crt")
	keyFile := filepath.Join(certDir, "server.key")
	
	if err := generateCertificatesIfNeeded(certFile, keyFile, logger); err != nil {
		logger.WithComponent("main").WithError(err).Fatal("Failed to generate certificates")
	}

	// Update config with actual cert paths
	config.Server.CertFile = certFile
	config.Server.KeyFile = keyFile

	// Initialize IoT manager
	iotManager := iot.NewManager(logger.Logger, 10000)
	
	// Register demo IoT devices
	registerDemoDevices(iotManager, logger)

	// Initialize handlers
	iotHandler := iot.NewHandler(iotManager, logger.Logger)
	streamingHandler := streaming.NewHandler(config.Streaming.VideoDir, logger.Logger)

	// Create QUIC server
	server := quic.NewServer(config.Server.QUICAddr, config.Server.CertFile, config.Server.KeyFile, logger.Logger)

	// Register health check
	server.RegisterHandler("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"healthy","timestamp":"%s","server":"quic","component":"main"}`, time.Now().Format(time.RFC3339))
	})

	// Register IoT routes
	iotHandler.RegisterRoutes(server.RegisterHandler)

	// Register streaming routes
	streamingHandler.RegisterRoutes(server.RegisterHandler)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-signalChan
		logger.WithComponent("main").Info("Received shutdown signal")
		cancel()
	}()

	// Start server
	logger.WithComponent("main").WithField("addr", config.Server.QUICAddr).Info("Starting QUIC server")
	
	if err := server.Start(ctx); err != nil {
		logger.WithComponent("main").WithError(err).Error("Server error")
	}

	logger.WithComponent("main").Info("Server stopped")
}

func generateCertificatesIfNeeded(certFile, keyFile string, logger *logging.Logger) error {
	// Check if certificates already exist
	if _, err := os.Stat(certFile); err == nil {
		if _, err := os.Stat(keyFile); err == nil {
			logger.WithComponent("main").Info("Using existing certificates")
			return nil
		}
	}

	logger.WithComponent("main").Info("Generating self-signed certificates")

	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"QUIC Communication System"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{""},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour), // 1 year
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
		DNSNames:     []string{"localhost"},
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	// Save certificate
	certOut, err := os.Create(certFile)
	if err != nil {
		return fmt.Errorf("failed to create cert file: %w", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return fmt.Errorf("failed to write certificate: %w", err)
	}

	// Save private key
	keyOut, err := os.Create(keyFile)
	if err != nil {
		return fmt.Errorf("failed to create key file: %w", err)
	}
	defer keyOut.Close()

	privateKeyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}

	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privateKeyDER}); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	logger.WithComponent("main").Info("Self-signed certificates generated successfully")
	return nil
}

func registerDemoDevices(manager *iot.Manager, logger *logging.Logger) {
	demoDevices := []*iot.Device{
		{
			ID:       "temp-sensor-001",
			Name:     "Temperature Sensor 1",
			Type:     iot.TemperatureSensor,
			Location: "room-1",
		},
		{
			ID:       "humidity-sensor-001",
			Name:     "Humidity Sensor 1",
			Type:     iot.HumiditySensor,
			Location: "room-1",
		},
		{
			ID:       "motion-sensor-001",
			Name:     "Motion Sensor 1",
			Type:     iot.MotionSensor,
			Location: "hallway",
		},
		{
			ID:       "light-sensor-001",
			Name:     "Light Sensor 1",
			Type:     iot.LightSensor,
			Location: "window",
		},
	}

	for _, device := range demoDevices {
		manager.RegisterDevice(device)
	}

	logger.WithComponent("main").WithField("count", len(demoDevices)).Info("Registered demo IoT devices")
}