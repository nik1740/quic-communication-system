package benchmark

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

// TestConfig represents benchmark test configuration
type TestConfig struct {
	Protocol      string        `json:"protocol"`       // "quic" or "tcp"
	Endpoint      string        `json:"endpoint"`       // server endpoint
	TestType      string        `json:"test_type"`      // "latency", "throughput", "iot", "streaming"
	Duration      time.Duration `json:"duration"`       // test duration
	Clients       int           `json:"clients"`        // concurrent clients
	RequestSize   int           `json:"request_size"`   // request payload size
	PacketLoss    float64       `json:"packet_loss"`    // simulated packet loss %
	Bandwidth     int64         `json:"bandwidth"`      // bandwidth limit (bytes/s)
	Jitter        time.Duration `json:"jitter"`         // network jitter
}

// TestResult represents benchmark test results
type TestResult struct {
	Protocol        string        `json:"protocol"`
	TestType        string        `json:"test_type"`
	Duration        time.Duration `json:"duration"`
	TotalRequests   int64         `json:"total_requests"`
	SuccessRequests int64         `json:"success_requests"`
	FailedRequests  int64         `json:"failed_requests"`
	Throughput      float64       `json:"throughput_rps"`     // requests per second
	Bandwidth       float64       `json:"bandwidth_mbps"`     // megabits per second
	AvgLatency      float64       `json:"avg_latency_ms"`     // milliseconds
	MinLatency      float64       `json:"min_latency_ms"`     // milliseconds
	MaxLatency      float64       `json:"max_latency_ms"`     // milliseconds
	P95Latency      float64       `json:"p95_latency_ms"`     // 95th percentile
	P99Latency      float64       `json:"p99_latency_ms"`     // 99th percentile
	BytesSent       int64         `json:"bytes_sent"`
	BytesReceived   int64         `json:"bytes_received"`
	Errors          []string      `json:"errors,omitempty"`
	Timestamp       time.Time     `json:"timestamp"`
}

// Benchmarker handles performance testing
type Benchmarker struct {
	config    TestConfig
	httpClient *http.Client
	results   *TestResult
	latencies []float64
	mutex     sync.Mutex
}

// NewBenchmarker creates a new benchmarker
func NewBenchmarker(config TestConfig) *Benchmarker {
	// Configure HTTP client based on protocol
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     30 * time.Second,
	}

	// For HTTP/3 (QUIC), we would need a different transport
	// This is a simplified version for HTTP/1.1 and HTTP/2 over TCP
	if config.Protocol == "quic" {
		// In a real implementation, we'd use quic-go's HTTP/3 client
		log.Println("Note: Using HTTP/2 client for QUIC endpoint simulation")
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	return &Benchmarker{
		config:     config,
		httpClient: client,
		results: &TestResult{
			Protocol:  config.Protocol,
			TestType:  config.TestType,
			Timestamp: time.Now(),
		},
		latencies: make([]float64, 0),
	}
}

// Run executes the benchmark test
func (b *Benchmarker) Run(ctx context.Context) (*TestResult, error) {
	log.Printf("Starting %s benchmark: %s test with %d clients for %v",
		b.config.Protocol, b.config.TestType, b.config.Clients, b.config.Duration)

	start := time.Now()
	endTime := start.Add(b.config.Duration)

	// Create worker goroutines
	var wg sync.WaitGroup
	clientCtx, cancel := context.WithDeadline(ctx, endTime)
	defer cancel()

	for i := 0; i < b.config.Clients; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()
			b.runClient(clientCtx, clientID)
		}(i)
	}

	// Wait for all clients to finish
	wg.Wait()

	// Calculate final results
	b.calculateResults(time.Since(start))

	log.Printf("Benchmark completed: %d requests, %.2f RPS, %.2f ms avg latency",
		b.results.TotalRequests, b.results.Throughput, b.results.AvgLatency)

	return b.results, nil
}

func (b *Benchmarker) runClient(ctx context.Context, clientID int) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			err := b.makeRequest(clientID)
			if err != nil {
				b.mutex.Lock()
				b.results.FailedRequests++
				b.results.Errors = append(b.results.Errors, err.Error())
				b.mutex.Unlock()
			}
		}
	}
}

