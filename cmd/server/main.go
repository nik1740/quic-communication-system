package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/nik1740/quic-communication-system/internal/iot"
	"github.com/nik1740/quic-communication-system/internal/quic"
	"github.com/nik1740/quic-communication-system/internal/streaming"
	"github.com/nik1740/quic-communication-system/pkg/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"
)

func main() {
	var (
		configFile = flag.String("config", "", "Path to configuration file")
		addr       = flag.String("addr", "localhost:4433", "Server address")
		logLevel   = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	)
	flag.Parse()

	// Initialize logger
	logger, err := initLogger(*logLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Load configuration
	config := loadConfig(*configFile, logger)

	// Override address if provided via flag
	if *addr != "localhost:4433" {
		config.Server.Host = "localhost"
		config.Server.Port = 4433
		if host, port, err := net.SplitHostPort(*addr); err == nil {
			config.Server.Host = host
			if p, err := strconv.Atoi(port); err == nil {
				config.Server.Port = p
			}
		}
	}

	serverAddr := fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port)

	// Create QUIC server
	server, err := quic.NewServer(serverAddr, logger)
	if err != nil {
		logger.Fatal("Failed to create QUIC server", zap.Error(err))
	}

	// Create and register handlers
	iotHandler := iot.NewHandler(logger.Named("iot"))
	streamingHandler := streaming.NewHandler(logger.Named("streaming"))

	server.RegisterHandler(iot.ProtocolIoT, iotHandler)
	server.RegisterHandler(streaming.ProtocolStreaming, streamingHandler)

	// Start IoT device monitor
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go iotHandler.StartDeviceMonitor(ctx, config.IoT.HeartbeatTimeout)

	// Start data processing goroutines
	go processIoTData(ctx, iotHandler, logger.Named("iot-processor"))
	go monitorStreaming(ctx, streamingHandler, logger.Named("streaming-monitor"))

	// Start server
	go func() {
		logger.Info("Starting QUIC communication server", 
			zap.String("address", serverAddr),
			zap.String("version", "1.0.0"),
		)
		
		if err := server.Start(ctx); err != nil && err != context.Canceled {
			logger.Fatal("Server error", zap.Error(err))
		}
	}()

	// Wait for shutdown signal
	waitForShutdown(logger)

	// Graceful shutdown
	logger.Info("Shutting down server...")
	cancel()
	
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	done := make(chan error, 1)
	go func() {
		done <- server.Stop()
	}()

	select {
	case err := <-done:
		if err != nil {
			logger.Error("Error during shutdown", zap.Error(err))
		} else {
			logger.Info("Server stopped gracefully")
		}
	case <-shutdownCtx.Done():
		logger.Warn("Shutdown timeout reached")
	}
}

// initLogger initializes the zap logger
func initLogger(level string) (*zap.Logger, error) {
	var zapLevel zap.AtomicLevel
	switch level {
	case "debug":
		zapLevel = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		zapLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		zapLevel = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		zapLevel = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		zapLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	config := zap.NewProductionConfig()
	config.Level = zapLevel
	config.Development = false
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	return config.Build()
}

// loadConfig loads configuration from file or returns default
func loadConfig(configFile string, logger *zap.Logger) *config.Config {
	if configFile == "" {
		logger.Info("No config file specified, using defaults")
		return config.DefaultConfig()
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		logger.Warn("Failed to read config file, using defaults", 
			zap.String("file", configFile),
			zap.Error(err),
		)
		return config.DefaultConfig()
	}

	var cfg config.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		logger.Warn("Failed to parse config file, using defaults",
			zap.String("file", configFile),
			zap.Error(err),
		)
		return config.DefaultConfig()
	}

	logger.Info("Loaded configuration from file", zap.String("file", configFile))
	return &cfg
}

// processIoTData processes incoming IoT sensor data
func processIoTData(ctx context.Context, handler *iot.Handler, logger *zap.Logger) {
	sensorDataChan := handler.GetSensorDataChannel()
	commandChan := handler.GetCommandChannel()

	for {
		select {
		case <-ctx.Done():
			return
		case sensorData := <-sensorDataChan:
			// Process sensor data (in real implementation, this might store to database, 
			// trigger alerts, or forward to analytics systems)
			logger.Debug("Processing sensor data",
				zap.String("device_id", sensorData.DeviceID),
				zap.String("type", string(sensorData.Type)),
				zap.Any("value", sensorData.Value),
				zap.Float64("quality", sensorData.Quality),
			)

			// Example: Trigger alert for high temperature
			if sensorData.Type == iot.DeviceTemperature {
				if temp, ok := sensorData.Value.(float64); ok && temp > 30.0 {
					logger.Warn("High temperature alert",
						zap.String("device_id", sensorData.DeviceID),
						zap.Float64("temperature", temp),
					)
				}
			}

		case command := <-commandChan:
			// Process commands (in real implementation, this might execute device actions,
			// log commands, or update device states)
			logger.Info("Processing device command",
				zap.String("device_id", command.DeviceID),
				zap.String("type", command.Type),
				zap.Int("priority", command.Priority),
			)
		}
	}
}

// monitorStreaming monitors active streaming sessions
func monitorStreaming(ctx context.Context, handler *streaming.Handler, logger *zap.Logger) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			streams := handler.GetActiveStreams()
			if len(streams) > 0 {
				logger.Info("Active streaming sessions",
					zap.Int("count", len(streams)),
				)
				
				for streamID, stats := range streams {
					logger.Debug("Stream stats",
						zap.String("stream_id", streamID),
						zap.Uint64("bytes_sent", stats.BytesSent),
						zap.Uint64("chunks_sent", stats.ChunksSent),
						zap.Float64("avg_bitrate_kbps", stats.AverageBitrate),
						zap.Float64("buffer_health", stats.BufferHealth),
						zap.Float64("packet_loss", stats.PacketLoss),
					)
				}
			}
		}
	}
}

// waitForShutdown waits for shutdown signals
func waitForShutdown(logger *zap.Logger) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	sig := <-signals
	logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
}