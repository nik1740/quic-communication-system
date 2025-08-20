package streaming

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
	"go.uber.org/zap"
)

const (
	ProtocolStreaming = "STRM"
)

// VideoQuality represents different video quality levels
type VideoQuality struct {
	Name      string `json:"name"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Bitrate   int    `json:"bitrate"`   // kbps
	Framerate int    `json:"framerate"` // fps
}

// StreamRequest represents a request to start a video stream
type StreamRequest struct {
	StreamID     string       `json:"stream_id"`
	Quality      VideoQuality `json:"quality"`
	BufferSize   int          `json:"buffer_size"`   // seconds
	StartTime    time.Time    `json:"start_time"`
	AdaptiveBitrate bool      `json:"adaptive_bitrate"`
}

// StreamResponse represents a response to a stream request
type StreamResponse struct {
	StreamID      string         `json:"stream_id"`
	Status        string         `json:"status"`
	AvailableQualities []VideoQuality `json:"available_qualities"`
	ChunkDuration time.Duration  `json:"chunk_duration"`
}

// VideoChunk represents a chunk of video data
type VideoChunk struct {
	StreamID     string    `json:"stream_id"`
	SequenceNum  uint64    `json:"sequence_num"`
	Timestamp    time.Time `json:"timestamp"`
	Quality      VideoQuality `json:"quality"`
	Data         []byte    `json:"-"` // Raw video data
	Size         int       `json:"size"`
	IsKeyframe   bool      `json:"is_keyframe"`
	Duration     time.Duration `json:"duration"`
}

// StreamStats represents streaming statistics
type StreamStats struct {
	StreamID        string        `json:"stream_id"`
	BytesSent       uint64        `json:"bytes_sent"`
	ChunksSent      uint64        `json:"chunks_sent"`
	AverageBitrate  float64       `json:"average_bitrate"`
	BufferHealth    float64       `json:"buffer_health"` // 0.0 to 1.0
	LastChunkTime   time.Time     `json:"last_chunk_time"`
	Latency         time.Duration `json:"latency"`
	PacketLoss      float64       `json:"packet_loss"`
}

// ActiveStream represents an active streaming session
type ActiveStream struct {
	ID              string
	Quality         VideoQuality
	StartTime       time.Time
	Stats           StreamStats
	AdaptiveBitrate bool
	Stream          quic.Stream
	Cancel          context.CancelFunc
	mutex           sync.RWMutex
}

// Handler handles video streaming protocol
type Handler struct {
	logger         *zap.Logger
	activeStreams  map[string]*ActiveStream
	videoQualities []VideoQuality
	chunkDuration  time.Duration
	mutex          sync.RWMutex
}

// NewHandler creates a new streaming handler
func NewHandler(logger *zap.Logger) *Handler {
	return &Handler{
		logger:        logger,
		activeStreams: make(map[string]*ActiveStream),
		videoQualities: []VideoQuality{
			{Name: "360p", Width: 640, Height: 360, Bitrate: 500, Framerate: 30},
			{Name: "480p", Width: 854, Height: 480, Bitrate: 1000, Framerate: 30},
			{Name: "720p", Width: 1280, Height: 720, Bitrate: 2000, Framerate: 30},
			{Name: "1080p", Width: 1920, Height: 1080, Bitrate: 4000, Framerate: 30},
		},
		chunkDuration: 2 * time.Second,
	}
}

// HandleStream implements the StreamHandler interface
func (h *Handler) HandleStream(ctx context.Context, stream quic.Stream) error {
	h.logger.Info("Handling streaming stream")

	decoder := json.NewDecoder(stream)
	encoder := json.NewEncoder(stream)

	// Read stream request
	var request StreamRequest
	if err := decoder.Decode(&request); err != nil {
		h.logger.Error("Failed to decode stream request", zap.Error(err))
		return err
	}

	h.logger.Info("Stream request received",
		zap.String("stream_id", request.StreamID),
		zap.String("quality", request.Quality.Name),
		zap.Bool("adaptive", request.AdaptiveBitrate),
	)

	// Validate quality
	quality, err := h.validateQuality(request.Quality)
	if err != nil {
		response := StreamResponse{
			StreamID:           request.StreamID,
			Status:            "error",
			AvailableQualities: h.videoQualities,
		}
		encoder.Encode(response)
		return err
	}

	// Send response
	response := StreamResponse{
		StreamID:           request.StreamID,
		Status:            "accepted",
		AvailableQualities: h.videoQualities,
		ChunkDuration:     h.chunkDuration,
	}

	if err := encoder.Encode(response); err != nil {
		return err
	}

	// Start streaming
	return h.startStream(ctx, request.StreamID, quality, request.AdaptiveBitrate, stream)
}

// validateQuality validates and finds the closest available quality
func (h *Handler) validateQuality(requested VideoQuality) (VideoQuality, error) {
	for _, quality := range h.videoQualities {
		if quality.Name == requested.Name ||
			(quality.Width == requested.Width && quality.Height == requested.Height) {
			return quality, nil
		}
	}

	// Return closest quality by bitrate
	var closest VideoQuality
	minDiff := int(^uint(0) >> 1) // max int

	for _, quality := range h.videoQualities {
		diff := abs(quality.Bitrate - requested.Bitrate)
		if diff < minDiff {
			minDiff = diff
			closest = quality
		}
	}

	if closest.Name == "" {
		return VideoQuality{}, fmt.Errorf("no suitable quality found")
	}

	return closest, nil
}

// startStream starts streaming video data
func (h *Handler) startStream(ctx context.Context, streamID string, quality VideoQuality, adaptive bool, stream quic.Stream) error {
	streamCtx, cancel := context.WithCancel(ctx)

	activeStream := &ActiveStream{
		ID:              streamID,
		Quality:         quality,
		StartTime:       time.Now(),
		AdaptiveBitrate: adaptive,
		Stream:          stream,
		Cancel:          cancel,
		Stats: StreamStats{
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

// streamVideo continuously streams video chunks
func (h *Handler) streamVideo(ctx context.Context, activeStream *ActiveStream) error {
	ticker := time.NewTicker(h.chunkDuration)
	defer ticker.Stop()

	sequenceNum := uint64(0)
	chunkSize := h.calculateChunkSize(activeStream.Quality)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			chunk := h.generateVideoChunk(activeStream.ID, sequenceNum, activeStream.Quality, chunkSize)
			
			if err := h.sendChunk(activeStream.Stream, chunk); err != nil {
				h.logger.Error("Failed to send video chunk",
					zap.String("stream_id", activeStream.ID),
					zap.Uint64("sequence", sequenceNum),
					zap.Error(err),
				)
				return err
			}

			h.updateStreamStats(activeStream, chunk)
			sequenceNum++

			// Adaptive bitrate logic
			if activeStream.AdaptiveBitrate {
				h.adaptBitrate(activeStream)
			}
		}
	}
}

// generateVideoChunk generates a simulated video chunk
func (h *Handler) generateVideoChunk(streamID string, sequenceNum uint64, quality VideoQuality, size int) VideoChunk {
	// Simulate video data (in real implementation, this would be actual encoded video)
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(sequenceNum % 256) // Simple pattern
	}

	return VideoChunk{
		StreamID:    streamID,
		SequenceNum: sequenceNum,
		Timestamp:   time.Now(),
		Quality:     quality,
		Data:        data,
		Size:        size,
		IsKeyframe:  sequenceNum%30 == 0, // Every 30th frame is a keyframe
		Duration:    h.chunkDuration,
	}
}

// sendChunk sends a video chunk over the QUIC stream
func (h *Handler) sendChunk(stream quic.Stream, chunk VideoChunk) error {
	// Send chunk metadata as JSON header
	header, err := json.Marshal(chunk)
	if err != nil {
		return err
	}

	// Send header size (4 bytes)
	headerSize := make([]byte, 4)
	binary.BigEndian.PutUint32(headerSize, uint32(len(header)))
	if _, err := stream.Write(headerSize); err != nil {
		return err
	}

	// Send header
	if _, err := stream.Write(header); err != nil {
		return err
	}

	// Send video data
	if _, err := stream.Write(chunk.Data); err != nil {
		return err
	}

	return nil
}

// calculateChunkSize calculates the size of a video chunk based on quality and duration
func (h *Handler) calculateChunkSize(quality VideoQuality) int {
	// Convert bitrate (kbps) to bytes per chunk duration
	bytesPerSecond := quality.Bitrate * 1000 / 8 // kbps to bytes per second
	return int(float64(bytesPerSecond) * h.chunkDuration.Seconds())
}

// updateStreamStats updates streaming statistics
func (h *Handler) updateStreamStats(activeStream *ActiveStream, chunk VideoChunk) {
	activeStream.mutex.Lock()
	defer activeStream.mutex.Unlock()

	activeStream.Stats.BytesSent += uint64(chunk.Size)
	activeStream.Stats.ChunksSent++
	activeStream.Stats.LastChunkTime = time.Now()

	// Calculate average bitrate
	duration := time.Since(activeStream.StartTime).Seconds()
	if duration > 0 {
		activeStream.Stats.AverageBitrate = float64(activeStream.Stats.BytesSent*8) / duration / 1000 // kbps
	}

	// Simulate buffer health (in real implementation, this would be based on client feedback)
	activeStream.Stats.BufferHealth = 0.8 + 0.2*float64(time.Now().UnixNano()%100)/100

	// Simulate packet loss (in real implementation, this would be measured)
	activeStream.Stats.PacketLoss = float64(time.Now().UnixNano()%100) / 10000 // 0-1%
}

// adaptBitrate adapts the video quality based on network conditions
func (h *Handler) adaptBitrate(activeStream *ActiveStream) {
	activeStream.mutex.RLock()
	bufferHealth := activeStream.Stats.BufferHealth
	packetLoss := activeStream.Stats.PacketLoss
	activeStream.mutex.RUnlock()

	currentQuality := activeStream.Quality
	var newQuality VideoQuality

	// Simple adaptive logic
	if bufferHealth < 0.3 || packetLoss > 0.05 {
		// Decrease quality
		newQuality = h.getDowngradedQuality(currentQuality)
	} else if bufferHealth > 0.8 && packetLoss < 0.01 {
		// Increase quality
		newQuality = h.getUpgradedQuality(currentQuality)
	} else {
		return // No change needed
	}

	if newQuality.Name != currentQuality.Name {
		activeStream.Quality = newQuality
		h.logger.Info("Adapted video quality",
			zap.String("stream_id", activeStream.ID),
			zap.String("from", currentQuality.Name),
			zap.String("to", newQuality.Name),
			zap.Float64("buffer_health", bufferHealth),
			zap.Float64("packet_loss", packetLoss),
		)
	}
}

// getDowngradedQuality returns the next lower quality
func (h *Handler) getDowngradedQuality(current VideoQuality) VideoQuality {
	for i, quality := range h.videoQualities {
		if quality.Name == current.Name && i > 0 {
			return h.videoQualities[i-1]
		}
	}
	return current
}

// getUpgradedQuality returns the next higher quality
func (h *Handler) getUpgradedQuality(current VideoQuality) VideoQuality {
	for i, quality := range h.videoQualities {
		if quality.Name == current.Name && i < len(h.videoQualities)-1 {
			return h.videoQualities[i+1]
		}
	}
	return current
}

// GetActiveStreams returns information about all active streams
func (h *Handler) GetActiveStreams() map[string]StreamStats {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	stats := make(map[string]StreamStats)
	for id, stream := range h.activeStreams {
		stream.mutex.RLock()
		stats[id] = stream.Stats
		stream.mutex.RUnlock()
	}
	return stats
}

// StopStream stops a specific stream
func (h *Handler) StopStream(streamID string) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if stream, exists := h.activeStreams[streamID]; exists {
		stream.Cancel()
		delete(h.activeStreams, streamID)
		h.logger.Info("Stream stopped", zap.String("stream_id", streamID))
		return nil
	}

	return fmt.Errorf("stream not found: %s", streamID)
}

// Helper function
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}