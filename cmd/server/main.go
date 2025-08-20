package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/nik1740/quic-communication-system/internal/iot"
	"github.com/nik1740/quic-communication-system/internal/quic"
	"github.com/nik1740/quic-communication-system/internal/streaming"
	"github.com/quic-go/quic-go/http3"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	logger = logrus.New()
	rootCmd = &cobra.Command{
		Use:   "quic-server",
		Short: "QUIC Communication System Server",
		Long:  "A comprehensive QUIC-based communication system for IoT devices and video streaming",
		Run:   runServer,
	}
)

func init() {
	// Configure logging
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	// Command line flags
	rootCmd.Flags().StringP("addr", "a", ":4433", "Server address")
	rootCmd.Flags().StringP("http-addr", "p", ":8080", "HTTP/3 server address")
	rootCmd.Flags().StringP("config", "c", "", "Configuration file path")
	rootCmd.Flags().BoolP("debug", "d", false, "Enable debug logging")

	// Bind flags to viper
	viper.BindPFlag("server.addr", rootCmd.Flags().Lookup("addr"))
	viper.BindPFlag("server.http_addr", rootCmd.Flags().Lookup("http-addr"))
	viper.BindPFlag("server.debug", rootCmd.Flags().Lookup("debug"))
	viper.BindPFlag("config", rootCmd.Flags().Lookup("config"))

	// Set defaults
	viper.SetDefault("server.addr", ":4433")
	viper.SetDefault("server.http_addr", ":8080")
	viper.SetDefault("server.debug", false)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logger.WithError(err).Fatal("Failed to execute command")
		os.Exit(1)
	}
}

func runServer(cmd *cobra.Command, args []string) {
	// Load configuration
	if configFile := viper.GetString("config"); configFile != "" {
		viper.SetConfigFile(configFile)
		if err := viper.ReadInConfig(); err != nil {
			logger.WithError(err).Warn("Failed to read config file")
		}
	}

	// Set log level
	if viper.GetBool("server.debug") {
		logger.SetLevel(logrus.DebugLevel)
		logger.Debug("Debug logging enabled")
	}

	addr := viper.GetString("server.addr")
	httpAddr := viper.GetString("server.http_addr")

	logger.Infof("Starting QUIC Communication System Server")
	logger.Infof("QUIC Server will listen on %s", addr)
	logger.Infof("HTTP/3 Server will listen on %s", httpAddr)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create QUIC server
	quicServer, err := quic.NewServer(addr, nil, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create QUIC server")
	}

	// Create handlers
	iotHandler := iot.NewHandler(logger)
	streamingHandler := streaming.NewHandler(logger)

	// Register protocol handlers
	quicServer.RegisterHandler("iot", iotHandler)
	quicServer.RegisterHandler("streaming", streamingHandler)

	// Create HTTP/3 server for video streaming
	http3Server := &http3.Server{
		Addr:    httpAddr,
		Handler: streamingHandler.SetupHTTP3Routes(),
	}

	var wg sync.WaitGroup

	// Start QUIC server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := quicServer.Start(ctx); err != nil && err != context.Canceled {
			logger.WithError(err).Error("QUIC server error")
		}
	}()

	// Start HTTP/3 server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := http3Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Error("HTTP/3 server error")
		}
	}()

	// Start IoT data processor
	wg.Add(1)
	go func() {
		defer wg.Done()
		processSensorData(ctx, iotHandler)
	}()

	// Start session cleanup routine
	wg.Add(1)
	go func() {
		defer wg.Done()
		cleanupSessions(ctx, streamingHandler)
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("Server started successfully. Press Ctrl+C to stop.")

	<-sigChan
	logger.Info("Shutdown signal received, stopping servers...")

	// Cancel context to stop all goroutines
	cancel()

	// Stop servers
	if err := quicServer.Stop(); err != nil {
		logger.WithError(err).Error("Error stopping QUIC server")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := http3Server.Shutdown(shutdownCtx); err != nil {
		logger.WithError(err).Error("Error stopping HTTP/3 server")
	}

	// Wait for all goroutines to finish
	wg.Wait()

	logger.Info("Server stopped gracefully")
}

// processSensorData processes incoming sensor data from IoT devices
func processSensorData(ctx context.Context, iotHandler *iot.Handler) {
	logger.Info("Starting sensor data processor")

	sensorChan := iotHandler.GetSensorDataChannel()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	dataCount := 0

	for {
		select {
		case <-ctx.Done():
			logger.Info("Stopping sensor data processor")
			return

		case data := <-sensorChan:
			dataCount++
			logger.Debugf("Processing sensor data from %s: %f %s (quality: %d)", 
				data.DeviceID, data.Value, data.Unit, data.Quality)

			// Here you would typically:
			// - Store data in database
			// - Apply filters and transformations
			// - Trigger alerts if thresholds are exceeded
			// - Send data to analytics systems

		case <-ticker.C:
			devices := iotHandler.GetDevices()
			logger.Infof("Sensor data processor status: %d data points processed, %d devices connected", 
				dataCount, len(devices))

			// Log device statuses
			for _, device := range devices {
				timeSinceLastSeen := time.Since(device.LastSeen)
				if timeSinceLastSeen > 30*time.Second {
					logger.Warnf("Device %s (%s) last seen %v ago", 
						device.ID, device.Type, timeSinceLastSeen)
				}
			}
		}
	}
}

// cleanupSessions periodically cleans up old streaming sessions
func cleanupSessions(ctx context.Context, streamingHandler *streaming.Handler) {
	logger.Info("Starting session cleanup routine")

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Stopping session cleanup routine")
			return

		case <-ticker.C:
			logger.Debug("Running session cleanup")
			streamingHandler.CleanupOldSessions(10 * time.Minute)
			
			activeCount := streamingHandler.GetActiveSessionsCount()
			logger.Infof("Session cleanup completed, %d active sessions", activeCount)
		}
	}
}

// version information
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

func init() {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("QUIC Communication System Server\n")
			fmt.Printf("Version: %s\n", Version)
			fmt.Printf("Git Commit: %s\n", GitCommit)
			fmt.Printf("Build Time: %s\n", BuildTime)
		},
	}
	rootCmd.AddCommand(versionCmd)
}