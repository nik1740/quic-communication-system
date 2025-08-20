package main

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/nik1740/quic-communication-system/internal/streaming"
	"go.uber.org/zap"
)

func main() {
	var (
		serverAddr = flag.String("server", "localhost:4433", "Server address")
		streamID   = flag.String("stream-id", "test-stream", "Stream ID")
		quality    = flag.String("quality", "720p", "Video quality (360p, 480p, 720p, 1080p)")
		adaptive   = flag.Bool("adaptive", true, "Enable adaptive bitrate")
		duration   = flag.Duration("duration", 60*time.Second, "Streaming duration")
		bufferSize = flag.Int("buffer", 30, "Buffer size in seconds")
	)
	flag.Parse()

	// Initialize logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting streaming client",
		zap.String("stream_id", *streamID),
		zap.String("quality", *quality),
		zap.String("server", *serverAddr),
		zap.Bool("adaptive", *adaptive),
		zap.Duration("duration", *duration),
	)

	// Create TLS config (insecure for testing)
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic-communication-system"},
	}

	// Connect to server
	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()

	conn, err := quic.DialAddr(ctx, *serverAddr, tlsConfig, &quic.Config{})
	if err != nil {
		logger.Fatal("Failed to connect to server", zap.Error(err))
	}
	defer conn.CloseWithError(0, "")

	logger.Info("Connected to server")

	// Open stream for video streaming
	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		logger.Fatal("Failed to open stream", zap.Error(err))
	}
	defer stream.Close()

	// Send protocol header
	if _, err := stream.Write([]byte(streaming.ProtocolStreaming)); err != nil {
		logger.Fatal("Failed to send protocol header", zap.Error(err))
	}

	encoder := json.NewEncoder(stream)

	// Map quality string to VideoQuality struct
	qualityMap := map[string]streaming.VideoQuality{
		"360p":  {Name: "360p", Width: 640, Height: 360, Bitrate: 500, Framerate: 30},
		"480p":  {Name: "480p", Width: 854, Height: 480, Bitrate: 1000, Framerate: 30},
		"720p":  {Name: "720p", Width: 1280, Height: 720, Bitrate: 2000, Framerate: 30},
		"1080p": {Name: "1080p", Width: 1920, Height: 1080, Bitrate: 4000, Framerate: 30},
	}

	videoQuality, exists := qualityMap[*quality]
	if !exists {
		logger.Fatal("Invalid quality specified", zap.String("quality", *quality))
	}

	// Send stream request
	request := streaming.StreamRequest{
		StreamID:        *streamID,
		Quality:         videoQuality,
		BufferSize:      *bufferSize,
		StartTime:       time.Now(),
		AdaptiveBitrate: *adaptive,
	}

	if err := encoder.Encode(request); err != nil {
		logger.Fatal("Failed to send stream request", zap.Error(err))
	}

	logger.Info("Sent stream request")

	// Read response
	decoder := json.NewDecoder(stream)
	var response streaming.StreamResponse
	if err := decoder.Decode(&response); err != nil {
		logger.Fatal("Failed to read stream response", zap.Error(err))
	}

	if response.Status != "accepted" {
		logger.Fatal("Stream request rejected",
			zap.String("status", response.Status),
			zap.Any("available_qualities", response.AvailableQualities),
		)
	}

	logger.Info("Stream accepted",
		zap.String("stream_id", response.StreamID),
		zap.Duration("chunk_duration", response.ChunkDuration),
	)

	// Start receiving video chunks
	stats := &StreamingStats{
		StartTime: time.Now(),
	}

	go logStats(stats, logger)

	for {
		select {
		case <-ctx.Done():
			logger.Info("Streaming finished")
			logFinalStats(stats, logger)
			return
		default:
			if err := receiveChunk(stream, stats, logger); err != nil {
				logger.Error("Failed to receive chunk", zap.Error(err))
				return
			}
		}
	}
}

// StreamingStats tracks client-side streaming statistics
type StreamingStats struct {
	StartTime     time.Time
	ChunksReceived uint64
	BytesReceived  uint64
	LastChunkTime  time.Time
	TotalLatency   time.Duration
	MaxLatency     time.Duration
	MinLatency     time.Duration
	PacketsLost    uint64
	LastSequence   uint64
}

