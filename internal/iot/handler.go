package iot

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// SensorData represents sensor readings
type SensorData struct {
	DeviceID    string    `json:"device_id"`
	SensorType  string    `json:"sensor_type"`
	Value       float64   `json:"value"`
	Unit        string    `json:"unit"`
	Timestamp   time.Time `json:"timestamp"`
	Quality     string    `json:"quality"` // "reliable" or "unreliable"
}

// Command represents a device command
type Command struct {
	DeviceID  string                 `json:"device_id"`
	Action    string                 `json:"action"`
	Parameters map[string]interface{} `json:"parameters"`
	Priority  string                 `json:"priority"` // "high", "medium", "low"
}

// Response represents a command response
type Response struct {
	CommandID string `json:"command_id"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	Data      interface{} `json:"data,omitempty"`
}

// Handler handles IoT HTTP requests
func Handler(w http.ResponseWriter, r *http.Request) {
	// Parse the URL path
	path := strings.TrimPrefix(r.URL.Path, "/iot/")
	parts := strings.Split(path, "/")
	
	if len(parts) == 0 {
		http.Error(w, "Invalid IoT endpoint", http.StatusBadRequest)
		return
	}

	switch parts[0] {
	case "sensor":
		handleSensorData(w, r)
	case "command":
		handleCommand(w, r)
	case "devices":
		handleDeviceList(w, r)
	case "simulate":
		handleSimulation(w, r)
	default:
		http.Error(w, "Unknown IoT endpoint", http.StatusNotFound)
	}
}

func handleSensorData(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Return simulated sensor data
		sensors := generateSensorData()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sensors)
	case http.MethodPost:
		// Accept sensor data from devices
		var data SensorData
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			http.Error(w, "Invalid sensor data", http.StatusBadRequest)
			return
		}
		
		log.Printf("Received sensor data: %+v", data)
		
		response := Response{
			Status:  "success",
			Message: "Sensor data received",
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleCommand(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var cmd Command
		if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
			http.Error(w, "Invalid command", http.StatusBadRequest)
			return
		}
		
		log.Printf("Received command: %+v", cmd)
		
		// Simulate command processing
		response := Response{
			CommandID: fmt.Sprintf("cmd_%d", time.Now().Unix()),
			Status:    "executed",
			Message:   fmt.Sprintf("Command %s executed on device %s", cmd.Action, cmd.DeviceID),
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleDeviceList(w http.ResponseWriter, r *http.Request) {
	devices := []map[string]interface{}{
		{"id": "temp_01", "type": "temperature", "status": "online", "location": "room_a"},
		{"id": "humid_01", "type": "humidity", "status": "online", "location": "room_a"},
		{"id": "motion_01", "type": "motion", "status": "online", "location": "hallway"},
		{"id": "temp_02", "type": "temperature", "status": "offline", "location": "room_b"},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"devices": devices,
		"count":   len(devices),
	})
}

func handleSimulation(w http.ResponseWriter, r *http.Request) {
	// Query parameters for simulation
	deviceCount := 10
	if dc := r.URL.Query().Get("devices"); dc != "" {
		if count, err := strconv.Atoi(dc); err == nil {
			deviceCount = count
		}
	}
	
	duration := 60 * time.Second
	if d := r.URL.Query().Get("duration"); d != "" {
		if dur, err := time.ParseDuration(d); err == nil {
			duration = dur
		}
	}
	
	log.Printf("Starting IoT simulation: %d devices for %v", deviceCount, duration)
	
	// Start simulation in background
	go runSimulation(deviceCount, duration)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "started",
		"devices":  deviceCount,
		"duration": duration.String(),
	})
}

func generateSensorData() []SensorData {
	now := time.Now()
	data := []SensorData{
		{
			DeviceID:   "temp_01",
			SensorType: "temperature",
			Value:      20.0 + rand.Float64()*10,
			Unit:       "celsius",
			Timestamp:  now,
			Quality:    "reliable",
		},
		{
			DeviceID:   "humid_01", 
			SensorType: "humidity",
			Value:      40.0 + rand.Float64()*20,
			Unit:       "percent",
			Timestamp:  now,
			Quality:    "reliable",
		},
		{
			DeviceID:   "motion_01",
			SensorType: "motion",
			Value:      float64(rand.Intn(2)), // 0 or 1
			Unit:       "boolean",
			Timestamp:  now,
			Quality:    "unreliable",
		},
	}
	
	return data
}

func runSimulation(deviceCount int, duration time.Duration) {
	end := time.Now().Add(duration)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	
	for time.Now().Before(end) {
		select {
		case <-ticker.C:
			for i := 0; i < deviceCount; i++ {
				data := SensorData{
					DeviceID:   fmt.Sprintf("sim_device_%d", i),
					SensorType: []string{"temperature", "humidity", "motion"}[rand.Intn(3)],
					Value:      rand.Float64() * 100,
					Unit:       "simulated",
					Timestamp:  time.Now(),
					Quality:    []string{"reliable", "unreliable"}[rand.Intn(2)],
				}
				log.Printf("Simulated data: %+v", data)
			}
		}
	}
	
	log.Println("IoT simulation completed")
}