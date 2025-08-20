package tcp

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"
	"github.com/nik1740/quic-communication-system/internal/iot"
	"github.com/nik1740/quic-communication-system/internal/streaming"
)

// Server represents a TCP/TLS server for comparison with QUIC
type Server struct {
	listener net.Listener
	logger   *zap.Logger
	handlers map[string]ConnectionHandler
	tlsConfig *tls.Config
}

// ConnectionHandler defines the interface for handling TCP connections
type ConnectionHandler interface {
	HandleConnection(ctx context.Context, conn net.Conn) error
}

// NewServer creates a new TCP/TLS server
func NewServer(addr string, tlsConfig *tls.Config, logger *zap.Logger) (*Server, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create TCP listener: %w", err)
	}

	if tlsConfig != nil {
		listener = tls.NewListener(listener, tlsConfig)
	}

	return &Server{
		listener:  listener,
		logger:    logger,
		handlers:  make(map[string]ConnectionHandler),
		tlsConfig: tlsConfig,
	}, nil
}

// RegisterHandler registers a connection handler for a specific protocol
func (s *Server) RegisterHandler(protocol string, handler ConnectionHandler) {
	s.handlers[protocol] = handler
}

// Start starts the TCP/TLS server
func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("Starting TCP/TLS server", zap.String("addr", s.listener.Addr().String()))

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				s.logger.Error("Failed to accept connection", zap.Error(err))
				continue
			}

			go s.handleConnection(ctx, conn)
		}
	}
}

// Stop stops the TCP/TLS server
func (s *Server) Stop() error {
	return s.listener.Close()
}

// handleConnection handles a new TCP connection
func (s *Server) handleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	s.logger.Info("New TCP connection", zap.String("remote_addr", conn.RemoteAddr().String()))

	// Read protocol header (first 4 bytes)
	protocolBytes := make([]byte, 4)
	n, err := conn.Read(protocolBytes)
	if err != nil || n != 4 {
		s.logger.Error("Failed to read protocol header", zap.Error(err))
		return
	}

	protocol := string(protocolBytes)
	handler, exists := s.handlers[protocol]
	if !exists {
		s.logger.Warn("Unknown protocol", zap.String("protocol", protocol))
		return
	}

	if err := handler.HandleConnection(ctx, conn); err != nil {
		s.logger.Error("Connection handler error",
			zap.String("protocol", protocol),
			zap.Error(err),
		)
	}
}

// IoTHandler handles IoT protocol over TCP/TLS
type IoTHandler struct {
	logger         *zap.Logger
	devices        map[string]*iot.Device
	sensorDataChan chan iot.SensorData
	commandChan    chan iot.Command
	mutex          sync.RWMutex
}

// NewIoTHandler creates a new TCP IoT handler
func NewIoTHandler(logger *zap.Logger) *IoTHandler {
	return &IoTHandler{
		logger:         logger,
		devices:        make(map[string]*iot.Device),
		sensorDataChan: make(chan iot.SensorData, 1000),
		commandChan:    make(chan iot.Command, 100),
	}
}