// receiveChunk receives and processes a video chunk
func receiveChunk(stream quic.Stream, stats *StreamingStats, logger *zap.Logger) error {
	// Read header size (4 bytes)
	headerSizeBytes := make([]byte, 4)
	if _, err := stream.Read(headerSizeBytes); err != nil {
		return err
	}
	headerSize := binary.BigEndian.Uint32(headerSizeBytes)

	// Read header
	headerBytes := make([]byte, headerSize)
	if _, err := stream.Read(headerBytes); err != nil {
		return err
	}

	// Parse chunk metadata
	var chunk streaming.VideoChunk
	if err := json.Unmarshal(headerBytes, &chunk); err != nil {
		return err
	}

	// Read video data
	videoData := make([]byte, chunk.Size)
	if _, err := stream.Read(videoData); err != nil {
		return err
	}

	// Update statistics
	now := time.Now()
	latency := now.Sub(chunk.Timestamp)
	
	stats.ChunksReceived++
	stats.BytesReceived += uint64(chunk.Size)
	stats.LastChunkTime = now
	stats.TotalLatency += latency

	if stats.ChunksReceived == 1 {
		stats.MinLatency = latency
		stats.MaxLatency = latency
	} else {
		if latency < stats.MinLatency {
			stats.MinLatency = latency
		}
		if latency > stats.MaxLatency {
			stats.MaxLatency = latency
		}
	}

	// Check for packet loss (gaps in sequence numbers)
	if stats.LastSequence > 0 && chunk.SequenceNum > stats.LastSequence+1 {
		stats.PacketsLost += chunk.SequenceNum - stats.LastSequence - 1
		logger.Warn("Detected packet loss",
			zap.Uint64("expected", stats.LastSequence+1),
			zap.Uint64("received", chunk.SequenceNum),
			zap.Uint64("lost", chunk.SequenceNum-stats.LastSequence-1),
		)
	}
	stats.LastSequence = chunk.SequenceNum

	logger.Debug("Received video chunk",
		zap.String("stream_id", chunk.StreamID),
		zap.Uint64("sequence", chunk.SequenceNum),
		zap.Int("size", chunk.Size),
		zap.String("quality", chunk.Quality.Name),
		zap.Bool("keyframe", chunk.IsKeyframe),
		zap.Duration("latency", latency),
	)

	return nil
}

// logStats periodically logs streaming statistics
func logStats(stats *StreamingStats, logger *zap.Logger) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		duration := time.Since(stats.StartTime).Seconds()
		if duration == 0 {
			continue
		}

		avgBitrate := float64(stats.BytesReceived*8) / duration / 1000 // kbps
		avgLatency := time.Duration(0)
		if stats.ChunksReceived > 0 {
			avgLatency = stats.TotalLatency / time.Duration(stats.ChunksReceived)
		}

		logger.Info("Streaming statistics",
			zap.Uint64("chunks_received", stats.ChunksReceived),
			zap.Uint64("bytes_received", stats.BytesReceived),
			zap.Float64("avg_bitrate_kbps", avgBitrate),
			zap.Duration("avg_latency", avgLatency),
			zap.Duration("min_latency", stats.MinLatency),
			zap.Duration("max_latency", stats.MaxLatency),
			zap.Uint64("packets_lost", stats.PacketsLost),
		)
	}
}

// logFinalStats logs final streaming statistics
func logFinalStats(stats *StreamingStats, logger *zap.Logger) {
	duration := time.Since(stats.StartTime).Seconds()
	avgBitrate := float64(stats.BytesReceived*8) / duration / 1000 // kbps
	avgLatency := time.Duration(0)
	if stats.ChunksReceived > 0 {
		avgLatency = stats.TotalLatency / time.Duration(stats.ChunksReceived)
	}

	packetLossRate := float64(stats.PacketsLost) / float64(stats.ChunksReceived+stats.PacketsLost) * 100

	logger.Info("Final streaming statistics",
		zap.Duration("total_duration", time.Since(stats.StartTime)),
		zap.Uint64("total_chunks", stats.ChunksReceived),
		zap.Uint64("total_bytes", stats.BytesReceived),
		zap.Float64("avg_bitrate_kbps", avgBitrate),
		zap.Duration("avg_latency", avgLatency),
		zap.Duration("min_latency", stats.MinLatency),
		zap.Duration("max_latency", stats.MaxLatency),
		zap.Uint64("packets_lost", stats.PacketsLost),
		zap.Float64("packet_loss_rate_percent", packetLossRate),
	)
}