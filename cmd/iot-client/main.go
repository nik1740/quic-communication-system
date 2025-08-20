package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/nik1740/quic-communication-system/internal/iot"
	"go.uber.org/zap"
)

func main() {
	var (
		serverAddr = flag.String("server", "localhost:4433", "Server address")
		deviceID   = flag.String("device-id", "", "Device ID (random if not specified)")
		deviceType = flag.String("device-type", "temperature", "Device type (temperature, humidity, motion, camera, light)")
		interval   = flag.Duration("interval", 5*time.Second, "Data sending interval")
		duration   = flag.Duration("duration", 60*time.Second, "Test duration (0 for infinite)")
	)
	flag.Parse()

	// Initialize logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Generate device ID if not provided
	if *deviceID == "" {
		*deviceID = fmt.Sprintf("device-%d", rand.Intn(10000))
	}

	logger.Info("Starting IoT device simulator",
		zap.String("device_id", *deviceID),
		zap.String("device_type", *deviceType),
		zap.String("server", *serverAddr),
		zap.Duration("interval", *interval),
	)

	// Create TLS config (insecure for testing)
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic-communication-system"},
	}

	// Connect to server
	ctx := context.Background()
	if *duration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *duration)
		defer cancel()
	}

	conn, err := quic.DialAddr(ctx, *serverAddr, tlsConfig, &quic.Config{})
	if err != nil {
		logger.Fatal("Failed to connect to server", zap.Error(err))
	}
	defer conn.CloseWithError(0, "")

	logger.Info("Connected to server")

	// Open stream for IoT communication
	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		logger.Fatal("Failed to open stream", zap.Error(err))
	}
	defer stream.Close()

	// Send protocol header
	if _, err := stream.Write([]byte(iot.ProtocolIoT)); err != nil {
		logger.Fatal("Failed to send protocol header", zap.Error(err))
	}

	encoder := json.NewEncoder(stream)
	decoder := json.NewDecoder(stream)

	// Send initial heartbeat
	if err := sendHeartbeat(encoder, *deviceID, logger); err != nil {
		logger.Fatal("Failed to send heartbeat", zap.Error(err))
	}

	// Start response handler
	go handleResponses(decoder, logger)

	// Start sending data
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	deviceTypeEnum := iot.DeviceType(*deviceType)
	sequenceNum := 0

	for {
		select {
		case <-ctx.Done():
			logger.Info("Simulation finished")
			return
		case <-ticker.C:
			if err := sendSensorData(encoder, *deviceID, deviceTypeEnum, sequenceNum, logger); err != nil {
				logger.Error("Failed to send sensor data", zap.Error(err))
				return
			}
			sequenceNum++

			// Send heartbeat every 10 intervals
			if sequenceNum%10 == 0 {
				if err := sendHeartbeat(encoder, *deviceID, logger); err != nil {
					logger.Error("Failed to send heartbeat", zap.Error(err))
					return
				}
			}
		}
	}
}

// sendSensorData sends simulated sensor data
func sendSensorData(encoder *json.Encoder, deviceID string, deviceType iot.DeviceType, seq int, logger *zap.Logger) error {
	var value interface{}
	var unit string

	// Generate realistic sensor data based on device type
	switch deviceType {
	case iot.DeviceTemperature:
		value = 20.0 + rand.Float64()*15.0 // 20-35°C
		unit = "°C"
	case iot.DeviceHumidity:
		value = 30.0 + rand.Float64()*40.0 // 30-70%
		unit = "%"
	case iot.DeviceMotion:
		value = rand.Intn(2) == 1 // boolean motion detection
		unit = "detected"
	case iot.DeviceLight:
		value = rand.Float64() * 1000.0 // 0-1000 lux
		unit = "lux"
	default:
		value = rand.Float64() * 100.0
		unit = "unknown"
	}

	sensorData := iot.SensorData{
		DeviceID:  deviceID,
		Type:      deviceType,
		Value:     value,
		Unit:      unit,
		Timestamp: time.Now(),
		Quality:   0.8 + rand.Float64()*0.2, // 0.8-1.0 quality
	}

	msg := iot.Message{
		Type:      iot.MessageSensorData,
		DeviceID:  deviceID,
		Timestamp: time.Now(),
		Data:      sensorData,
		Reliable:  false, // Sensor data can be unreliable
	}

	if err := encoder.Encode(msg); err != nil {
		return err
	}

	logger.Debug("Sent sensor data",
		zap.String("device_id", deviceID),
		zap.String("type", string(deviceType)),
		zap.Any("value", value),
		zap.String("unit", unit),
	)

	return nil
}

// sendHeartbeat sends a heartbeat message
func sendHeartbeat(encoder *json.Encoder, deviceID string, logger *zap.Logger) error {
	msg := iot.Message{
		Type:      iot.MessageHeartbeat,
		DeviceID:  deviceID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"status": "alive",
			"uptime": time.Now().Unix(),
		},
		Reliable: true, // Heartbeats should be reliable
	}

	if err := encoder.Encode(msg); err != nil {
		return err
	}

	logger.Debug("Sent heartbeat", zap.String("device_id", deviceID))
	return nil
}

// handleResponses handles responses from the server
func handleResponses(decoder *json.Decoder, logger *zap.Logger) {
	for {
		var response iot.Message
		if err := decoder.Decode(&response); err != nil {
			logger.Error("Failed to decode response", zap.Error(err))
			return
		}

		logger.Debug("Received response",
			zap.String("type", string(response.Type)),
			zap.String("device_id", response.DeviceID),
			zap.Any("data", response.Data),
		)

		// Handle different response types
		switch response.Type {
		case iot.MessageResponse:
			if data, ok := response.Data.(map[string]interface{}); ok {
				if status, exists := data["status"]; exists {
					logger.Info("Server response",
						zap.String("device_id", response.DeviceID),
						zap.Any("status", status),
					)
				}
			}
		case iot.MessageCommand:
			logger.Info("Received command",
				zap.String("device_id", response.DeviceID),
				zap.Any("command", response.Data),
			)
			// In a real implementation, we would execute the command here
		}
	}
}