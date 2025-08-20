package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"
)

// SensorData represents sensor readings
type SensorData struct {
	DeviceID    string    `json:"device_id"`
	SensorType  string    `json:"sensor_type"`
	Value       float64   `json:"value"`
	Unit        string    `json:"unit"`
	Timestamp   time.Time `json:"timestamp"`
	Quality     string    `json:"quality"`
}

func main() {
	var (
		serverAddr   = flag.String("server", "https://localhost:8443", "Server address")
		deviceID     = flag.String("device", "iot_client_001", "Device ID")
		sensorType   = flag.String("sensor", "temperature", "Sensor type (temperature, humidity, motion)")
		interval     = flag.Duration("interval", 5*time.Second, "Data transmission interval")
		duration     = flag.Duration("duration", 60*time.Second, "Total runtime duration")
		protocol     = flag.String("protocol", "quic", "Protocol to use (quic or tcp)")
	)
	flag.Parse()

	log.Printf("Starting IoT client: %s", *deviceID)
	log.Printf("Server: %s", *serverAddr)
	log.Printf("Sensor: %s", *sensorType)
	log.Printf("Interval: %v", *interval)
	log.Printf("Duration: %v", *duration)
	log.Printf("Protocol: %s", *protocol)

	// Create HTTP client with TLS config
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		Timeout: 10 * time.Second,
	}

	// Run simulation
	runSimulation(client, *serverAddr, *deviceID, *sensorType, *interval, *duration)
}

func runSimulation(client *http.Client, serverAddr, deviceID, sensorType string, interval, duration time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	timeout := time.After(duration)
	requestCount := 0
	successCount := 0

	for {
		select {
		case <-ticker.C:
			data := generateSensorData(deviceID, sensorType)
			
			if err := sendSensorData(client, serverAddr, data); err != nil {
				log.Printf("Failed to send data: %v", err)
			} else {
				successCount++
				log.Printf("Sent data: %s=%.2f%s", data.SensorType, data.Value, data.Unit)
			}
			requestCount++
			
		case <-timeout:
			log.Printf("Simulation completed: %d/%d requests successful", successCount, requestCount)
			return
		}
	}
}

func generateSensorData(deviceID, sensorType string) SensorData {
	data := SensorData{
		DeviceID:   deviceID,
		SensorType: sensorType,
		Timestamp:  time.Now(),
		Quality:    "reliable",
	}

	switch sensorType {
	case "temperature":
		data.Value = 18.0 + rand.Float64()*15.0 // 18-33°C
		data.Unit = "celsius"
	case "humidity":
		data.Value = 30.0 + rand.Float64()*40.0 // 30-70%
		data.Unit = "percent"
	case "motion":
		data.Value = float64(rand.Intn(2)) // 0 or 1
		data.Unit = "boolean"
		data.Quality = "unreliable" // Motion detection is less reliable
	case "pressure":
		data.Value = 1000.0 + rand.Float64()*50.0 // 1000-1050 hPa
		data.Unit = "hPa"
	case "light":
		data.Value = rand.Float64() * 1000.0 // 0-1000 lux
		data.Unit = "lux"
	default:
		data.Value = rand.Float64() * 100.0
		data.Unit = "unknown"
	}

	return data
}

func sendSensorData(client *http.Client, serverAddr string, data SensorData) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	url := serverAddr + "/iot/sensor"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Device-ID", data.DeviceID)
	req.Header.Set("X-Sensor-Type", data.SensorType)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	return nil
}