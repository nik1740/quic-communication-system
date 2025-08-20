package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// StreamInfo represents video stream metadata
type StreamInfo struct {
	StreamID    string    `json:"stream_id"`
	Title       string    `json:"title"`
	Duration    int       `json:"duration"`
	Bitrates    []Bitrate `json:"bitrates"`
	Format      string    `json:"format"`
	Resolution  string    `json:"resolution"`
	FrameRate   int       `json:"frame_rate"`
	CreatedAt   time.Time `json:"created_at"`
}

// Bitrate represents different quality levels
type Bitrate struct {
	Quality    string `json:"quality"`
	Bitrate    int    `json:"bitrate"`
	Resolution string `json:"resolution"`
	URL        string `json:"url"`
}

func main() {
	var (
		serverAddr = flag.String("server", "https://localhost:8443", "Server address")
		streamID   = flag.String("stream", "stream_001", "Stream ID to play")
		quality    = flag.String("quality", "medium", "Video quality (low, medium, high, ultra)")
		duration   = flag.Duration("duration", 30*time.Second, "Playback duration")
		protocol   = flag.String("protocol", "quic", "Protocol to use (quic or tcp)")
	)
	flag.Parse()

	log.Printf("Starting streaming client")
	log.Printf("Server: %s", *serverAddr)
	log.Printf("Stream: %s", *streamID)
	log.Printf("Quality: %s", *quality)
	log.Printf("Duration: %v", *duration)
	log.Printf("Protocol: %s", *protocol)

	// Create HTTP client with TLS config
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		Timeout: 30 * time.Second,
	}

	// List available streams
	streams, err := listStreams(client, *serverAddr)
	if err != nil {
		log.Fatal("Failed to list streams:", err)
	}

	log.Printf("Available streams:")
	for _, stream := range streams {
		log.Printf("  - %s: %s (%s)", stream.StreamID, stream.Title, stream.Resolution)
	}

	// Get stream info
	streamInfo, err := getStreamInfo(client, *serverAddr, *streamID)
	if err != nil {
		log.Fatal("Failed to get stream info:", err)
	}

	log.Printf("Stream info: %s - %s (%s, %d fps)", 
		streamInfo.StreamID, streamInfo.Title, streamInfo.Resolution, streamInfo.FrameRate)

	// Start streaming
	startStreaming(client, *serverAddr, *streamID, *quality, *duration)
}

func listStreams(client *http.Client, serverAddr string) ([]StreamInfo, error) {
	url := serverAddr + "/stream/list"
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Streams []StreamInfo `json:"streams"`
		Count   int          `json:"count"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Streams, nil
}

func getStreamInfo(client *http.Client, serverAddr, streamID string) (*StreamInfo, error) {
	url := fmt.Sprintf("%s/stream/info/%s", serverAddr, streamID)
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var streamInfo StreamInfo
	if err := json.NewDecoder(resp.Body).Decode(&streamInfo); err != nil {
		return nil, err
	}

	return &streamInfo, nil
}

func startStreaming(client *http.Client, serverAddr, streamID, quality string, duration time.Duration) {
	start := time.Now()
	chunkIndex := 0
	totalBytes := int64(0)
	chunksReceived := 0

	ticker := time.NewTicker(100 * time.Millisecond) // Request chunks every 100ms
	defer ticker.Stop()

	timeout := time.After(duration)

	log.Printf("Starting stream playback...")

	for {
		select {
		case <-ticker.C:
			chunkStart := time.Now()
			
			bytes, err := getStreamChunk(client, serverAddr, streamID, quality, chunkIndex)
			if err != nil {
				log.Printf("Failed to get chunk %d: %v", chunkIndex, err)
				continue
			}

			latency := time.Since(chunkStart)
			totalBytes += int64(len(bytes))
			chunksReceived++
			chunkIndex++

			log.Printf("Chunk %d: %d bytes, %.2f ms latency", chunkIndex, len(bytes), float64(latency.Nanoseconds())/1e6)

		case <-timeout:
			elapsed := time.Since(start)
			avgBandwidth := float64(totalBytes*8) / elapsed.Seconds() / 1e6 // Mbps
			avgLatency := elapsed.Seconds() * 1000 / float64(chunksReceived) // ms per chunk

			log.Printf("Streaming completed:")
			log.Printf("  Duration: %v", elapsed)
			log.Printf("  Chunks received: %d", chunksReceived)
			log.Printf("  Total bytes: %d", totalBytes)
			log.Printf("  Average bandwidth: %.2f Mbps", avgBandwidth)
			log.Printf("  Average chunk latency: %.2f ms", avgLatency)
			return
		}
	}
}

func getStreamChunk(client *http.Client, serverAddr, streamID, quality string, chunkIndex int) ([]byte, error) {
	url := fmt.Sprintf("%s/stream/chunk/%s?quality=%s&chunk=%d", serverAddr, streamID, quality, chunkIndex)
	
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}