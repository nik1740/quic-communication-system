package tcp

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/nik1740/quic-communication-system/internal/iot"
	"github.com/sirupsen/logrus"
)

// Server represents a TCP/TLS server for comparison with QUIC
type Server struct {
	listener net.Listener
	logger   *logrus.Logger
	clients  map[string]*ClientConnection
	clientsMux sync.RWMutex
	iotHandler *iot.Handler
}

// ClientConnection represents a connected client
type ClientConnection struct {
	ID         string
	Conn       net.Conn
	DeviceID   string
	ConnTime   time.Time
	LastSeen   time.Time
	BytesSent  int64
	BytesRecv  int64
}

// NewServer creates a new TCP/TLS server
func NewServer(addr string, tlsConfig *tls.Config, logger *logrus.Logger) (*Server, error) {
	var listener net.Listener
	var err error

	if tlsConfig != nil {
		listener, err = tls.Listen("tcp", addr, tlsConfig)
	} else {
		listener, err = net.Listen("tcp", addr)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to start TCP listener: %w", err)
	}

	return &Server{
		listener:   listener,
		logger:     logger,
		clients:    make(map[string]*ClientConnection),
		iotHandler: iot.NewHandler(logger),
	}, nil
}

// Start starts the TCP/TLS server
func (s *Server) Start(ctx context.Context) error {
	s.logger.Infof("TCP/TLS server starting on %s", s.listener.Addr())

	// Start cleanup routine
	go s.cleanupRoutine(ctx)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				s.logger.WithError(err).Error("Failed to accept connection")
				continue
			}

			go s.handleConnection(ctx, conn)
		}
	}
}

// Stop stops the TCP/TLS server
func (s *Server) Stop() error {
	s.clientsMux.Lock()
	defer s.clientsMux.Unlock()

	// Close all client connections
	for _, client := range s.clients {
		client.Conn.Close()
	}

	return s.listener.Close()
}

// Addr returns the server's listening address
func (s *Server) Addr() net.Addr {
	return s.listener.Addr()
}

func (s *Server) handleConnection(ctx context.Context, conn net.Conn) {
	clientID := fmt.Sprintf("client_%d", time.Now().UnixNano())
	
	client := &ClientConnection{
		ID:       clientID,
		Conn:     conn,
		ConnTime: time.Now(),
		LastSeen: time.Now(),
	}

	s.clientsMux.Lock()
	s.clients[clientID] = client
	s.clientsMux.Unlock()

	defer func() {
		conn.Close()
		s.clientsMux.Lock()
		delete(s.clients, clientID)
		s.clientsMux.Unlock()
		s.logger.Infof("Client %s disconnected", clientID)
	}()

	s.logger.Infof("New TCP connection from %s (client: %s)", conn.RemoteAddr(), clientID)

	// Set connection timeouts
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetWriteDeadline(time.Now().Add(60 * time.Second))

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Reset read deadline
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))

			var msgType struct {
				Type string `json:"type"`
			}

			if err := decoder.Decode(&msgType); err != nil {
				s.logger.WithError(err).Debug("Failed to decode message type")
				return
			}

			client.LastSeen = time.Now()

			switch msgType.Type {
			case "device_registration":
				if err := s.handleDeviceRegistration(client, decoder, encoder); err != nil {
					s.logger.WithError(err).Error("Failed to handle device registration")
					return
				}

			case "sensor_data":
				if err := s.handleSensorData(client, decoder); err != nil {
					s.logger.WithError(err).Error("Failed to handle sensor data")
					return
				}

			case "control_command":
				if err := s.handleControlCommand(client, decoder, encoder); err != nil {
					s.logger.WithError(err).Error("Failed to handle control command")
					return
				}

			default:
				s.logger.Warnf("Unknown message type: %s", msgType.Type)
			}
		}
	}
}

func (s *Server) handleDeviceRegistration(client *ClientConnection, decoder *json.Decoder, encoder *json.Encoder) error {
	var device iot.Device
	if err := decoder.Decode(&device); err != nil {
		return fmt.Errorf("failed to decode device registration: %w", err)
	}

	client.DeviceID = device.ID
	s.logger.Infof("Device %s registered via TCP (client: %s)", device.ID, client.ID)

	// Send confirmation
	response := map[string]string{
		"status":  "registered",
		"message": "Device successfully registered via TCP",
	}

	return encoder.Encode(response)
}

func (s *Server) handleSensorData(client *ClientConnection, decoder *json.Decoder) error {
	var data iot.SensorData
	if err := decoder.Decode(&data); err != nil {
		return fmt.Errorf("failed to decode sensor data: %w", err)
	}

	data.Timestamp = time.Now()
	client.DeviceID = data.DeviceID

	s.logger.Debugf("Received TCP sensor data from %s: %f %s", data.DeviceID, data.Value, data.Unit)

	// Process through IoT handler for consistency
	sensorChan := s.iotHandler.GetSensorDataChannel()
	select {
	case sensorChan <- data:
	default:
		s.logger.Warn("Sensor data channel full, dropping data")
	}

	return nil
}

func (s *Server) handleControlCommand(client *ClientConnection, decoder *json.Decoder, encoder *json.Encoder) error {
	var cmd iot.ControlCommand
	if err := decoder.Decode(&cmd); err != nil {
		return fmt.Errorf("failed to decode control command: %w", err)
	}

	s.logger.Infof("Received TCP control command for %s: %s", cmd.DeviceID, cmd.Command)

	// Process command (simplified for demo)
	response := iot.ControlResponse{
		DeviceID:  cmd.DeviceID,
		Command:   cmd.Command,
		Status:    "success",
		Message:   "Command executed via TCP",
		Timestamp: time.Now(),
	}

	return encoder.Encode(response)
}

func (s *Server) cleanupRoutine(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.cleanupStaleConnections()
		}
	}
}

func (s *Server) cleanupStaleConnections() {
	s.clientsMux.Lock()
	defer s.clientsMux.Unlock()

	staleThreshold := time.Now().Add(-5 * time.Minute)
	
	for id, client := range s.clients {
		if client.LastSeen.Before(staleThreshold) {
			s.logger.Infof("Closing stale connection: %s", id)
			client.Conn.Close()
			delete(s.clients, id)
		}
	}
}

// GetConnectedClients returns the number of connected clients
func (s *Server) GetConnectedClients() int {
	s.clientsMux.RLock()
	defer s.clientsMux.RUnlock()
	return len(s.clients)
}

// GetClientStats returns statistics about connected clients
func (s *Server) GetClientStats() map[string]*ClientConnection {
	s.clientsMux.RLock()
	defer s.clientsMux.RUnlock()
	
	stats := make(map[string]*ClientConnection)
	for id, client := range s.clients {
		// Create a copy to avoid race conditions
		stats[id] = &ClientConnection{
			ID:        client.ID,
			DeviceID:  client.DeviceID,
			ConnTime:  client.ConnTime,
			LastSeen:  client.LastSeen,
			BytesSent: client.BytesSent,
			BytesRecv: client.BytesRecv,
		}
	}
	
	return stats
}

// GetIoTHandler returns the IoT handler for accessing sensor data
func (s *Server) GetIoTHandler() *iot.Handler {
	return s.iotHandler
}