// HandleConnection implements the ConnectionHandler interface for IoT
func (h *IoTHandler) HandleConnection(ctx context.Context, conn net.Conn) error {
	h.logger.Info("Handling IoT TCP connection")

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			var msg iot.Message
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

// processMessage processes an incoming IoT message (similar to QUIC version)
func (h *IoTHandler) processMessage(ctx context.Context, msg *iot.Message, encoder *json.Encoder) error {
	switch msg.Type {
	case iot.MessageSensorData:
		return h.handleSensorData(msg)
	case iot.MessageCommand:
		return h.handleCommand(msg, encoder)
	case iot.MessageHeartbeat:
		return h.handleHeartbeat(msg, encoder)
	default:
		return fmt.Errorf("unknown message type: %s", msg.Type)
	}
}

// handleSensorData processes sensor data messages
func (h *IoTHandler) handleSensorData(msg *iot.Message) error {
	dataBytes, err := json.Marshal(msg.Data)
	if err != nil {
		return err
	}

	var sensorData iot.SensorData
	if err := json.Unmarshal(dataBytes, &sensorData); err != nil {
		return err
	}

	sensorData.Timestamp = msg.Timestamp
	sensorData.DeviceID = msg.DeviceID

	h.mutex.Lock()
	if device, exists := h.devices[msg.DeviceID]; exists {
		device.LastSeen = time.Now()
		device.Online = true
	}
	h.mutex.Unlock()

	select {
	case h.sensorDataChan <- sensorData:
		h.logger.Debug("Sensor data processed (TCP)",
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
func (h *IoTHandler) handleCommand(msg *iot.Message, encoder *json.Encoder) error {
	commandBytes, err := json.Marshal(msg.Data)
	if err != nil {
		return err
	}

	var command iot.Command
	if err := json.Unmarshal(commandBytes, &command); err != nil {
		return err
	}

	command.DeviceID = msg.DeviceID

	h.logger.Info("Processing command (TCP)",
		zap.String("device_id", command.DeviceID),
		zap.String("type", command.Type),
		zap.Int("priority", command.Priority),
	)

	select {
	case h.commandChan <- command:
	default:
		h.logger.Warn("Command channel full", zap.String("device_id", command.DeviceID))
	}

	response := iot.Message{
		Type:      iot.MessageResponse,
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
func (h *IoTHandler) handleHeartbeat(msg *iot.Message, encoder *json.Encoder) error {
	h.mutex.Lock()
	device, exists := h.devices[msg.DeviceID]
	if !exists {
		device = &iot.Device{
			ID:       msg.DeviceID,
			LastSeen: time.Now(),
			Online:   true,
		}
		h.devices[msg.DeviceID] = device
		h.logger.Info("New device registered (TCP)", zap.String("device_id", msg.DeviceID))
	} else {
		device.LastSeen = time.Now()
		device.Online = true
	}
	h.mutex.Unlock()

	response := iot.Message{
		Type:      iot.MessageResponse,
		DeviceID:  msg.DeviceID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"status":    "alive",
			"timestamp": time.Now(),
		},
	}

	return encoder.Encode(response)
}

// StreamingHandler handles video streaming over TCP/TLS
type StreamingHandler struct {
	logger         *zap.Logger
	activeStreams  map[string]*TCPActiveStream
	videoQualities []streaming.VideoQuality
	chunkDuration  time.Duration
	mutex          sync.RWMutex
}

// TCPActiveStream represents an active TCP streaming session
type TCPActiveStream struct {
	ID              string
	Quality         streaming.VideoQuality
	StartTime       time.Time
	Stats           streaming.StreamStats
	AdaptiveBitrate bool
	Conn            net.Conn
	Cancel          context.CancelFunc
	mutex           sync.RWMutex
}

// NewStreamingHandler creates a new TCP streaming handler
func NewStreamingHandler(logger *zap.Logger) *StreamingHandler {
	return &StreamingHandler{
		logger:        logger,
		activeStreams: make(map[string]*TCPActiveStream),
		videoQualities: []streaming.VideoQuality{
			{Name: "360p", Width: 640, Height: 360, Bitrate: 500, Framerate: 30},
			{Name: "480p", Width: 854, Height: 480, Bitrate: 1000, Framerate: 30},
			{Name: "720p", Width: 1280, Height: 720, Bitrate: 2000, Framerate: 30},
			{Name: "1080p", Width: 1920, Height: 1080, Bitrate: 4000, Framerate: 30},
		},
		chunkDuration: 2 * time.Second,
	}
}

// HandleConnection implements the ConnectionHandler interface for streaming
func (h *StreamingHandler) HandleConnection(ctx context.Context, conn net.Conn) error {
	h.logger.Info("Handling streaming TCP connection")

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	var request streaming.StreamRequest
	if err := decoder.Decode(&request); err != nil {
		h.logger.Error("Failed to decode stream request", zap.Error(err))
		return err
	}

	h.logger.Info("Stream request received (TCP)",
		zap.String("stream_id", request.StreamID),
		zap.String("quality", request.Quality.Name),
		zap.Bool("adaptive", request.AdaptiveBitrate),
	)

	quality, err := h.validateQuality(request.Quality)
	if err != nil {
		response := streaming.StreamResponse{
			StreamID:           request.StreamID,
			Status:            "error",
			AvailableQualities: h.videoQualities,
		}
		encoder.Encode(response)
		return err
	}

	response := streaming.StreamResponse{
		StreamID:           request.StreamID,
		Status:            "accepted",
		AvailableQualities: h.videoQualities,
		ChunkDuration:     h.chunkDuration,
	}

	if err := encoder.Encode(response); err != nil {
		return err
	}

	return h.startStream(ctx, request.StreamID, quality, request.AdaptiveBitrate, conn)
}

// validateQuality validates and finds the closest available quality
func (h *StreamingHandler) validateQuality(requested streaming.VideoQuality) (streaming.VideoQuality, error) {
	for _, quality := range h.videoQualities {
		if quality.Name == requested.Name ||
			(quality.Width == requested.Width && quality.Height == requested.Height) {
			return quality, nil
		}
	}

	var closest streaming.VideoQuality
	minDiff := int(^uint(0) >> 1)

	for _, quality := range h.videoQualities {
		diff := abs(quality.Bitrate - requested.Bitrate)
		if diff < minDiff {
			minDiff = diff
			closest = quality
		}
	}

	if closest.Name == "" {
		return streaming.VideoQuality{}, fmt.Errorf("no suitable quality found")
	}

	return closest, nil
}

// startStream starts streaming video data over TCP
func (h *StreamingHandler) startStream(ctx context.Context, streamID string, quality streaming.VideoQuality, adaptive bool, conn net.Conn) error {
	streamCtx, cancel := context.WithCancel(ctx)

	activeStream := &TCPActiveStream{
		ID:              streamID,
		Quality:         quality,
		StartTime:       time.Now(),
		AdaptiveBitrate: adaptive,
		Conn:            conn,
		Cancel:          cancel,
		Stats: streaming.StreamStats{
			StreamID:      streamID,
			LastChunkTime: time.Now(),
		},
	}

	h.mutex.Lock()
	h.activeStreams[streamID] = activeStream
	h.mutex.Unlock()

	defer func() {
		h.mutex.Lock()
		delete(h.activeStreams, streamID)
		h.mutex.Unlock()
		cancel()
	}()

	return h.streamVideo(streamCtx, activeStream)
}

// streamVideo continuously streams video chunks over TCP
func (h *StreamingHandler) streamVideo(ctx context.Context, activeStream *TCPActiveStream) error {
	ticker := time.NewTicker(h.chunkDuration)
	defer ticker.Stop()

	sequenceNum := uint64(0)
	chunkSize := h.calculateChunkSize(activeStream.Quality)
	encoder := json.NewEncoder(activeStream.Conn)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			chunk := h.generateVideoChunk(activeStream.ID, sequenceNum, activeStream.Quality, chunkSize)
			
			if err := encoder.Encode(chunk); err != nil {
				h.logger.Error("Failed to send video chunk (TCP)",
					zap.String("stream_id", activeStream.ID),
					zap.Uint64("sequence", sequenceNum),
					zap.Error(err),
				)
				return err
			}

			h.updateStreamStats(activeStream, chunk)
			sequenceNum++
		}
	}
}

// generateVideoChunk generates a simulated video chunk
func (h *StreamingHandler) generateVideoChunk(streamID string, sequenceNum uint64, quality streaming.VideoQuality, size int) streaming.VideoChunk {
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(sequenceNum % 256)
	}

	return streaming.VideoChunk{
		StreamID:    streamID,
		SequenceNum: sequenceNum,
		Timestamp:   time.Now(),
		Quality:     quality,
		Data:        data,
		Size:        size,
		IsKeyframe:  sequenceNum%30 == 0,
		Duration:    h.chunkDuration,
	}
}

// calculateChunkSize calculates the size of a video chunk
func (h *StreamingHandler) calculateChunkSize(quality streaming.VideoQuality) int {
	bytesPerSecond := quality.Bitrate * 1000 / 8
	return int(float64(bytesPerSecond) * h.chunkDuration.Seconds())
}

// updateStreamStats updates streaming statistics
func (h *StreamingHandler) updateStreamStats(activeStream *TCPActiveStream, chunk streaming.VideoChunk) {
	activeStream.mutex.Lock()
	defer activeStream.mutex.Unlock()

	activeStream.Stats.BytesSent += uint64(chunk.Size)
	activeStream.Stats.ChunksSent++
	activeStream.Stats.LastChunkTime = time.Now()

	duration := time.Since(activeStream.StartTime).Seconds()
	if duration > 0 {
		activeStream.Stats.AverageBitrate = float64(activeStream.Stats.BytesSent*8) / duration / 1000
	}

	activeStream.Stats.BufferHealth = 0.8 + 0.2*float64(time.Now().UnixNano()%100)/100
	activeStream.Stats.PacketLoss = float64(time.Now().UnixNano()%100) / 10000
}

// Helper function
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}