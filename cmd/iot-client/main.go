package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/nik1740/quic-communication-system/internal/iot"
	"github.com/quic-go/quic-go"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	logger = logrus.New()
	rootCmd = &cobra.Command{
		Use:   "iot-client",
		Short: "IoT Device Simulator",
		Long:  "Simulates IoT devices connecting to the QUIC communication system",
		Run:   runClient,
	}
)

func init() {
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	rootCmd.Flags().StringP("server", "s", "localhost:4433", "Server address")
	rootCmd.Flags().StringP("device-id", "i", "", "Device ID (auto-generated if not provided)")
	rootCmd.Flags().StringP("device-type", "t", "temperature", "Device type (temperature, humidity, motion, pressure)")
	rootCmd.Flags().StringP("location", "l", "office", "Device location")
	rootCmd.Flags().IntP("interval", "n", 5, "Data reporting interval in seconds")
	rootCmd.Flags().IntP("count", "c", 0, "Number of data points to send (0 = infinite)")
	rootCmd.Flags().BoolP("debug", "d", false, "Enable debug logging")

	viper.BindPFlags(rootCmd.Flags())
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logger.WithError(err).Fatal("Failed to execute command")
		os.Exit(1)
	}
}

func runClient(cmd *cobra.Command, args []string) {
	if viper.GetBool("debug") {
		logger.SetLevel(logrus.DebugLevel)
	}

	serverAddr := viper.GetString("server")
	deviceID := viper.GetString("device-id")
	if deviceID == "" {
		deviceID = fmt.Sprintf("device_%d", time.Now().UnixNano())
	}

	deviceType := iot.DeviceType(viper.GetString("device-type"))
	location := viper.GetString("location")
	interval := time.Duration(viper.GetInt("interval")) * time.Second
	count := viper.GetInt("count")

	logger.Infof("Starting IoT client simulation")
	logger.Infof("Device ID: %s", deviceID)
	logger.Infof("Device Type: %s", deviceType)
	logger.Infof("Location: %s", location)
	logger.Infof("Server: %s", serverAddr)
	logger.Infof("Reporting Interval: %v", interval)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create QUIC connection
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // For testing only
		NextProtos:         []string{"quic-iot"},
	}

	config := &quic.Config{
		MaxIdleTimeout: 30 * time.Second,
		KeepAlivePeriod: 10 * time.Second,
	}

	conn, err := quic.DialAddr(ctx, serverAddr, tlsConfig, config)
	if err != nil {
		logger.WithError(err).Fatal("Failed to connect to server")
	}
	defer conn.CloseWithError(0, "")

	logger.Info("Connected to QUIC server")

	// Register device
	if err := registerDevice(ctx, conn, deviceID, deviceType, location); err != nil {
		logger.WithError(err).Fatal("Failed to register device")
	}

	var wg sync.WaitGroup

	// Start sensor data sender
	wg.Add(1)
	go func() {
		defer wg.Done()
		sendSensorData(ctx, conn, deviceID, deviceType, interval, count)
	}()

	// Start command handler
	wg.Add(1)
	go func() {
		defer wg.Done()
		handleCommands(ctx, conn, deviceID)
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("IoT client started. Press Ctrl+C to stop.")

	<-sigChan
	logger.Info("Shutdown signal received, stopping client...")

	cancel()
	wg.Wait()

	logger.Info("IoT client stopped")
}

// registerDevice registers the device with the server
func registerDevice(ctx context.Context, conn *quic.Conn, deviceID string, deviceType iot.DeviceType, location string) error {
	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		return fmt.Errorf("failed to open registration stream: %w", err)
	}
	defer stream.Close()

	// Send protocol identifier
	if _, err := stream.Write([]byte("iot")); err != nil {
		return fmt.Errorf("failed to write protocol identifier: %w", err)
	}

	// Send message type
	msgType := map[string]string{"type": "device_registration"}
	if err := json.NewEncoder(stream).Encode(msgType); err != nil {
		return fmt.Errorf("failed to encode message type: %w", err)
	}

	// Send device registration
	device := iot.Device{
		ID:       deviceID,
		Type:     deviceType,
		Location: location,
		Status:   "online",
	}

	if err := json.NewEncoder(stream).Encode(device); err != nil {
		return fmt.Errorf("failed to encode device registration: %w", err)
	}

	// Read response
	var response map[string]string
	if err := json.NewDecoder(stream).Decode(&response); err != nil {
		return fmt.Errorf("failed to decode registration response: %w", err)
	}

	if response["status"] == "registered" {
		logger.Info("Device registered successfully")
		return nil
	}

	return fmt.Errorf("registration failed: %s", response["message"])
}