func (b *Benchmarker) makeRequest(clientID int) error {
	start := time.Now()

	// Build request URL based on test type
	url := b.buildRequestURL()
	
	// Create request payload
	payload := b.createPayload()
	
	// Make HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-ID", fmt.Sprintf("client_%d", clientID))
	
	resp, err := b.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	
	latency := time.Since(start)
	
	// Record metrics
	b.mutex.Lock()
	b.results.TotalRequests++
	if resp.StatusCode == 200 {
		b.results.SuccessRequests++
	} else {
		b.results.FailedRequests++
	}
	b.results.BytesSent += int64(len(payload))
	b.results.BytesReceived += int64(len(respBody))
	b.latencies = append(b.latencies, float64(latency.Nanoseconds())/1e6) // Convert to ms
	b.mutex.Unlock()
	
	return nil
}

func (b *Benchmarker) buildRequestURL() string {
	baseURL := b.config.Endpoint
	
	switch b.config.TestType {
	case "latency":
		return baseURL + "/benchmark/"
	case "throughput":
		return baseURL + "/benchmark/"
	case "iot":
		return baseURL + "/iot/sensor"
	case "streaming":
		return baseURL + "/stream/chunk/test_stream"
	default:
		return baseURL + "/health"
	}
}

func (b *Benchmarker) createPayload() []byte {
	switch b.config.TestType {
	case "iot":
		data := map[string]interface{}{
			"device_id":    fmt.Sprintf("bench_device_%d", time.Now().UnixNano()),
			"sensor_type":  "temperature",
			"value":        25.5,
			"unit":         "celsius",
			"timestamp":    time.Now(),
			"quality":      "reliable",
		}
		payload, _ := json.Marshal(data)
		return payload
	case "streaming":
		// Simulate video chunk request
		data := map[string]interface{}{
			"stream_id": "benchmark_stream",
			"quality":   "medium",
			"chunk":     time.Now().UnixNano() % 1000,
		}
		payload, _ := json.Marshal(data)
		return payload
	default:
		// Generic payload for latency/throughput tests
		data := make([]byte, b.config.RequestSize)
		for i := range data {
			data[i] = byte(i % 256)
		}
		return data
	}
}

func (b *Benchmarker) calculateResults(duration time.Duration) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	
	b.results.Duration = duration
	
	if duration.Seconds() > 0 {
		b.results.Throughput = float64(b.results.TotalRequests) / duration.Seconds()
		b.results.Bandwidth = float64(b.results.BytesSent+b.results.BytesReceived) * 8 / duration.Seconds() / 1e6 // Mbps
	}
	
	if len(b.latencies) > 0 {
		// Calculate latency statistics
		sum := 0.0
		min := b.latencies[0]
		max := b.latencies[0]
		
		for _, lat := range b.latencies {
			sum += lat
			if lat < min {
				min = lat
			}
			if lat > max {
				max = lat
			}
		}
		
		b.results.AvgLatency = sum / float64(len(b.latencies))
		b.results.MinLatency = min
		b.results.MaxLatency = max
		
		// Calculate percentiles (simplified)
		if len(b.latencies) >= 20 {
			p95Index := int(float64(len(b.latencies)) * 0.95)
			p99Index := int(float64(len(b.latencies)) * 0.99)
			
			// Sort latencies for percentile calculation
			sortedLatencies := make([]float64, len(b.latencies))
			copy(sortedLatencies, b.latencies)
			
			// Simple sort (for production use a proper sorting algorithm)
			for i := 0; i < len(sortedLatencies); i++ {
				for j := i + 1; j < len(sortedLatencies); j++ {
					if sortedLatencies[i] > sortedLatencies[j] {
						sortedLatencies[i], sortedLatencies[j] = sortedLatencies[j], sortedLatencies[i]
					}
				}
			}
			
			b.results.P95Latency = sortedLatencies[p95Index]
			b.results.P99Latency = sortedLatencies[p99Index]
		}
	}
}