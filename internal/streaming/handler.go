package streaming

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

// StreamInfo represents video stream metadata
type StreamInfo struct {
	StreamID    string    `json:"stream_id"`
	Title       string    `json:"title"`
	Duration    int       `json:"duration"` // seconds
	Bitrates    []Bitrate `json:"bitrates"`
	Format      string    `json:"format"`
	Resolution  string    `json:"resolution"`
	FrameRate   int       `json:"frame_rate"`
	CreatedAt   time.Time `json:"created_at"`
}

// Bitrate represents different quality levels
type Bitrate struct {
	Quality    string `json:"quality"`    // "low", "medium", "high", "ultra"
	Bitrate    int    `json:"bitrate"`    // kbps
	Resolution string `json:"resolution"` // e.g., "1920x1080"
	URL        string `json:"url"`
}

// StreamChunk represents a video chunk
type StreamChunk struct {
	StreamID    string `json:"stream_id"`
	ChunkIndex  int    `json:"chunk_index"`
	Quality     string `json:"quality"`
	Data        []byte `json:"data,omitempty"`
	Size        int    `json:"size"`
	Duration    int    `json:"duration"` // milliseconds
	Timestamp   int64  `json:"timestamp"`
	IsKeyFrame  bool   `json:"is_keyframe"`
}

// StreamStats represents streaming statistics
type StreamStats struct {
	StreamID      string  `json:"stream_id"`
	BytesSent     int64   `json:"bytes_sent"`
	ChunksSent    int     `json:"chunks_sent"`
	Latency       float64 `json:"latency_ms"`
	Bandwidth     float64 `json:"bandwidth_mbps"`
	PacketLoss    float64 `json:"packet_loss_percent"`
	ActiveClients int     `json:"active_clients"`
	Uptime        int64   `json:"uptime_seconds"`
}

// Handler handles video streaming HTTP/3 requests
func Handler(w http.ResponseWriter, r *http.Request) {
	// Parse the URL path
	path := strings.TrimPrefix(r.URL.Path, "/stream/")
	parts := strings.Split(path, "/")
	
	if len(parts) == 0 {
		http.Error(w, "Invalid streaming endpoint", http.StatusBadRequest)
		return
	}

	switch parts[0] {
	case "list":
		handleStreamList(w, r)
	case "info":
		if len(parts) < 2 {
			http.Error(w, "Stream ID required", http.StatusBadRequest)
			return
		}
		handleStreamInfo(w, r, parts[1])
	case "chunk":
		if len(parts) < 2 {
			http.Error(w, "Stream ID required", http.StatusBadRequest)
			return
		}
		handleStreamChunk(w, r, parts[1])
	case "stats":
		if len(parts) < 2 {
			http.Error(w, "Stream ID required", http.StatusBadRequest)
			return
		}
		handleStreamStats(w, r, parts[1])
	case "live":
		handleLiveStream(w, r)
	default:
		http.Error(w, "Unknown streaming endpoint", http.StatusNotFound)
	}
}

func handleStreamList(w http.ResponseWriter, r *http.Request) {
	streams := []StreamInfo{
		{
			StreamID: "stream_001",
			Title:    "Sample Video 1",
			Duration: 120,
			Bitrates: []Bitrate{
				{"low", 500, "640x360", "/stream/chunk/stream_001?quality=low"},
				{"medium", 1500, "1280x720", "/stream/chunk/stream_001?quality=medium"},
				{"high", 3000, "1920x1080", "/stream/chunk/stream_001?quality=high"},
				{"ultra", 6000, "3840x2160", "/stream/chunk/stream_001?quality=ultra"},
			},
			Format:    "h264",
			Resolution: "1920x1080",
			FrameRate: 30,
			CreatedAt: time.Now().Add(-time.Hour),
		},
		{
			StreamID: "stream_002",
			Title:    "Live Camera Feed",
			Duration: -1, // Live stream
			Bitrates: []Bitrate{
				{"low", 300, "480x270", "/stream/chunk/stream_002?quality=low"},
				{"medium", 800, "854x480", "/stream/chunk/stream_002?quality=medium"},
				{"high", 1500, "1280x720", "/stream/chunk/stream_002?quality=high"},
			},
			Format:    "h264",
			Resolution: "1280x720",
			FrameRate: 25,
			CreatedAt: time.Now().Add(-10 * time.Minute),
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"streams": streams,
		"count":   len(streams),
	})
}

