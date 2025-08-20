package streaming

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Quality represents video quality settings
type Quality struct {
	Name       string `json:"name"`
	Resolution string `json:"resolution"`
	Bitrate    int    `json:"bitrate"`
	FPS        int    `json:"fps"`
}

// StreamInfo contains stream metadata
type StreamInfo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Duration  int       `json:"duration"`
	Qualities []Quality `json:"qualities"`
	CreatedAt time.Time `json:"created_at"`
}

// Handler handles video streaming requests
type Handler struct {
	videoDir string
	logger   *logrus.Logger
	streams  map[string]*StreamInfo
}

// NewHandler creates a new streaming handler
func NewHandler(videoDir string, logger *logrus.Logger) *Handler {
	return &Handler{
		videoDir: videoDir,
		logger:   logger,
		streams:  make(map[string]*StreamInfo),
	}
}

// RegisterRoutes registers streaming routes
func (h *Handler) RegisterRoutes(registerFunc func(string, http.HandlerFunc)) {
	registerFunc("/stream/list", h.handleStreamList)
	registerFunc("/stream/info/", h.handleStreamInfo)
	registerFunc("/stream/video/", h.handleVideo)
	registerFunc("/stream/manifest/", h.handleManifest)
	registerFunc("/stream/health", h.handleHealth)
	
	// Initialize with demo streams
	h.initializeDemoStreams()
}

// initializeDemoStreams creates demo stream info
func (h *Handler) initializeDemoStreams() {
	demoQualities := []Quality{
		{Name: "480p", Resolution: "854x480", Bitrate: 1000, FPS: 30},
		{Name: "720p", Resolution: "1280x720", Bitrate: 2500, FPS: 30},
		{Name: "1080p", Resolution: "1920x1080", Bitrate: 5000, FPS: 30},
	}
	
	h.streams["demo1"] = &StreamInfo{
		ID:        "demo1",
		Name:      "Demo Stream 1",
		Duration:  300, // 5 minutes
		Qualities: demoQualities,
		CreatedAt: time.Now(),
	}
	
	h.streams["demo2"] = &StreamInfo{
		ID:        "demo2",
		Name:      "Demo Stream 2",
		Duration:  600, // 10 minutes
		Qualities: demoQualities,
		CreatedAt: time.Now(),
	}
}

// handleStreamList returns available streams
func (h *Handler) handleStreamList(w http.ResponseWriter, r *http.Request) {
	h.logger.WithFields(logrus.Fields{
		"method":    r.Method,
		"path":      r.URL.Path,
		"component": "streaming-handler",
	}).Info("Handling stream list request")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	
	streams := make([]*StreamInfo, 0, len(h.streams))
	for _, stream := range h.streams {
		streams = append(streams, stream)
	}
	
	response := map[string]interface{}{
		"streams":   streams,
		"count":     len(streams),
		"timestamp": time.Now(),
	}
	
	json.NewEncoder(w).Encode(response)
}

// handleStreamInfo returns stream information
func (h *Handler) handleStreamInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract stream ID from path
	path := strings.TrimPrefix(r.URL.Path, "/stream/info/")
	streamID := strings.Split(path, "/")[0]
	
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
	json.NewEncoder(w).Encode(stream)
}

// handleVideo serves video content
func (h *Handler) handleVideo(w http.ResponseWriter, r *http.Request) {
	h.logger.WithFields(logrus.Fields{
		"method":    r.Method,
		"path":      r.URL.Path,
		"component": "streaming-handler",
	}).Info("Handling video request")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse path: /stream/video/{streamID}/{quality}/{segment}
	path := strings.TrimPrefix(r.URL.Path, "/stream/video/")
	parts := strings.Split(path, "/")
	
	if len(parts) < 3 {
		http.Error(w, "Invalid video path", http.StatusBadRequest)
		return
	}
	
	streamID := parts[0]
	quality := parts[1]
	segment := parts[2]
	
	// Check if stream exists
	stream, exists := h.streams[streamID]
	if !exists {
		http.Error(w, "Stream not found", http.StatusNotFound)
		return
	}
	
	// Validate quality
	validQuality := false
	for _, q := range stream.Qualities {
		if q.Name == quality {
			validQuality = true
			break
		}
	}
	
	if !validQuality {
		http.Error(w, "Quality not available", http.StatusBadRequest)
		return
	}
	
	// For demo purposes, generate synthetic video data
	h.serveVideoSegment(w, r, streamID, quality, segment)
}

