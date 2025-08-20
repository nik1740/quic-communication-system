package iot

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// SensorType represents different types of sensors
type SensorType string

const (
	TemperatureSensor SensorType = "temperature"
	HumiditySensor    SensorType = "humidity"
	MotionSensor      SensorType = "motion"
	LightSensor       SensorType = "light"
)

// SensorData represents sensor reading data
type SensorData struct {
	ID        string      `json:"id"`
	Type      SensorType  `json:"type"`
	Value     float64     `json:"value"`
	Unit      string      `json:"unit"`
	Timestamp time.Time   `json:"timestamp"`
	Location  string      `json:"location,omitempty"`
	Quality   string      `json:"quality,omitempty"`
}

// Command represents an IoT command
type Command struct {
	ID        string                 `json:"id"`
	DeviceID  string                 `json:"device_id"`
	Type      string                 `json:"type"`
	Payload   map[string]interface{} `json:"payload"`
	Timestamp time.Time              `json:"timestamp"`
	Priority  int                    `json:"priority"`
}

// Response represents an IoT command response
type Response struct {
	CommandID string      `json:"command_id"`
	DeviceID  string      `json:"device_id"`
	Status    string      `json:"status"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// Device represents an IoT device
type Device struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	Type     SensorType `json:"type"`
	Location string     `json:"location"`
	Online   bool       `json:"online"`
	LastSeen time.Time  `json:"last_seen"`
}

// Manager manages IoT devices and data
type Manager struct {
	devices   map[string]*Device
	readings  []SensorData
	mu        sync.RWMutex
	logger    *logrus.Logger
	dataLimit int
}

// NewManager creates a new IoT manager
func NewManager(logger *logrus.Logger, dataLimit int) *Manager {
	return &Manager{
		devices:   make(map[string]*Device),
		readings:  make([]SensorData, 0),
		logger:    logger,
		dataLimit: dataLimit,
	}
}

// RegisterDevice registers a new IoT device
func (m *Manager) RegisterDevice(device *Device) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	device.Online = true
	device.LastSeen = time.Now()
	m.devices[device.ID] = device
	
	m.logger.WithFields(logrus.Fields{
		"device_id": device.ID,
		"type":      device.Type,
		"location":  device.Location,
		"component": "iot-manager",
	}).Info("Device registered")
}

// AddSensorData adds new sensor reading
func (m *Manager) AddSensorData(data SensorData) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	data.Timestamp = time.Now()
	m.readings = append(m.readings, data)
	
	// Limit data retention
	if len(m.readings) > m.dataLimit {
		m.readings = m.readings[len(m.readings)-m.dataLimit:]
	}
	
	// Update device last seen
	if device, exists := m.devices[data.ID]; exists {
		device.LastSeen = time.Now()
		device.Online = true
	}
	
	m.logger.WithFields(logrus.Fields{
		"device_id": data.ID,
		"type":      data.Type,
		"value":     data.Value,
		"component": "iot-manager",
	}).Debug("Sensor data added")
}

// GetDevice returns device information
func (m *Manager) GetDevice(id string) (*Device, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	device, exists := m.devices[id]
	return device, exists
}

// GetAllDevices returns all registered devices
func (m *Manager) GetAllDevices() []*Device {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	devices := make([]*Device, 0, len(m.devices))
	for _, device := range m.devices {
		devices = append(devices, device)
	}
	return devices
}

// GetRecentReadings returns recent sensor readings
func (m *Manager) GetRecentReadings(limit int) []SensorData {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if limit <= 0 || limit > len(m.readings) {
		limit = len(m.readings)
	}
	
	start := len(m.readings) - limit
	return append([]SensorData(nil), m.readings[start:]...)
}

// ProcessCommand processes an IoT command
func (m *Manager) ProcessCommand(cmd Command) Response {
	m.logger.WithFields(logrus.Fields{
		"command_id": cmd.ID,
		"device_id":  cmd.DeviceID,
		"type":       cmd.Type,
		"component":  "iot-manager",
	}).Info("Processing command")
	
	device, exists := m.GetDevice(cmd.DeviceID)
	if !exists {
		return Response{
			CommandID: cmd.ID,
			DeviceID:  cmd.DeviceID,
			Status:    "error",
			Error:     "device not found",
			Timestamp: time.Now(),
		}
	}
	
	if !device.Online {
		return Response{
			CommandID: cmd.ID,
			DeviceID:  cmd.DeviceID,
			Status:    "error",
			Error:     "device offline",
			Timestamp: time.Now(),
		}
	}
	
	// Simulate command processing
	time.Sleep(time.Millisecond * time.Duration(rand.Intn(100)+50))
	
	return Response{
		CommandID: cmd.ID,
		DeviceID:  cmd.DeviceID,
		Status:    "success",
		Data:      map[string]interface{}{"result": "command executed"},
		Timestamp: time.Now(),
	}
}

// GenerateTestSensorData generates realistic test sensor data
func GenerateTestSensorData(deviceID string, sensorType SensorType) SensorData {
	var value float64
	var unit string
	
	switch sensorType {
	case TemperatureSensor:
		value = 20.0 + rand.Float64()*10.0 // 20-30°C
		unit = "°C"
	case HumiditySensor:
		value = 40.0 + rand.Float64()*20.0 // 40-60%
		unit = "%"
	case MotionSensor:
		value = float64(rand.Intn(2)) // 0 or 1
		unit = "boolean"
	case LightSensor:
		value = rand.Float64() * 1000.0 // 0-1000 lux
		unit = "lux"
	default:
		value = rand.Float64() * 100.0
		unit = "unknown"
	}
	
	return SensorData{
		ID:        deviceID,
		Type:      sensorType,
		Value:     value,
		Unit:      unit,
		Timestamp: time.Now(),
		Location:  fmt.Sprintf("room-%d", rand.Intn(10)+1),
		Quality:   "good",
	}
}