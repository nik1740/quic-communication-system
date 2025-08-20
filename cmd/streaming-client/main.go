package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/nik1740/quic-communication-system/pkg/logging"
	"github.com/quic-go/quic-go/http3"
	"github.com/spf13/cobra"
)

var (
	serverAddr string
	streamID   string
	quality    string
	useQUIC    bool
	outputDir  string
)

type StreamInfo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Duration  int       `json:"duration"`
	Qualities []Quality `json:"qualities"`
	CreatedAt time.Time `json:"created_at"`
}

type Quality struct {
	Name       string `json:"name"`
	Resolution string `json:"resolution"`
	Bitrate    int    `json:"bitrate"`
	FPS        int    `json:"fps"`
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "streaming-client",
		Short: "Video Streaming Client",
		Long:  "Downloads video streams from the QUIC communication server",
		Run:   runClient,
	}

	rootCmd.Flags().StringVarP(&serverAddr, "server", "s", "https://localhost:8443", "Server address")
	rootCmd.Flags().StringVarP(&streamID, "stream", "S", "demo1", "Stream ID to download")
	rootCmd.Flags().StringVarP(&quality, "quality", "q", "720p", "Video quality (480p, 720p, 1080p)")
	rootCmd.Flags().BoolVarP(&useQUIC, "quic", "Q", true, "Use QUIC/HTTP3 protocol")
	rootCmd.Flags().StringVarP(&outputDir, "output", "o", "./downloads", "Output directory")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runClient(cmd *cobra.Command, args []string) {
	// Initialize logger
	logger, err := logging.NewLogger("info", "text", "")
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	logger.WithComponent("streaming-client").Info("Starting streaming client")

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		logger.WithComponent("streaming-client").WithError(err).Fatal("Failed to create output directory")
	}

	// Create HTTP client
	var client *http.Client
	if useQUIC {
		logger.WithComponent("streaming-client").Info("Using QUIC/HTTP3 protocol")
		client = &http.Client{
			Transport: &http3.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, // For demo with self-signed certs
				},
			},
		}
	} else {
		logger.WithComponent("streaming-client").Info("Using HTTP/1.1 protocol")
		client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	}

	// Get stream information
	streamInfo, err := getStreamInfo(client, serverAddr, streamID, logger)
	if err != nil {
		logger.WithComponent("streaming-client").WithError(err).Fatal("Failed to get stream info")
	}

	logger.WithComponent("streaming-client").WithFields(map[string]interface{}{
		"stream_id": streamInfo.ID,
		"name":      streamInfo.Name,
		"duration":  streamInfo.Duration,
		"qualities": len(streamInfo.Qualities),
	}).Info("Stream information retrieved")

	// Validate quality
	validQuality := false
	for _, q := range streamInfo.Qualities {
		if q.Name == quality {
			validQuality = true
			logger.WithComponent("streaming-client").WithFields(map[string]interface{}{
				"quality":    q.Name,
				"resolution": q.Resolution,
				"bitrate":    q.Bitrate,
			}).Info("Using quality")
			break
		}
	}

	if !validQuality {
		logger.WithComponent("streaming-client").WithField("quality", quality).Fatal("Invalid quality specified")
	}

	// Download manifest
	if err := downloadManifest(client, serverAddr, streamID, quality, outputDir, logger); err != nil {
		logger.WithComponent("streaming-client").WithError(err).Fatal("Failed to download manifest")
	}

	// Download video segments
	segments := streamInfo.Duration / 10 // 10 seconds per segment
	logger.WithComponent("streaming-client").WithField("segments", segments).Info("Starting segment downloads")

	startTime := time.Now()
	var totalBytes int64

	for i := 0; i < segments; i++ {
		segmentStart := time.Now()
		
		size, err := downloadSegment(client, serverAddr, streamID, quality, i, outputDir, logger)
		if err != nil {
			logger.WithComponent("streaming-client").WithError(err).WithField("segment", i).Error("Failed to download segment")
			continue
		}

		totalBytes += size
		segmentDuration := time.Since(segmentStart)

		logger.WithComponent("streaming-client").WithFields(map[string]interface{}{
			"segment":     i,
			"size":        size,
			"duration":    segmentDuration,
			"throughput":  float64(size) / segmentDuration.Seconds() / 1024 / 1024, // MB/s
		}).Debug("Segment downloaded")

		// Show progress
		if (i+1)%5 == 0 || i == segments-1 {
			progress := float64(i+1) / float64(segments) * 100
			avgThroughput := float64(totalBytes) / time.Since(startTime).Seconds() / 1024 / 1024
			
			logger.WithComponent("streaming-client").WithFields(map[string]interface{}{
				"progress":          fmt.Sprintf("%.1f%%", progress),
				"segments":          fmt.Sprintf("%d/%d", i+1, segments),
				"avg_throughput":    fmt.Sprintf("%.2f MB/s", avgThroughput),
				"total_downloaded":  fmt.Sprintf("%.2f MB", float64(totalBytes)/1024/1024),
			}).Info("Download progress")
		}
	}

	totalDuration := time.Since(startTime)
	avgThroughput := float64(totalBytes) / totalDuration.Seconds() / 1024 / 1024

	logger.WithComponent("streaming-client").WithFields(map[string]interface{}{
		"total_time":    totalDuration,
		"total_size":    fmt.Sprintf("%.2f MB", float64(totalBytes)/1024/1024),
		"avg_throughput": fmt.Sprintf("%.2f MB/s", avgThroughput),
		"segments":      segments,
	}).Info("Download completed")
}

