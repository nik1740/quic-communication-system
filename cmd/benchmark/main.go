package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/nik1740/quic-communication-system/pkg/logging"
	"github.com/quic-go/quic-go/http3"
	"github.com/spf13/cobra"
)

var (
	serverAddr    string
	duration      int
	connections   int
	requestsPerSec int
	useQUIC       bool
	testType      string
	outputFile    string
)

type BenchmarkResult struct {
	TestType         string        `json:"test_type"`
	Protocol         string        `json:"protocol"`
	Duration         time.Duration `json:"duration"`
	TotalRequests    int           `json:"total_requests"`
	SuccessfulReqs   int           `json:"successful_requests"`
	FailedReqs       int           `json:"failed_requests"`
	RequestsPerSec   float64       `json:"requests_per_second"`
	AvgResponseTime  time.Duration `json:"avg_response_time"`
	MinResponseTime  time.Duration `json:"min_response_time"`
	MaxResponseTime  time.Duration `json:"max_response_time"`
	P50ResponseTime  time.Duration `json:"p50_response_time"`
	P95ResponseTime  time.Duration `json:"p95_response_time"`
	P99ResponseTime  time.Duration `json:"p99_response_time"`
	Throughput       float64       `json:"throughput_mbps"`
	TotalBytes       int64         `json:"total_bytes"`
	Timestamp        time.Time     `json:"timestamp"`
}

type RequestResult struct {
	Duration time.Duration
	Size     int64
	Success  bool
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "benchmark",
		Short: "QUIC vs TCP/TLS Benchmark Tool",
		Long:  "Benchmarks QUIC vs TCP/TLS performance for IoT and streaming scenarios",
		Run:   runBenchmark,
	}

	rootCmd.Flags().StringVarP(&serverAddr, "server", "s", "https://localhost:8443", "Server address")
	rootCmd.Flags().IntVarP(&duration, "duration", "d", 60, "Test duration in seconds")
	rootCmd.Flags().IntVarP(&connections, "connections", "c", 10, "Number of concurrent connections")
	rootCmd.Flags().IntVarP(&requestsPerSec, "rate", "r", 10, "Requests per second per connection")
	rootCmd.Flags().BoolVarP(&useQUIC, "quic", "q", true, "Use QUIC protocol (false for TCP/TLS)")
	rootCmd.Flags().StringVarP(&testType, "test", "t", "iot", "Test type (iot, streaming, mixed)")
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file for results (JSON)")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runBenchmark(cmd *cobra.Command, args []string) {
	// Initialize logger
	logger, err := logging.NewLogger("info", "text", "")
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	protocol := "TCP/TLS"
	if useQUIC {
		protocol = "QUIC"
	}

	logger.WithComponent("benchmark").WithFields(map[string]interface{}{
		"protocol":         protocol,
		"test_type":        testType,
		"duration":         duration,
		"connections":      connections,
		"requests_per_sec": requestsPerSec,
	}).Info("Starting benchmark")

	// Create HTTP client
	var client *http.Client
	if useQUIC {
		client = &http.Client{
			Transport: &http3.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	} else {
		client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	}

	// Run benchmark
	results := runLoadTest(client, serverAddr, testType, duration, connections, requestsPerSec, logger)

	// Calculate statistics
	result := calculateResults(results, testType, protocol, duration, logger)

	// Display results
	displayResults(result, logger)

	// Save results to file if specified
	if outputFile != "" {
		if err := saveResults(result, outputFile, logger); err != nil {
			logger.WithComponent("benchmark").WithError(err).Error("Failed to save results")
		}
	}
}

func runLoadTest(client *http.Client, serverAddr, testType string, duration, connections, requestsPerSec int, logger *logging.Logger) []RequestResult {
	var wg sync.WaitGroup
	resultsChan := make(chan RequestResult, connections*duration*requestsPerSec)
	
	testDuration := time.Duration(duration) * time.Second
	requestInterval := time.Second / time.Duration(requestsPerSec)

	// Start concurrent workers
	for i := 0; i < connections; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			startTime := time.Now()
			ticker := time.NewTicker(requestInterval)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					if time.Since(startTime) >= testDuration {
						return
					}
					
					// Send request based on test type
					result := sendTestRequest(client, serverAddr, testType, workerID)
					resultsChan <- result
				}
			}
		}(i)
	}

	// Wait for all workers to complete
	wg.Wait()
	close(resultsChan)

	// Collect results
	var results []RequestResult
	for result := range resultsChan {
		results = append(results, result)
	}

	logger.WithComponent("benchmark").WithField("total_requests", len(results)).Info("Load test completed")
	return results
}

