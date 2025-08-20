package iot

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/quic-go/quic-go"
	"go.uber.org/zap"
)

const (
	ProtocolIoT = "IOT\x00"
)

// DeviceType represents the type of IoT device
type DeviceType string

const (
	DeviceTemperature DeviceType = "temperature"
	DeviceHumidity    DeviceType = "humidity"
	DeviceMotion      DeviceType = "motion"
	DeviceCamera      DeviceType = "camera"
	DeviceLight       DeviceType = "light"
)

// MessageType represents the type of IoT message
type MessageType string

const (
	MessageSensorData MessageType = "sensor_data"
	MessageCommand    MessageType = "command"
	MessageResponse   MessageType = "response"
	MessageHeartbeat  MessageType = "heartbeat"
)

// Device represents an IoT device
type Device struct {
	ID       string     `json:"id"`
	Type     DeviceType `json:"type"`
	Location string     `json:"location"`
	LastSeen time.Time  `json:"last_seen"`
	Online   bool       `json:"online"`
}

// SensorData represents sensor reading data
type SensorData struct {
	DeviceID  string      `json:"device_id"`
	Type      DeviceType  `json:"type"`
	Value     interface{} `json:"value"`
	Unit      string      `json:"unit"`
	Timestamp time.Time   `json:"timestamp"`
	Quality   float64     `json:"quality"` // 0.0 to 1.0
}

// Command represents a command sent to a device
type Command struct {
	ID       string      `json:"id"`
	DeviceID string      `json:"device_id"`
	Type     string      `json:"type"`
	Params   interface{} `json:"params"`
	Priority int         `json:"priority"` // 1-10, 10 being highest
}

// Message represents a generic IoT message
type Message struct {
	Type      MessageType `json:"type"`
	DeviceID  string      `json:"device_id"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
	Reliable  bool        `json:"reliable"` // Whether this requires reliable delivery
}

// Handler handles IoT protocol messages
type Handler struct {
	logger         *zap.Logger
	devices        map[string]*Device
	sensorDataChan chan SensorData
	commandChan    chan Command
}

// NewHandler creates a new IoT handler
func NewHandler(logger *zap.Logger) *Handler {
	return &Handler{
		logger:         logger,
		devices:        make(map[string]*Device),
		sensorDataChan: make(chan SensorData, 1000),
		commandChan:    make(chan Command, 100),
	}
}

// HandleStream implements the StreamHandler interface
func (h *Handler) HandleStream(ctx context.Context, stream quic.Stream) error {
	h.logger.Info("Handling IoT stream")

	decoder := json.NewDecoder(stream)
	encoder := json.NewEncoder(stream)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			var msg Message
			if err := decoder.Decode(&msg); err != nil {
				h.logger.Error("Failed to decode IoT message", zap.Error(err))
				return err
			}

			if err := h.processMessage(ctx, &msg, encoder); err != nil {
				h.logger.Error("Failed to process IoT message", zap.Error(err))
				return err
			}
		}
	}
}

// processMessage processes an incoming IoT message
func (h *Handler) processMessage(ctx context.Context, msg *Message, encoder *json.Encoder) error {
	switch msg.Type {
	case MessageSensorData:
		return h.handleSensorData(msg)
	case MessageCommand:
		return h.handleCommand(msg, encoder)
	case MessageHeartbeat:
		return h.handleHeartbeat(msg, encoder)
	default:
		return fmt.Errorf("unknown message type: %s", msg.Type)
	}
}

// handleSensorData processes sensor data messages
func (h *Handler) handleSensorData(msg *Message) error {
	dataBytes, err := json.Marshal(msg.Data)
	if err != nil {
		return err
	}

	var sensorData SensorData
	if err := json.Unmarshal(dataBytes, &sensorData); err != nil {
		return err
	}

	sensorData.Timestamp = msg.Timestamp
	sensorData.DeviceID = msg.DeviceID

	// Update device last seen
	if device, exists := h.devices[msg.DeviceID]; exists {
		device.LastSeen = time.Now()
		device.Online = true
	}

	// Send to processing channel (non-blocking)
	select {
	case h.sensorDataChan <- sensorData:
		h.logger.Debug("Sensor data processed",
			zap.String("device_id", sensorData.DeviceID),
			zap.String("type", string(sensorData.Type)),
			zap.Any("value", sensorData.Value),
		)
	default:
		h.logger.Warn("Sensor data channel full, dropping data",
			zap.String("device_id", sensorData.DeviceID),
		)
	}

	return nil
}

// handleCommand processes command messages
func (h *Handler) handleCommand(msg *Message, encoder *json.Encoder) error {
	commandBytes, err := json.Marshal(msg.Data)
	if err != nil {
		return err
	}

	var command Command
	if err := json.Unmarshal(commandBytes, &command); err != nil {
		return err
	}

	command.DeviceID = msg.DeviceID

	// Process command (simplified - in real implementation, this would route to device)
	h.logger.Info("Processing command",
		zap.String("device_id", command.DeviceID),
		zap.String("type", command.Type),
		zap.Int("priority", command.Priority),
	)

	// Send command to processing channel
	select {
	case h.commandChan <- command:
	default:
		h.logger.Warn("Command channel full", zap.String("device_id", command.DeviceID))
	}

	// Send response
	response := Message{
		Type:      MessageResponse,
		DeviceID:  msg.DeviceID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"command_id": command.ID,
			"status":     "received",
			"timestamp":  time.Now(),
		},
	}

	return encoder.Encode(response)
}

// handleHeartbeat processes heartbeat messages
func (h *Handler) handleHeartbeat(msg *Message, encoder *json.Encoder) error {
	// Update or create device
	device, exists := h.devices[msg.DeviceID]
	if !exists {
		device = &Device{
			ID:       msg.DeviceID,
			LastSeen: time.Now(),
			Online:   true,
		}
		h.devices[msg.DeviceID] = device
		h.logger.Info("New device registered", zap.String("device_id", msg.DeviceID))
	} else {
		device.LastSeen = time.Now()
		device.Online = true
	}

	// Send heartbeat response
	response := Message{
		Type:      MessageResponse,
		DeviceID:  msg.DeviceID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"status":    "alive",
			"timestamp": time.Now(),
		},
	}

	return encoder.Encode(response)
}

// GetSensorDataChannel returns the sensor data channel for external consumption
func (h *Handler) GetSensorDataChannel() <-chan SensorData {
	return h.sensorDataChan
}

// GetCommandChannel returns the command channel for external consumption
func (h *Handler) GetCommandChannel() <-chan Command {
	return h.commandChan
}

// GetDevices returns a copy of all registered devices
func (h *Handler) GetDevices() map[string]*Device {
	devices := make(map[string]*Device)
	for id, device := range h.devices {
		deviceCopy := *device
		devices[id] = &deviceCopy
	}
	return devices
}

// StartDeviceMonitor starts monitoring for offline devices
func (h *Handler) StartDeviceMonitor(ctx context.Context, timeout time.Duration) {
	ticker := time.NewTicker(timeout / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			for id, device := range h.devices {
				if now.Sub(device.LastSeen) > timeout {
					device.Online = false
					h.logger.Warn("Device went offline",
						zap.String("device_id", id),
						zap.Time("last_seen", device.LastSeen),
					)
				}
			}
		}
	}
}