func getStreamInfo(client *http.Client, serverAddr, streamID string, logger *logging.Logger) (*StreamInfo, error) {
	resp, err := client.Get(fmt.Sprintf("%s/stream/info/%s", serverAddr, streamID))
	if err != nil {
		return nil, fmt.Errorf("failed to get stream info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get stream info, status: %d", resp.StatusCode)
	}

	var streamInfo StreamInfo
	if err := json.NewDecoder(resp.Body).Decode(&streamInfo); err != nil {
		return nil, fmt.Errorf("failed to decode stream info: %w", err)
	}

	return &streamInfo, nil
}

func downloadManifest(client *http.Client, serverAddr, streamID, quality, outputDir string, logger *logging.Logger) error {
	// Download master manifest
	masterURL := fmt.Sprintf("%s/stream/manifest/%s/master.m3u8", serverAddr, streamID)
	if err := downloadFile(client, masterURL, fmt.Sprintf("%s/master.m3u8", outputDir)); err != nil {
		return fmt.Errorf("failed to download master manifest: %w", err)
	}

	// Download quality-specific manifest
	qualityURL := fmt.Sprintf("%s/stream/manifest/%s/%s.m3u8", serverAddr, streamID, quality)
	if err := downloadFile(client, qualityURL, fmt.Sprintf("%s/%s.m3u8", outputDir, quality)); err != nil {
		return fmt.Errorf("failed to download quality manifest: %w", err)
	}

	logger.WithComponent("streaming-client").Info("Manifests downloaded")
	return nil
}

func downloadSegment(client *http.Client, serverAddr, streamID, quality string, segment int, outputDir string, logger *logging.Logger) (int64, error) {
	url := fmt.Sprintf("%s/stream/video/%s/%s/%d.ts", serverAddr, streamID, quality, segment)
	outputPath := fmt.Sprintf("%s/segment_%d.ts", outputDir, segment)

	resp, err := client.Get(url)
	if err != nil {
		return 0, fmt.Errorf("failed to download segment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to download segment, status: %d", resp.StatusCode)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return 0, fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	size, err := io.Copy(file, resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to write segment data: %w", err)
	}

	return size, nil
}

func downloadFile(client *http.Client, url, outputPath string) error {
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file, status: %d", resp.StatusCode)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file data: %w", err)
	}

	return nil
}