package iot

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

// Handler handles IoT HTTP requests
type Handler struct {
	manager *Manager
	logger  *logrus.Logger
}

// NewHandler creates a new IoT handler
func NewHandler(manager *Manager, logger *logrus.Logger) *Handler {
	return &Handler{
		manager: manager,
		logger:  logger,
	}
}

// RegisterRoutes registers IoT routes with the server
func (h *Handler) RegisterRoutes(registerFunc func(string, http.HandlerFunc)) {
	registerFunc("/iot/devices", h.handleDevices)
	registerFunc("/iot/data", h.handleSensorData)
	registerFunc("/iot/command", h.handleCommand)
	registerFunc("/iot/health", h.handleHealth)
}

// handleDevices handles device registration and listing
func (h *Handler) handleDevices(w http.ResponseWriter, r *http.Request) {
	h.logger.WithFields(logrus.Fields{
		"method":    r.Method,
		"path":      r.URL.Path,
		"component": "iot-handler",
	}).Info("Handling devices request")

	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		h.listDevices(w, r)
	case http.MethodPost:
		h.registerDevice(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// listDevices returns all registered devices
func (h *Handler) listDevices(w http.ResponseWriter, r *http.Request) {
	devices := h.manager.GetAllDevices()
	
	response := map[string]interface{}{
		"devices": devices,
		"count":   len(devices),
		"timestamp": time.Now(),
	}
	
	json.NewEncoder(w).Encode(response)
}

// registerDevice registers a new device
func (h *Handler) registerDevice(w http.ResponseWriter, r *http.Request) {
	var device Device
	if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	if device.ID == "" || device.Name == "" {
		http.Error(w, "ID and name are required", http.StatusBadRequest)
		return
	}
	
	h.manager.RegisterDevice(&device)
	
	response := map[string]interface{}{
		"status":  "success",
		"message": "Device registered successfully",
		"device":  device,
	}
	
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// handleSensorData handles sensor data submission and retrieval
func (h *Handler) handleSensorData(w http.ResponseWriter, r *http.Request) {
	h.logger.WithFields(logrus.Fields{
		"method":    r.Method,
		"path":      r.URL.Path,
		"component": "iot-handler",
	}).Info("Handling sensor data request")

	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		h.getSensorData(w, r)
	case http.MethodPost:
		h.submitSensorData(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getSensorData returns recent sensor readings
func (h *Handler) getSensorData(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 100 // default
	
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	
	readings := h.manager.GetRecentReadings(limit)
	
	response := map[string]interface{}{
		"readings":  readings,
		"count":     len(readings),
		"limit":     limit,
		"timestamp": time.Now(),
	}
	
	json.NewEncoder(w).Encode(response)
}

// submitSensorData accepts new sensor readings
func (h *Handler) submitSensorData(w http.ResponseWriter, r *http.Request) {
	var data SensorData
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	if data.ID == "" || data.Type == "" {
		http.Error(w, "ID and type are required", http.StatusBadRequest)
		return
	}
	
	h.manager.AddSensorData(data)
	
	response := map[string]interface{}{
		"status":  "success",
		"message": "Sensor data received",
		"data":    data,
	}
	
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// handleCommand handles IoT commands
func (h *Handler) handleCommand(w http.ResponseWriter, r *http.Request) {
	h.logger.WithFields(logrus.Fields{
		"method":    r.Method,
		"path":      r.URL.Path,
		"component": "iot-handler",
	}).Info("Handling command request")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var command Command
	if err := json.NewDecoder(r.Body).Decode(&command); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	if command.DeviceID == "" || command.Type == "" {
		http.Error(w, "DeviceID and type are required", http.StatusBadRequest)
		return
	}
	
	if command.ID == "" {
		command.ID = fmt.Sprintf("cmd-%d", time.Now().UnixNano())
	}
	
	response := h.manager.ProcessCommand(command)
	json.NewEncoder(w).Encode(response)
}

// handleHealth handles health check requests
func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	
	devices := h.manager.GetAllDevices()
	onlineCount := 0
	for _, device := range devices {
		if device.Online {
			onlineCount++
		}
	}
	
	response := map[string]interface{}{
		"status":        "healthy",
		"timestamp":     time.Now(),
		"total_devices": len(devices),
		"online_devices": onlineCount,
		"component":     "iot-service",
	}
	
	json.NewEncoder(w).Encode(response)
}