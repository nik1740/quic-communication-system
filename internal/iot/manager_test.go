package iot

import (
	"testing"
	"time"

	"github.com/nik1740/quic-communication-system/pkg/logging"
)

func TestIoTManager(t *testing.T) {
	// Initialize logger
	logger, err := logging.NewLogger("info", "text", "")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Create IoT manager
	manager := NewManager(logger.Logger, 100)

	// Test device registration
	device := &Device{
		ID:       "test-sensor-001",
		Name:     "Test Sensor",
		Type:     TemperatureSensor,
		Location: "test-room",
	}

	manager.RegisterDevice(device)

	// Verify device was registered
	retrievedDevice, exists := manager.GetDevice("test-sensor-001")
	if !exists {
		t.Fatal("Device was not registered")
	}

	if retrievedDevice.Name != "Test Sensor" {
		t.Errorf("Expected device name 'Test Sensor', got '%s'", retrievedDevice.Name)
	}

	// Test sensor data
	sensorData := SensorData{
		ID:        "test-sensor-001",
		Type:      TemperatureSensor,
		Value:     25.5,
		Unit:      "°C",
		Timestamp: time.Now(),
		Location:  "test-room",
		Quality:   "good",
	}

	manager.AddSensorData(sensorData)

	// Verify data was added
	readings := manager.GetRecentReadings(10)
	if len(readings) == 0 {
		t.Fatal("No sensor readings found")
	}

	if readings[0].Value != 25.5 {
		t.Errorf("Expected sensor value 25.5, got %f", readings[0].Value)
	}

	// Test command processing
	command := Command{
		ID:       "test-cmd-001",
		DeviceID: "test-sensor-001",
		Type:     "ping",
		Payload:  map[string]interface{}{"message": "test"},
		Priority: 1,
	}

	response := manager.ProcessCommand(command)
	if response.Status != "success" {
		t.Errorf("Expected command status 'success', got '%s'", response.Status)
	}

	// Test data generation
	generatedData := GenerateTestSensorData("test-device", TemperatureSensor)
	if generatedData.ID != "test-device" {
		t.Errorf("Expected device ID 'test-device', got '%s'", generatedData.ID)
	}

	if generatedData.Type != TemperatureSensor {
		t.Errorf("Expected sensor type 'temperature', got '%s'", generatedData.Type)
	}

	if generatedData.Unit != "°C" {
		t.Errorf("Expected unit '°C', got '%s'", generatedData.Unit)
	}
}