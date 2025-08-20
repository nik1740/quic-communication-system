package iot

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/sirupsen/logrus"
)

// DeviceType represents different types of IoT devices
type DeviceType string

const (
	DeviceTypeTemperature DeviceType = "temperature"
	DeviceTypeHumidity    DeviceType = "humidity"
	DeviceTypeMotion      DeviceType = "motion"
	DeviceTypePressure    DeviceType = "pressure"
)

// SensorData represents sensor readings from IoT devices
type SensorData struct {
	DeviceID  string      `json:"device_id"`
	Type      DeviceType  `json:"type"`
	Value     float64     `json:"value"`
	Unit      string      `json:"unit"`
	Timestamp time.Time   `json:"timestamp"`
	Quality   int         `json:"quality"` // 0-100 reliability score
}

// ControlCommand represents commands sent to IoT devices
type ControlCommand struct {
	DeviceID  string                 `json:"device_id"`
	Command   string                 `json:"command"`
	Parameters map[string]interface{} `json:"parameters"`
	Timestamp time.Time              `json:"timestamp"`
	Priority  int                    `json:"priority"` // 0-10, higher is more critical
}

// ControlResponse represents responses from IoT devices
type ControlResponse struct {
	DeviceID  string    `json:"device_id"`
	Command   string    `json:"command"`
	Status    string    `json:"status"` // "success", "error", "pending"
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// Handler implements the QUIC StreamHandler interface for IoT communication
type Handler struct {
	logger          *logrus.Logger
	sensorDataChan  chan SensorData
	commandChan     chan ControlCommand
	responseChan    chan ControlResponse
	devices         map[string]*Device
}

// Device represents an IoT device
type Device struct {
	ID       string     `json:"id"`
	Type     DeviceType `json:"type"`
	Location string     `json:"location"`
	Status   string     `json:"status"`
	LastSeen time.Time  `json:"last_seen"`
}

// NewHandler creates a new IoT handler
func NewHandler(logger *logrus.Logger) *Handler {
	return &Handler{
		logger:          logger,
		sensorDataChan:  make(chan SensorData, 1000),
		commandChan:     make(chan ControlCommand, 100),
		responseChan:    make(chan ControlResponse, 100),
		devices:         make(map[string]*Device),
	}
}

// HandleStream handles incoming QUIC streams for IoT communication
func (h *Handler) HandleStream(ctx context.Context, stream *quic.Stream) error {
	h.logger.Info("Handling IoT stream")

	// Determine stream type based on first message
	decoder := json.NewDecoder(stream)
	encoder := json.NewEncoder(stream)

	// Read message type
	var msgType struct {
		Type string `json:"type"`
	}

	if err := decoder.Decode(&msgType); err != nil {
		return fmt.Errorf("failed to decode message type: %w", err)
	}

	switch msgType.Type {
	case "sensor_data":
		return h.handleSensorStream(ctx, stream, decoder)
	case "control_command":
		return h.handleControlStream(ctx, stream, decoder, encoder)
	case "device_registration":
		return h.handleDeviceRegistration(ctx, stream, decoder, encoder)
	default:
		return fmt.Errorf("unknown message type: %s", msgType.Type)
	}
}

// handleSensorStream handles unreliable sensor data streams
func (h *Handler) handleSensorStream(ctx context.Context, stream *quic.Stream, decoder *json.Decoder) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			var data SensorData
			if err := decoder.Decode(&data); err != nil {
				h.logger.WithError(err).Error("Failed to decode sensor data")
				continue
			}

			data.Timestamp = time.Now()
			h.logger.Infof("Received sensor data from %s: %f %s", data.DeviceID, data.Value, data.Unit)

			// Update device last seen
			if device, exists := h.devices[data.DeviceID]; exists {
				device.LastSeen = time.Now()
			}

			// Send to processing channel
			select {
			case h.sensorDataChan <- data:
			default:
				h.logger.Warn("Sensor data channel full, dropping data")
			}
		}
	}
}

// handleControlStream handles reliable control command streams
func (h *Handler) handleControlStream(ctx context.Context, stream *quic.Stream, decoder *json.Decoder, encoder *json.Encoder) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			var cmd ControlCommand
			if err := decoder.Decode(&cmd); err != nil {
				h.logger.WithError(err).Error("Failed to decode control command")
				continue
			}

			cmd.Timestamp = time.Now()
			h.logger.Infof("Received control command for %s: %s", cmd.DeviceID, cmd.Command)

			// Process command
			response := h.processControlCommand(cmd)

			// Send response
			if err := encoder.Encode(response); err != nil {
				h.logger.WithError(err).Error("Failed to encode control response")
				return err
			}
		}
	}
}

// handleDeviceRegistration handles device registration
func (h *Handler) handleDeviceRegistration(ctx context.Context, stream *quic.Stream, decoder *json.Decoder, encoder *json.Encoder) error {
	var device Device
	if err := decoder.Decode(&device); err != nil {
		return fmt.Errorf("failed to decode device registration: %w", err)
	}

	device.Status = "online"
	device.LastSeen = time.Now()
	h.devices[device.ID] = &device

	h.logger.Infof("Registered device %s (%s) at %s", device.ID, device.Type, device.Location)

	// Send registration confirmation
	response := map[string]string{
		"status":  "registered",
		"message": "Device successfully registered",
	}

	return encoder.Encode(response)
}

// processControlCommand processes control commands and returns responses
func (h *Handler) processControlCommand(cmd ControlCommand) ControlResponse {
	response := ControlResponse{
		DeviceID:  cmd.DeviceID,
		Command:   cmd.Command,
		Timestamp: time.Now(),
	}

	// Check if device exists
	device, exists := h.devices[cmd.DeviceID]
	if !exists {
		response.Status = "error"
		response.Message = "Device not found"
		return response
	}

	// Simulate command processing based on device type and command
	switch cmd.Command {
	case "read_sensor":
		response.Status = "success"
		response.Message = "Sensor reading initiated"
	case "set_threshold":
		response.Status = "success"
		response.Message = "Threshold updated"
	case "calibrate":
		response.Status = "success"
		response.Message = "Calibration started"
	case "restart":
		response.Status = "success"
		response.Message = "Device restart initiated"
		device.Status = "restarting"
	default:
		response.Status = "error"
		response.Message = "Unknown command"
	}

	return response
}

// GetSensorDataChannel returns the channel for receiving sensor data
func (h *Handler) GetSensorDataChannel() <-chan SensorData {
	return h.sensorDataChan
}

// GetDevices returns a copy of registered devices
func (h *Handler) GetDevices() map[string]Device {
	devices := make(map[string]Device)
	for id, device := range h.devices {
		devices[id] = *device
	}
	return devices
}

// GetDeviceStatus returns the status of a specific device
func (h *Handler) GetDeviceStatus(deviceID string) (Device, bool) {
	device, exists := h.devices[deviceID]
	if !exists {
		return Device{}, false
	}
	return *device, true
}