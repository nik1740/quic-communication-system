package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/nik1740/quic-communication-system/internal/iot"
	"github.com/nik1740/quic-communication-system/pkg/logging"
	"github.com/quic-go/quic-go/http3"
	"github.com/spf13/cobra"
)

var (
	serverAddr   string
	deviceID     string
	deviceType   string
	interval     int
	duration     int
	useQUIC      bool
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "iot-client",
		Short: "IoT Device Simulator",
		Long:  "Simulates IoT devices sending data to the QUIC communication server",
		Run:   runClient,
	}

	rootCmd.Flags().StringVarP(&serverAddr, "server", "s", "https://localhost:8443", "Server address")
	rootCmd.Flags().StringVarP(&deviceID, "device", "d", "", "Device ID (auto-generated if empty)")
	rootCmd.Flags().StringVarP(&deviceType, "type", "t", "temperature", "Device type (temperature, humidity, motion, light)")
	rootCmd.Flags().IntVarP(&interval, "interval", "i", 5, "Data sending interval in seconds")
	rootCmd.Flags().IntVarP(&duration, "duration", "D", 60, "Simulation duration in seconds (0 for infinite)")
	rootCmd.Flags().BoolVarP(&useQUIC, "quic", "q", true, "Use QUIC/HTTP3 protocol")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runClient(cmd *cobra.Command, args []string) {
	// Initialize logger
	logger, err := logging.NewLogger("info", "text", "")
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	logger.WithComponent("iot-client").Info("Starting IoT client simulator")

	// Generate device ID if not provided
	if deviceID == "" {
		deviceID = fmt.Sprintf("%s-device-%d", deviceType, rand.Intn(1000))
	}

	// Validate device type
	var sensorType iot.SensorType
	switch deviceType {
	case "temperature":
		sensorType = iot.TemperatureSensor
	case "humidity":
		sensorType = iot.HumiditySensor
	case "motion":
		sensorType = iot.MotionSensor
	case "light":
		sensorType = iot.LightSensor
	default:
		logger.WithComponent("iot-client").WithField("type", deviceType).Fatal("Invalid device type")
	}

	// Create HTTP client
	var client *http.Client
	if useQUIC {
		logger.WithComponent("iot-client").Info("Using QUIC/HTTP3 protocol")
		client = &http.Client{
			Transport: &http3.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, // For demo with self-signed certs
				},
			},
		}
	} else {
		logger.WithComponent("iot-client").Info("Using HTTP/1.1 protocol")
		client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	}

	// Register device
	if err := registerDevice(client, serverAddr, deviceID, deviceType, sensorType, logger); err != nil {
		logger.WithComponent("iot-client").WithError(err).Fatal("Failed to register device")
	}

	// Start sending sensor data
	logger.WithComponent("iot-client").WithFields(map[string]interface{}{
		"device_id": deviceID,
		"type":      deviceType,
		"interval":  interval,
		"duration":  duration,
	}).Info("Starting sensor data simulation")

	startTime := time.Now()
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check duration limit
			if duration > 0 && time.Since(startTime).Seconds() >= float64(duration) {
				logger.WithComponent("iot-client").Info("Simulation duration reached, stopping")
				return
			}

			// Generate and send sensor data
			data := iot.GenerateTestSensorData(deviceID, sensorType)
			if err := sendSensorData(client, serverAddr, data, logger); err != nil {
				logger.WithComponent("iot-client").WithError(err).Error("Failed to send sensor data")
			} else {
				logger.WithComponent("iot-client").WithFields(map[string]interface{}{
					"device_id": deviceID,
					"value":     data.Value,
					"unit":      data.Unit,
				}).Debug("Sent sensor data")
			}

			// Occasionally send a command to test command/response
			if rand.Intn(10) == 0 {
				if err := sendTestCommand(client, serverAddr, deviceID, logger); err != nil {
					logger.WithComponent("iot-client").WithError(err).Error("Failed to send test command")
				}
			}
		}
	}
}

func registerDevice(client *http.Client, serverAddr, deviceID, deviceType string, sensorType iot.SensorType, logger *logging.Logger) error {
	device := iot.Device{
		ID:       deviceID,
		Name:     fmt.Sprintf("%s Device", deviceType),
		Type:     sensorType,
		Location: fmt.Sprintf("room-%d", rand.Intn(5)+1),
	}

	data, err := json.Marshal(device)
	if err != nil {
		return fmt.Errorf("failed to marshal device: %w", err)
	}

	resp, err := client.Post(serverAddr+"/iot/devices", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to register device: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to register device, status: %d", resp.StatusCode)
	}

	logger.WithComponent("iot-client").WithField("device_id", deviceID).Info("Device registered successfully")
	return nil
}

func sendSensorData(client *http.Client, serverAddr string, data iot.SensorData, logger *logging.Logger) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal sensor data: %w", err)
	}

	resp, err := client.Post(serverAddr+"/iot/data", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send sensor data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to send sensor data, status: %d", resp.StatusCode)
	}

	return nil
}

func sendTestCommand(client *http.Client, serverAddr, deviceID string, logger *logging.Logger) error {
	command := iot.Command{
		ID:       fmt.Sprintf("cmd-%d", time.Now().UnixNano()),
		DeviceID: deviceID,
		Type:     "ping",
		Payload: map[string]interface{}{
			"message": "health check",
		},
		Timestamp: time.Now(),
		Priority:  1,
	}

	data, err := json.Marshal(command)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	resp, err := client.Post(serverAddr+"/iot/command", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to send command: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send command, status: %d", resp.StatusCode)
	}

	var response iot.Response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	logger.WithComponent("iot-client").WithFields(map[string]interface{}{
		"command_id": command.ID,
		"status":     response.Status,
	}).Debug("Command response received")

	return nil
}