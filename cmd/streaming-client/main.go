package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/quic-go/quic-go/http3"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	logger = logrus.New()
	rootCmd = &cobra.Command{
		Use:   "streaming-client",
		Short: "Video Streaming Client",
		Long:  "Client for testing video streaming over HTTP/3",
		Run:   runClient,
	}
)

type StreamManifest struct {
	StreamID     string         `json:"stream_id"`
	Title        string         `json:"title"`
	Duration     float64        `json:"duration"`
	Qualities    []QualityLevel `json:"qualities"`
	SegmentCount int            `json:"segment_count"`
}

type QualityLevel struct {
	Name       string `json:"name"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	Bitrate    int    `json:"bitrate"`
	Framerate  int    `json:"framerate"`
	SegmentURL string `json:"segment_url"`
}

func init() {
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	rootCmd.Flags().StringP("server", "s", "localhost:8080", "HTTP/3 server address")
	rootCmd.Flags().StringP("stream", "t", "", "Stream ID to request (auto-discover if not provided)")
	rootCmd.Flags().StringP("quality", "q", "720p", "Quality level to request")
	rootCmd.Flags().IntP("duration", "d", 30, "Duration to stream in seconds")
	rootCmd.Flags().BoolP("list-streams", "l", false, "List available streams and exit")
	rootCmd.Flags().BoolP("debug", "v", false, "Enable debug logging")

	viper.BindPFlags(rootCmd.Flags())
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logger.WithError(err).Fatal("Failed to execute command")
		os.Exit(1)
	}
}

func runClient(cmd *cobra.Command, args []string) {
	if viper.GetBool("debug") {
		logger.SetLevel(logrus.DebugLevel)
	}

	serverAddr := viper.GetString("server")
	streamID := viper.GetString("stream")
	quality := viper.GetString("quality")
	duration := viper.GetInt("duration")
	listOnly := viper.GetBool("list-streams")

	logger.Infof("Starting streaming client")
	logger.Infof("Server: %s", serverAddr)

	// Create HTTP/3 client
	client := &http.Client{
		Transport: &http3.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // For testing only
			},
		},
		Timeout: 30 * time.Second,
	}

	baseURL := fmt.Sprintf("https://%s", serverAddr)

	// Test server connectivity
	if err := testServerHealth(client, baseURL); err != nil {
		logger.WithError(err).Fatal("Failed to connect to server")
	}

	// List available streams
	streams, err := listStreams(client, baseURL)
	if err != nil {
		logger.WithError(err).Fatal("Failed to get stream list")
	}

	logger.Infof("Available streams:")
	for _, stream := range streams {
		logger.Infof("  - %s: %s (%.1fs)", stream.StreamID, stream.Title, stream.Duration)
		for _, q := range stream.Qualities {
			logger.Infof("    - %s (%dx%d, %d kbps)", q.Name, q.Width, q.Height, q.Bitrate)
		}
	}

	if listOnly {
		return
	}

	// Select stream if not specified
	if streamID == "" && len(streams) > 0 {
		streamID = streams[0].StreamID
		logger.Infof("No stream specified, using: %s", streamID)
	}

	if streamID == "" {
		logger.Fatal("No streams available")
	}

	// Get stream info
	streamInfo, err := getStreamInfo(client, baseURL, streamID)
	if err != nil {
		logger.WithError(err).Fatal("Failed to get stream info")
	}

	logger.Infof("Selected stream: %s (%s)", streamInfo.StreamID, streamInfo.Title)

	// Find requested quality
	var selectedQuality *QualityLevel
	for _, q := range streamInfo.Qualities {
		if q.Name == quality {
			selectedQuality = &q
			break
		}
	}

	if selectedQuality == nil {
		logger.Warnf("Quality %s not available, using first available: %s", 
			quality, streamInfo.Qualities[0].Name)
		selectedQuality = &streamInfo.Qualities[0]
	}

	logger.Infof("Selected quality: %s (%dx%d, %d kbps)", 
		selectedQuality.Name, selectedQuality.Width, selectedQuality.Height, selectedQuality.Bitrate)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start streaming
	go func() {
		if err := streamVideo(ctx, client, baseURL, selectedQuality, duration); err != nil {
			logger.WithError(err).Error("Streaming error")
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Infof("Streaming started. Press Ctrl+C to stop.")

	<-sigChan
	logger.Info("Shutdown signal received, stopping client...")

	cancel()
	logger.Info("Streaming client stopped")
}

// testServerHealth tests if the server is reachable
func testServerHealth(client *http.Client, baseURL string) error {
	resp, err := client.Get(baseURL + "/health")
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server unhealthy: status %d", resp.StatusCode)
	}

	var health map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return fmt.Errorf("failed to decode health response: %w", err)
	}

	logger.Infof("Server health: %s", health["status"])
	return nil
}

// listStreams retrieves the list of available streams
func listStreams(client *http.Client, baseURL string) ([]StreamManifest, error) {
	resp, err := client.Get(baseURL + "/api/streams")
	if err != nil {
		return nil, fmt.Errorf("failed to get streams: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get streams: status %d", resp.StatusCode)
	}

	var streams []StreamManifest
	if err := json.NewDecoder(resp.Body).Decode(&streams); err != nil {
		return nil, fmt.Errorf("failed to decode streams: %w", err)
	}

	return streams, nil
}

// getStreamInfo retrieves information about a specific stream
func getStreamInfo(client *http.Client, baseURL, streamID string) (*StreamManifest, error) {
	url := fmt.Sprintf("%s/api/streams/%s", baseURL, streamID)
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get stream info: status %d", resp.StatusCode)
	}

	var stream StreamManifest
	if err := json.NewDecoder(resp.Body).Decode(&stream); err != nil {
		return nil, fmt.Errorf("failed to decode stream info: %w", err)
	}

	return &stream, nil
}

// streamVideo streams video data from the server
func streamVideo(ctx context.Context, client *http.Client, baseURL string, quality *QualityLevel, duration int) error {
	url := baseURL + quality.SegmentURL
	logger.Infof("Starting video stream from: %s", url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to start stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("stream request failed: status %d", resp.StatusCode)
	}

	logger.Infof("Stream started, receiving data...")

	// Read streaming data
	buffer := make([]byte, 64*1024) // 64KB buffer
	totalBytes := 0
	startTime := time.Now()
	lastReport := time.Now()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Streaming cancelled")
			return ctx.Err()

		default:
			// Set read deadline
			if deadline, ok := ctx.Deadline(); ok {
				if tcpConn, ok := resp.Body.(interface{ SetReadDeadline(time.Time) error }); ok {
					tcpConn.SetReadDeadline(deadline)
				}
			}

			n, err := resp.Body.Read(buffer)
			if n > 0 {
				totalBytes += n
				
				// Log progress every 5 seconds
				if time.Since(lastReport) >= 5*time.Second {
					elapsed := time.Since(startTime)
					bytesPerSecond := float64(totalBytes) / elapsed.Seconds()
					mbps := (bytesPerSecond * 8) / (1024 * 1024) // Convert to Mbps
					
					logger.Infof("Streaming progress: %.2f MB received, %.2f Mbps, elapsed: %v", 
						float64(totalBytes)/(1024*1024), mbps, elapsed.Truncate(time.Second))
					lastReport = time.Now()
				}
			}

			if err != nil {
				if err == io.EOF {
					logger.Info("Stream ended")
					break
				}
				return fmt.Errorf("read error: %w", err)
			}

			// Check duration limit
			if time.Since(startTime) >= time.Duration(duration)*time.Second {
				logger.Infof("Duration limit reached (%d seconds)", duration)
				break
			}
		}
	}

	elapsed := time.Since(startTime)
	avgBytesPerSecond := float64(totalBytes) / elapsed.Seconds()
	avgMbps := (avgBytesPerSecond * 8) / (1024 * 1024)

	logger.Infof("Streaming completed:")
	logger.Infof("  Total bytes: %.2f MB", float64(totalBytes)/(1024*1024))
	logger.Infof("  Duration: %v", elapsed.Truncate(time.Millisecond))
	logger.Infof("  Average speed: %.2f Mbps", avgMbps)

	return nil
}