func handleStreamInfo(w http.ResponseWriter, r *http.Request, streamID string) {
	// Simulate stream info retrieval
	stream := StreamInfo{
		StreamID: streamID,
		Title:    fmt.Sprintf("Stream %s", streamID),
		Duration: 300,
		Bitrates: []Bitrate{
			{"low", 500, "640x360", fmt.Sprintf("/stream/chunk/%s?quality=low", streamID)},
			{"medium", 1500, "1280x720", fmt.Sprintf("/stream/chunk/%s?quality=medium", streamID)},
			{"high", 3000, "1920x1080", fmt.Sprintf("/stream/chunk/%s?quality=high", streamID)},
		},
		Format:    "h264",
		Resolution: "1920x1080",
		FrameRate: 30,
		CreatedAt: time.Now().Add(-time.Hour),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stream)
}

func handleStreamChunk(w http.ResponseWriter, r *http.Request, streamID string) {
	quality := r.URL.Query().Get("quality")
	if quality == "" {
		quality = "medium"
	}
	
	chunkIndex := 0
	if idx := r.URL.Query().Get("chunk"); idx != "" {
		if i, err := strconv.Atoi(idx); err == nil {
			chunkIndex = i
		}
	}
	
	// Simulate video chunk generation
	chunkSize := getChunkSize(quality)
	chunk := StreamChunk{
		StreamID:   streamID,
		ChunkIndex: chunkIndex,
		Quality:    quality,
		Data:       generateVideoData(chunkSize),
		Size:       chunkSize,
		Duration:   2000, // 2 seconds
		Timestamp:  time.Now().UnixMilli(),
		IsKeyFrame: chunkIndex%10 == 0, // Every 10th chunk is a keyframe
	}
	
	// Set appropriate headers for video streaming
	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Stream-ID", streamID)
	w.Header().Set("X-Chunk-Index", strconv.Itoa(chunkIndex))
	w.Header().Set("X-Quality", quality)
	
	// For JSON response (metadata)
	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		chunk.Data = nil // Don't include binary data in JSON
		json.NewEncoder(w).Encode(chunk)
		return
	}
	
	// Return binary video data
	w.Write(chunk.Data)
	
	log.Printf("Served chunk %d for stream %s (quality: %s, size: %d bytes)", 
		chunkIndex, streamID, quality, chunkSize)
}

func handleStreamStats(w http.ResponseWriter, r *http.Request, streamID string) {
	stats := StreamStats{
		StreamID:      streamID,
		BytesSent:     int64(rand.Intn(1000000000)), // Random bytes sent
		ChunksSent:    rand.Intn(10000),
		Latency:       float64(rand.Intn(100)) + rand.Float64(),
		Bandwidth:     float64(rand.Intn(50)) + rand.Float64(),
		PacketLoss:    rand.Float64() * 5, // 0-5% packet loss
		ActiveClients: rand.Intn(100),
		Uptime:        int64(rand.Intn(86400)), // Up to 24 hours
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func handleLiveStream(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers for live streaming
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	// Simulate live stream events
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	
	for i := 0; i < 30; i++ { // Stream for 30 seconds
		select {
		case <-ticker.C:
			event := map[string]interface{}{
				"type":      "frame",
				"timestamp": time.Now().UnixMilli(),
				"frame_id":  i,
				"size":      rand.Intn(50000) + 10000,
				"quality":   []string{"low", "medium", "high"}[rand.Intn(3)],
			}
			
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "data: %s\n\n", data)
			
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			
		case <-r.Context().Done():
			return
		}
	}
}

func getChunkSize(quality string) int {
	switch quality {
	case "low":
		return 50000 + rand.Intn(20000) // 50-70KB
	case "medium":
		return 150000 + rand.Intn(50000) // 150-200KB
	case "high":
		return 400000 + rand.Intn(100000) // 400-500KB
	case "ultra":
		return 800000 + rand.Intn(200000) // 800KB-1MB
	default:
		return 150000
	}
}

func generateVideoData(size int) []byte {
	// Generate simulated video data
	data := make([]byte, size)
	rand.Read(data)
	return data
}