func sendTestRequest(client *http.Client, serverAddr, testType string, workerID int) RequestResult {
	start := time.Now()
	
	var url string
	switch testType {
	case "iot":
		url = fmt.Sprintf("%s/iot/devices", serverAddr)
	case "streaming":
		url = fmt.Sprintf("%s/stream/list", serverAddr)
	case "mixed":
		if workerID%2 == 0 {
			url = fmt.Sprintf("%s/iot/devices", serverAddr)
		} else {
			url = fmt.Sprintf("%s/stream/list", serverAddr)
		}
	default:
		url = fmt.Sprintf("%s/health", serverAddr)
	}

	resp, err := client.Get(url)
	if err != nil {
		return RequestResult{
			Duration: time.Since(start),
			Size:     0,
			Success:  false,
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return RequestResult{
			Duration: time.Since(start),
			Size:     0,
			Success:  false,
		}
	}

	return RequestResult{
		Duration: time.Since(start),
		Size:     int64(len(body)),
		Success:  resp.StatusCode == http.StatusOK,
	}
}

func calculateResults(results []RequestResult, testType, protocol string, duration int, logger *logging.Logger) BenchmarkResult {
	if len(results) == 0 {
		return BenchmarkResult{}
	}

	var totalDuration time.Duration
	var totalBytes int64
	var successCount int
	var durations []time.Duration

	minDuration := results[0].Duration
	maxDuration := results[0].Duration

	for _, result := range results {
		totalDuration += result.Duration
		totalBytes += result.Size
		durations = append(durations, result.Duration)

		if result.Success {
			successCount++
		}

		if result.Duration < minDuration {
			minDuration = result.Duration
		}
		if result.Duration > maxDuration {
			maxDuration = result.Duration
		}
	}

	// Sort durations for percentile calculations
	sortDurations(durations)

	avgDuration := totalDuration / time.Duration(len(results))
	p50 := durations[len(durations)*50/100]
	p95 := durations[len(durations)*95/100]
	p99 := durations[len(durations)*99/100]

	testDuration := time.Duration(duration) * time.Second
	rps := float64(len(results)) / testDuration.Seconds()
	throughput := float64(totalBytes) / testDuration.Seconds() / 1024 / 1024 // MB/s

	return BenchmarkResult{
		TestType:         testType,
		Protocol:         protocol,
		Duration:         testDuration,
		TotalRequests:    len(results),
		SuccessfulReqs:   successCount,
		FailedReqs:       len(results) - successCount,
		RequestsPerSec:   rps,
		AvgResponseTime:  avgDuration,
		MinResponseTime:  minDuration,
		MaxResponseTime:  maxDuration,
		P50ResponseTime:  p50,
		P95ResponseTime:  p95,
		P99ResponseTime:  p99,
		Throughput:       throughput,
		TotalBytes:       totalBytes,
		Timestamp:        time.Now(),
	}
}

func sortDurations(durations []time.Duration) {
	// Simple bubble sort for durations
	n := len(durations)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if durations[j] > durations[j+1] {
				durations[j], durations[j+1] = durations[j+1], durations[j]
			}
		}
	}
}

func displayResults(result BenchmarkResult, logger *logging.Logger) {
	fmt.Println("\n=== Benchmark Results ===")
	fmt.Printf("Test Type: %s\n", result.TestType)
	fmt.Printf("Protocol: %s\n", result.Protocol)
	fmt.Printf("Duration: %v\n", result.Duration)
	fmt.Printf("Total Requests: %d\n", result.TotalRequests)
	fmt.Printf("Successful: %d (%.2f%%)\n", result.SuccessfulReqs, float64(result.SuccessfulReqs)/float64(result.TotalRequests)*100)
	fmt.Printf("Failed: %d (%.2f%%)\n", result.FailedReqs, float64(result.FailedReqs)/float64(result.TotalRequests)*100)
	fmt.Printf("Requests/sec: %.2f\n", result.RequestsPerSec)
	fmt.Printf("Avg Response Time: %v\n", result.AvgResponseTime)
	fmt.Printf("Min Response Time: %v\n", result.MinResponseTime)
	fmt.Printf("Max Response Time: %v\n", result.MaxResponseTime)
	fmt.Printf("P50 Response Time: %v\n", result.P50ResponseTime)
	fmt.Printf("P95 Response Time: %v\n", result.P95ResponseTime)
	fmt.Printf("P99 Response Time: %v\n", result.P99ResponseTime)
	fmt.Printf("Throughput: %.2f MB/s\n", result.Throughput)
	fmt.Printf("Total Data: %.2f MB\n", float64(result.TotalBytes)/1024/1024)
	fmt.Println("========================")
}

func saveResults(result BenchmarkResult, filename string, logger *logging.Logger) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write results file: %w", err)
	}

	logger.WithComponent("benchmark").WithField("file", filename).Info("Results saved")
	return nil
}