// serveVideoSegment serves a video segment
func (h *Handler) serveVideoSegment(w http.ResponseWriter, r *http.Request, streamID, quality, segment string) {
	// In a real implementation, this would serve actual video files
	// For demo, we'll serve synthetic data
	
	segmentNum, err := strconv.Atoi(strings.TrimSuffix(segment, ".ts"))
	if err != nil {
		http.Error(w, "Invalid segment number", http.StatusBadRequest)
		return
	}
	
	// Set appropriate headers for video streaming
	w.Header().Set("Content-Type", "video/mp2t") // MPEG-TS for HLS
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	// Generate synthetic video data (in practice, read from file)
	dataSize := 1024 * 500 // 500KB per segment
	switch quality {
	case "1080p":
		dataSize = 1024 * 1000 // 1MB
	case "720p":
		dataSize = 1024 * 750  // 750KB
	case "480p":
		dataSize = 1024 * 500  // 500KB
	}
	
	data := make([]byte, dataSize)
	// Fill with pseudo-random data to simulate video content
	for i := range data {
		data[i] = byte((segmentNum*i + int(time.Now().UnixNano())) % 256)
	}
	
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Write(data)
	
	h.logger.WithFields(logrus.Fields{
		"stream_id": streamID,
		"quality":   quality,
		"segment":   segment,
		"size":      len(data),
		"component": "streaming-handler",
	}).Debug("Served video segment")
}

// handleManifest serves HLS manifest files
func (h *Handler) handleManifest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse path: /stream/manifest/{streamID}/{quality}.m3u8
	path := strings.TrimPrefix(r.URL.Path, "/stream/manifest/")
	parts := strings.Split(path, "/")
	
	if len(parts) < 2 {
		http.Error(w, "Invalid manifest path", http.StatusBadRequest)
		return
	}
	
	streamID := parts[0]
	manifestFile := parts[1]
	
	stream, exists := h.streams[streamID]
	if !exists {
		http.Error(w, "Stream not found", http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	if manifestFile == "master.m3u8" {
		h.serveMasterPlaylist(w, stream)
	} else {
		// Extract quality from filename
		quality := strings.TrimSuffix(manifestFile, ".m3u8")
		h.serveMediaPlaylist(w, stream, quality)
	}
}

// serveMasterPlaylist serves the master HLS playlist
func (h *Handler) serveMasterPlaylist(w http.ResponseWriter, stream *StreamInfo) {
	manifest := "#EXTM3U\n#EXT-X-VERSION:3\n\n"
	
	for _, quality := range stream.Qualities {
		manifest += fmt.Sprintf("#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%s\n",
			quality.Bitrate*1000, quality.Resolution)
		manifest += fmt.Sprintf("%s.m3u8\n\n", quality.Name)
	}
	
	w.Write([]byte(manifest))
}

// serveMediaPlaylist serves a media playlist for a specific quality
func (h *Handler) serveMediaPlaylist(w http.ResponseWriter, stream *StreamInfo, quality string) {
	manifest := "#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-TARGETDURATION:10\n#EXT-X-MEDIA-SEQUENCE:0\n\n"
	
	// Calculate number of segments (10 seconds each)
	segments := stream.Duration / 10
	
	for i := 0; i < segments; i++ {
		manifest += "#EXTINF:10.0,\n"
		manifest += fmt.Sprintf("../video/%s/%s/%d.ts\n", stream.ID, quality, i)
	}
	
	manifest += "#EXT-X-ENDLIST\n"
	w.Write([]byte(manifest))
}

// handleHealth handles health check requests
func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	
	response := map[string]interface{}{
		"status":         "healthy",
		"timestamp":      time.Now(),
		"available_streams": len(h.streams),
		"video_dir":      h.videoDir,
		"component":      "streaming-service",
	}
	
	json.NewEncoder(w).Encode(response)
}