// sendSensorData sends periodic sensor data to the server
func sendSensorData(ctx context.Context, conn *quic.Conn, deviceID string, deviceType iot.DeviceType, interval time.Duration, count int) {
	logger.Info("Starting sensor data transmission")

	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		logger.WithError(err).Error("Failed to open sensor data stream")
		return
	}
	defer stream.Close()

	// Send protocol identifier
	if _, err := stream.Write([]byte("iot")); err != nil {
		logger.WithError(err).Error("Failed to write protocol identifier")
		return
	}

	// Send message type
	msgType := map[string]string{"type": "sensor_data"}
	if err := json.NewEncoder(stream).Encode(msgType); err != nil {
		logger.WithError(err).Error("Failed to encode message type")
		return
	}

	encoder := json.NewEncoder(stream)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	sentCount := 0

	for {
		select {
		case <-ctx.Done():
			logger.Info("Stopping sensor data transmission")
			return

		case <-ticker.C:
			data := generateSensorData(deviceID, deviceType)
			
			if err := encoder.Encode(data); err != nil {
				logger.WithError(err).Error("Failed to send sensor data")
				return
			}

			sentCount++
			logger.Infof("Sent sensor data #%d: %f %s", sentCount, data.Value, data.Unit)

			if count > 0 && sentCount >= count {
				logger.Infof("Sent %d data points, stopping", sentCount)
				return
			}
		}
	}
}

// handleCommands handles incoming commands from the server
func handleCommands(ctx context.Context, conn *quic.Conn, deviceID string) {
	logger.Info("Starting command handler")

	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		logger.WithError(err).Error("Failed to open command stream")
		return
	}
	defer stream.Close()

	// Send protocol identifier
	if _, err := stream.Write([]byte("iot")); err != nil {
		logger.WithError(err).Error("Failed to write protocol identifier")
		return
	}

	// Send message type
	msgType := map[string]string{"type": "control_command"}
	if err := json.NewEncoder(stream).Encode(msgType); err != nil {
		logger.WithError(err).Error("Failed to encode message type")
		return
	}

	decoder := json.NewDecoder(stream)
	encoder := json.NewEncoder(stream)

	for {
		select {
		case <-ctx.Done():
			logger.Info("Stopping command handler")
			return

		default:
			var cmd iot.ControlCommand
			if err := decoder.Decode(&cmd); err != nil {
				// No command available, continue
				time.Sleep(100 * time.Millisecond)
				continue
			}

			logger.Infof("Received command: %s", cmd.Command)

			// Process command and send response
			response := iot.ControlResponse{
				DeviceID:  deviceID,
				Command:   cmd.Command,
				Status:    "success",
				Message:   "Command executed successfully",
				Timestamp: time.Now(),
			}

			if err := encoder.Encode(response); err != nil {
				logger.WithError(err).Error("Failed to send command response")
				return
			}

			logger.Infof("Sent response for command: %s", cmd.Command)
		}
	}
}

// generateSensorData generates realistic sensor data based on device type
func generateSensorData(deviceID string, deviceType iot.DeviceType) iot.SensorData {
	data := iot.SensorData{
		DeviceID:  deviceID,
		Type:      deviceType,
		Timestamp: time.Now(),
		Quality:   rand.Intn(20) + 80, // 80-100% quality
	}

	switch deviceType {
	case iot.DeviceTypeTemperature:
		// Generate temperature between 18-30°C with some variation
		baseTemp := 20.0 + rand.Float64()*10.0
		variation := (rand.Float64() - 0.5) * 2.0
		data.Value = baseTemp + variation
		data.Unit = "°C"

	case iot.DeviceTypeHumidity:
		// Generate humidity between 30-80%
		data.Value = 30.0 + rand.Float64()*50.0
		data.Unit = "%"

	case iot.DeviceTypeMotion:
		// Motion detection (0 or 1)
		if rand.Float64() < 0.1 { // 10% chance of motion
			data.Value = 1.0
		} else {
			data.Value = 0.0
		}
		data.Unit = "bool"

	case iot.DeviceTypePressure:
		// Generate pressure around 1013 hPa (standard atmospheric pressure)
		basePressure := 1013.25
		variation := (rand.Float64() - 0.5) * 20.0
		data.Value = basePressure + variation
		data.Unit = "hPa"

	default:
		// Default random value
		data.Value = rand.Float64() * 100.0
		data.Unit = "units"
	}

	return data
}