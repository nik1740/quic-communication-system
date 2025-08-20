package streaming

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/sirupsen/logrus"
)

// QualityLevel represents different video quality levels
type QualityLevel struct {
	Name       string `json:"name"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	Bitrate    int    `json:"bitrate"`    // kbps
	Framerate  int    `json:"framerate"` // fps
	SegmentURL string `json:"segment_url"`
}

// StreamManifest represents video stream information
type StreamManifest struct {
	StreamID     string         `json:"stream_id"`
	Title        string         `json:"title"`
	Duration     float64        `json:"duration"`
	Qualities    []QualityLevel `json:"qualities"`
	SegmentCount int            `json:"segment_count"`
}

// StreamSession represents an active streaming session
type StreamSession struct {
	ID            string    `json:"id"`
	ClientAddr    string    `json:"client_addr"`
	StreamID      string    `json:"stream_id"`
	QualityLevel  string    `json:"quality_level"`
	StartTime     time.Time `json:"start_time"`
	BytesSent     int64     `json:"bytes_sent"`
	FramesSent    int       `json:"frames_sent"`
	LastFrameTime time.Time `json:"last_frame_time"`
}

// Handler implements HTTP/3 based video streaming
type Handler struct {
	logger      *logrus.Logger
	streams     map[string]*StreamManifest
	sessions    map[string]*StreamSession
	sessionsMux sync.RWMutex
	server      *http3.Server
}

// NewHandler creates a new streaming handler
func NewHandler(logger *logrus.Logger) *Handler {
	handler := &Handler{
		logger:   logger,
		streams:  make(map[string]*StreamManifest),
		sessions: make(map[string]*StreamSession),
	}

	// Initialize with sample streams
	handler.initializeSampleStreams()

	return handler
}

// initializeSampleStreams creates sample video streams for testing
func (h *Handler) initializeSampleStreams() {
	// Sample stream 1: Live camera feed
	h.streams["live-cam-1"] = &StreamManifest{
		StreamID:     "live-cam-1",
		Title:        "Security Camera 1",
		Duration:     0, // Live stream
		SegmentCount: 0, // Live stream
		Qualities: []QualityLevel{
			{Name: "1080p", Width: 1920, Height: 1080, Bitrate: 5000, Framerate: 30, SegmentURL: "/stream/live-cam-1/1080p"},
			{Name: "720p", Width: 1280, Height: 720, Bitrate: 2500, Framerate: 30, SegmentURL: "/stream/live-cam-1/720p"},
			{Name: "480p", Width: 854, Height: 480, Bitrate: 1200, Framerate: 25, SegmentURL: "/stream/live-cam-1/480p"},
			{Name: "360p", Width: 640, Height: 360, Bitrate: 600, Framerate: 25, SegmentURL: "/stream/live-cam-1/360p"},
		},
	}

	// Sample stream 2: Recorded video
	h.streams["video-1"] = &StreamManifest{
		StreamID:     "video-1",
		Title:        "Sample Video Content",
		Duration:     120.5, // 2 minutes
		SegmentCount: 24,    // 5-second segments
		Qualities: []QualityLevel{
			{Name: "4K", Width: 3840, Height: 2160, Bitrate: 15000, Framerate: 60, SegmentURL: "/stream/video-1/4k"},
			{Name: "1080p", Width: 1920, Height: 1080, Bitrate: 5000, Framerate: 30, SegmentURL: "/stream/video-1/1080p"},
			{Name: "720p", Width: 1280, Height: 720, Bitrate: 2500, Framerate: 30, SegmentURL: "/stream/video-1/720p"},
			{Name: "480p", Width: 854, Height: 480, Bitrate: 1200, Framerate: 25, SegmentURL: "/stream/video-1/480p"},
		},
	}
}

// SetupHTTP3Routes sets up HTTP/3 routes for video streaming
func (h *Handler) SetupHTTP3Routes() http.Handler {
	mux := http.NewServeMux()

	// Stream discovery endpoints
	mux.HandleFunc("/api/streams", h.handleStreamList)
	mux.HandleFunc("/api/streams/", h.handleStreamInfo)

	// Video streaming endpoints
	mux.HandleFunc("/stream/", h.handleVideoStream)

	// Session management
	mux.HandleFunc("/api/sessions", h.handleSessions)

	// Health check
	mux.HandleFunc("/health", h.handleHealth)

	return mux
}

// handleStreamList returns available streams
func (h *Handler) handleStreamList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	streams := make([]StreamManifest, 0, len(h.streams))
	for _, stream := range h.streams {
		streams = append(streams, *stream)
	}

	if err := json.NewEncoder(w).Encode(streams); err != nil {
		h.logger.WithError(err).Error("Failed to encode streams list")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleStreamInfo returns information about a specific stream
func (h *Handler) handleStreamInfo(w http.ResponseWriter, r *http.Request) {
	streamID := r.URL.Path[len("/api/streams/"):]
	if streamID == "" {
		http.Error(w, "Stream ID required", http.StatusBadRequest)
		return
	}

	stream, exists := h.streams[streamID]
	if !exists {
		http.Error(w, "Stream not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if err := json.NewEncoder(w).Encode(stream); err != nil {
		h.logger.WithError(err).Error("Failed to encode stream info")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleVideoStream handles video streaming requests
func (h *Handler) handleVideoStream(w http.ResponseWriter, r *http.Request) {
	// Parse URL: /stream/streamID/quality/segment
	urlParts := r.URL.Path[len("/stream/"):]
	h.logger.Infof("Streaming request: %s", urlParts)

	// Create or update session
	sessionID := h.createOrUpdateSession(r, urlParts)

	// Set streaming headers
	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// For demo purposes, generate simulated video data
	h.streamSimulatedVideo(w, sessionID)
}

// createOrUpdateSession creates or updates a streaming session
func (h *Handler) createOrUpdateSession(r *http.Request, urlParts string) string {
	h.sessionsMux.Lock()
	defer h.sessionsMux.Unlock()

	sessionID := fmt.Sprintf("session_%d", time.Now().UnixNano())
	session := &StreamSession{
		ID:            sessionID,
		ClientAddr:    r.RemoteAddr,
		StreamID:      urlParts, // Simplified for demo
		QualityLevel:  "720p",   // Default quality
		StartTime:     time.Now(),
		LastFrameTime: time.Now(),
	}

	h.sessions[sessionID] = session
	h.logger.Infof("Created streaming session %s for %s", sessionID, r.RemoteAddr)

	return sessionID
}

// streamSimulatedVideo generates simulated video data for demonstration
func (h *Handler) streamSimulatedVideo(w http.ResponseWriter, sessionID string) {
	session := h.sessions[sessionID]
	if session == nil {
		return
	}

	// Simulate video streaming with timed data chunks
	flusher, ok := w.(http.Flusher)
	if !ok {
		h.logger.Error("Streaming not supported")
		return
	}

	// Simulate 30 FPS video stream
	ticker := time.NewTicker(33 * time.Millisecond) // ~30 FPS
	defer ticker.Stop()

	frameCount := 0
	for range ticker.C {
		// Generate simulated frame data (normally would be actual video data)
		frameData := h.generateSimulatedFrame(frameCount, session.QualityLevel)

		// Write frame data
		n, err := w.Write(frameData)
		if err != nil {
			h.logger.WithError(err).Error("Failed to write frame data")
			break
		}

		// Update session statistics
		session.BytesSent += int64(n)
		session.FramesSent++
		session.LastFrameTime = time.Now()

		// Flush to ensure immediate transmission
		flusher.Flush()

		frameCount++

		// Stop after 10 seconds for demo
		if frameCount >= 300 {
			break
		}
	}

	h.logger.Infof("Streaming session %s completed: %d frames, %d bytes", 
		sessionID, session.FramesSent, session.BytesSent)
}

// generateSimulatedFrame generates simulated video frame data
func (h *Handler) generateSimulatedFrame(frameNumber int, quality string) []byte {
	// Simulate different frame sizes based on quality
	var frameSize int
	switch quality {
	case "4K":
		frameSize = 50000 // 50KB per frame
	case "1080p":
		frameSize = 20000 // 20KB per frame
	case "720p":
		frameSize = 10000 // 10KB per frame
	case "480p":
		frameSize = 5000 // 5KB per frame
	case "360p":
		frameSize = 3000 // 3KB per frame
	default:
		frameSize = 10000
	}

	// Generate frame header
	header := fmt.Sprintf("FRAME_%06d_%s\n", frameNumber, quality)
	
	// Fill with simulated data
	data := make([]byte, frameSize)
	copy(data, []byte(header))
	
	// Fill remaining with pattern for demo
	for i := len(header); i < frameSize; i++ {
		data[i] = byte(i % 256)
	}

	return data
}

// handleSessions returns active streaming sessions
func (h *Handler) handleSessions(w http.ResponseWriter, r *http.Request) {
	h.sessionsMux.RLock()
	defer h.sessionsMux.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	sessions := make([]StreamSession, 0, len(h.sessions))
	for _, session := range h.sessions {
		sessions = append(sessions, *session)
	}

	if err := json.NewEncoder(w).Encode(sessions); err != nil {
		h.logger.WithError(err).Error("Failed to encode sessions")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleHealth returns server health status
func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	health := map[string]interface{}{
		"status":          "healthy",
		"timestamp":       time.Now(),
		"active_streams":  len(h.streams),
		"active_sessions": len(h.sessions),
	}

	json.NewEncoder(w).Encode(health)
}

// HandleStream implements the QUIC StreamHandler interface
func (h *Handler) HandleStream(ctx context.Context, stream *quic.Stream) error {
	h.logger.Info("Handling streaming QUIC stream")

	// Read request data
	buffer := make([]byte, 4096)
	n, err := stream.Read(buffer)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read stream data: %w", err)
	}

	request := string(buffer[:n])
	h.logger.Infof("Received streaming request: %s", request)

	// Parse request and generate response
	response := h.processStreamingRequest(request)

	// Send response
	_, err = stream.Write([]byte(response))
	return err
}

// processStreamingRequest processes streaming requests from QUIC streams
func (h *Handler) processStreamingRequest(request string) string {
	// Simple request processing for demo
	if request == "list_streams" {
		streams := make([]string, 0, len(h.streams))
		for id := range h.streams {
			streams = append(streams, id)
		}
		
		response, _ := json.Marshal(map[string]interface{}{
			"streams": streams,
			"count":   len(streams),
		})
		return string(response)
	}

	return `{"error": "unknown request"}`
}

// GetActiveSessionsCount returns the number of active sessions
func (h *Handler) GetActiveSessionsCount() int {
	h.sessionsMux.RLock()
	defer h.sessionsMux.RUnlock()
	return len(h.sessions)
}

// CleanupOldSessions removes sessions that haven't been active
func (h *Handler) CleanupOldSessions(maxAge time.Duration) {
	h.sessionsMux.Lock()
	defer h.sessionsMux.Unlock()

	now := time.Now()
	for id, session := range h.sessions {
		if now.Sub(session.LastFrameTime) > maxAge {
			delete(h.sessions, id)
			h.logger.Infof("Cleaned up old session %s", id)
		}
	}
}