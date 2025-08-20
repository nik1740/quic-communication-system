package benchmark

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"go.uber.org/zap"
)

// TestType represents the type of performance test
type TestType string

const (
	TestIoT       TestType = "iot"
	TestStreaming TestType = "streaming"
	TestLatency   TestType = "latency"
	TestThroughput TestType = "throughput"
)

// ProtocolType represents the protocol being tested
type ProtocolType string

const (
	ProtocolQUIC ProtocolType = "quic"
	ProtocolTCP  ProtocolType = "tcp"
)

// TestResult represents the result of a performance test
type TestResult struct {
	TestType        TestType      `json:"test_type"`
	Protocol        ProtocolType  `json:"protocol"`
	Duration        time.Duration `json:"duration"`
	Connections     int           `json:"connections"`
	BytesTransferred uint64       `json:"bytes_transferred"`
	MessagesCount   uint64        `json:"messages_count"`
	AvgLatency      time.Duration `json:"avg_latency"`
	MinLatency      time.Duration `json:"min_latency"`
	MaxLatency      time.Duration `json:"max_latency"`
	P95Latency      time.Duration `json:"p95_latency"`
	P99Latency      time.Duration `json:"p99_latency"`
	Throughput      float64       `json:"throughput_mbps"`
	PacketLoss      float64       `json:"packet_loss_rate"`
	ConnectionTime  time.Duration `json:"connection_time"`
	ErrorCount      uint64        `json:"error_count"`
	Timestamp       time.Time     `json:"timestamp"`
}

// BenchmarkConfig holds benchmark configuration
type BenchmarkConfig struct {
	TestType      TestType      `json:"test_type"`
	Protocol      ProtocolType  `json:"protocol"`
	ServerAddr    string        `json:"server_addr"`
	Connections   int           `json:"connections"`
	Duration      time.Duration `json:"duration"`
	MessageSize   int           `json:"message_size"`
	MessageRate   int           `json:"message_rate"`
	PacketLoss    float64       `json:"packet_loss"`
	Latency       time.Duration `json:"latency"`
	Bandwidth     int           `json:"bandwidth_mbps"`
}

// Benchmark manages performance testing
type Benchmark struct {
	logger  *zap.Logger
	results []TestResult
	mutex   sync.RWMutex
}

// NewBenchmark creates a new benchmark instance
func NewBenchmark(logger *zap.Logger) *Benchmark {
	return &Benchmark{
		logger:  logger,
		results: make([]TestResult, 0),
	}
}

// RunTest runs a performance test with the given configuration
func (b *Benchmark) RunTest(ctx context.Context, config BenchmarkConfig) (*TestResult, error) {
	b.logger.Info("Starting benchmark test",
		zap.String("test_type", string(config.TestType)),
		zap.String("protocol", string(config.Protocol)),
		zap.String("server", config.ServerAddr),
		zap.Int("connections", config.Connections),
		zap.Duration("duration", config.Duration),
	)

	startTime := time.Now()
	
	result := &TestResult{
		TestType:    config.TestType,
		Protocol:    config.Protocol,
		Duration:    config.Duration,
		Connections: config.Connections,
		Timestamp:   startTime,
	}

	switch config.TestType {
	case TestLatency:
		return b.runLatencyTest(ctx, config, result)
	case TestThroughput:
		return b.runThroughputTest(ctx, config, result)
	case TestIoT:
		return b.runIoTTest(ctx, config, result)
	case TestStreaming:
		return b.runStreamingTest(ctx, config, result)
	default:
		return nil, fmt.Errorf("unknown test type: %s", config.TestType)
	}
}

// runLatencyTest measures connection establishment and message latency
func (b *Benchmark) runLatencyTest(ctx context.Context, config BenchmarkConfig, result *TestResult) (*TestResult, error) {
	b.logger.Info("Running latency test")
	
	latencies := make([]time.Duration, 0)
	connectionTimes := make([]time.Duration, 0)
	var totalBytes uint64
	var errorCount uint64
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Create test context with timeout
	testCtx, cancel := context.WithTimeout(ctx, config.Duration)
	defer cancel()

	// Run concurrent connections
	for i := 0; i < config.Connections; i++ {
		wg.Add(1)
		go func(connID int) {
			defer wg.Done()

			// Simulate connection establishment based on protocol
			var connLatency time.Duration
			if config.Protocol == ProtocolQUIC {
				connLatency = b.simulateQUICConnection()
			} else {
				connLatency = b.simulateTCPConnection()
			}
			
			mu.Lock()
			connectionTimes = append(connectionTimes, connLatency)
			mu.Unlock()

			// Send messages and measure latency
			messageCount := 0
			ticker := time.NewTicker(time.Second / time.Duration(config.MessageRate))
			defer ticker.Stop()

			for {
				select {
				case <-testCtx.Done():
					return
				case <-ticker.C:
					// Simulate message sending and response
					msgLatency, bytes, err := b.simulateMessage(config.Protocol, config.MessageSize)
					if err != nil {
						mu.Lock()
						errorCount++
						mu.Unlock()
						continue
					}

					mu.Lock()
					latencies = append(latencies, msgLatency)
					totalBytes += uint64(bytes)
					mu.Unlock()

					messageCount++
					if messageCount >= 100 { // Limit messages per connection
						return
					}
				}
			}
		}(i)
	}

	wg.Wait()

	// Calculate statistics
	result.MessagesCount = uint64(len(latencies))
	result.BytesTransferred = totalBytes
	result.ErrorCount = errorCount

	if len(latencies) > 0 {
		result.AvgLatency = calculateAverage(latencies)
		result.MinLatency = calculateMin(latencies)
		result.MaxLatency = calculateMax(latencies)
		result.P95Latency = calculatePercentile(latencies, 95)
		result.P99Latency = calculatePercentile(latencies, 99)
	}

	if len(connectionTimes) > 0 {
		result.ConnectionTime = calculateAverage(connectionTimes)
	}

	duration := time.Since(result.Timestamp).Seconds()
	if duration > 0 {
		result.Throughput = float64(totalBytes*8) / duration / 1000000 // Mbps
	}

	b.storeResult(*result)
	return result, nil
}

// runThroughputTest measures data transfer throughput
func (b *Benchmark) runThroughputTest(ctx context.Context, config BenchmarkConfig, result *TestResult) (*TestResult, error) {
	b.logger.Info("Running throughput test")
	
	var totalBytes uint64
	var errorCount uint64
	var wg sync.WaitGroup
	var mu sync.Mutex

	testCtx, cancel := context.WithTimeout(ctx, config.Duration)
	defer cancel()

	// Large message size for throughput testing
	largeMessageSize := 64 * 1024 // 64KB

	for i := 0; i < config.Connections; i++ {
		wg.Add(1)
		go func(connID int) {
			defer wg.Done()

			for {
				select {
				case <-testCtx.Done():
					return
				default:
					_, bytes, err := b.simulateMessage(config.Protocol, largeMessageSize)
					if err != nil {
						mu.Lock()
						errorCount++
						mu.Unlock()
						continue
					}

					mu.Lock()
					totalBytes += uint64(bytes)
					mu.Unlock()
				}
			}
		}(i)
	}

	wg.Wait()

	result.BytesTransferred = totalBytes
	result.ErrorCount = errorCount

	duration := time.Since(result.Timestamp).Seconds()
	if duration > 0 {
		result.Throughput = float64(totalBytes*8) / duration / 1000000 // Mbps
	}

	b.storeResult(*result)
	return result, nil
}

// runIoTTest simulates IoT device communication patterns
func (b *Benchmark) runIoTTest(ctx context.Context, config BenchmarkConfig, result *TestResult) (*TestResult, error) {
	b.logger.Info("Running IoT simulation test")
	
	latencies := make([]time.Duration, 0)
	var totalBytes uint64
	var messageCount uint64
	var errorCount uint64
	var wg sync.WaitGroup
	var mu sync.Mutex

	testCtx, cancel := context.WithTimeout(ctx, config.Duration)
	defer cancel()

	// Simulate IoT devices (many small, frequent messages)
	for i := 0; i < config.Connections; i++ {
		wg.Add(1)
		go func(deviceID int) {
			defer wg.Done()

			// IoT devices send small messages frequently
			ticker := time.NewTicker(5 * time.Second) // Every 5 seconds
			defer ticker.Stop()

			for {
				select {
				case <-testCtx.Done():
					return
				case <-ticker.C:
					// Send sensor data (small payload)
					latency, bytes, err := b.simulateMessage(config.Protocol, 128) // 128 bytes
					if err != nil {
						mu.Lock()
						errorCount++
						mu.Unlock()
						continue
					}

					mu.Lock()
					latencies = append(latencies, latency)
					totalBytes += uint64(bytes)
					messageCount++
					mu.Unlock()
				}
			}
		}(i)
	}

	wg.Wait()

	result.MessagesCount = messageCount
	result.BytesTransferred = totalBytes
	result.ErrorCount = errorCount

	if len(latencies) > 0 {
		result.AvgLatency = calculateAverage(latencies)
		result.MinLatency = calculateMin(latencies)
		result.MaxLatency = calculateMax(latencies)
		result.P95Latency = calculatePercentile(latencies, 95)
		result.P99Latency = calculatePercentile(latencies, 99)
	}

	duration := time.Since(result.Timestamp).Seconds()
	if duration > 0 {
		result.Throughput = float64(totalBytes*8) / duration / 1000000 // Mbps
	}

	b.storeResult(*result)
	return result, nil
}

// runStreamingTest simulates video streaming patterns
func (b *Benchmark) runStreamingTest(ctx context.Context, config BenchmarkConfig, result *TestResult) (*TestResult, error) {
	b.logger.Info("Running streaming simulation test")
	
	var totalBytes uint64
	var chunkCount uint64
	var errorCount uint64
	var wg sync.WaitGroup
	var mu sync.Mutex

	testCtx, cancel := context.WithTimeout(ctx, config.Duration)
	defer cancel()

	// Simulate video streams (large, regular chunks)
	for i := 0; i < config.Connections; i++ {
		wg.Add(1)
		go func(streamID int) {
			defer wg.Done()

			// Video chunks every 2 seconds
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()

			chunkSize := 64 * 1024 // 64KB video chunks

			for {
				select {
				case <-testCtx.Done():
					return
				case <-ticker.C:
					_, bytes, err := b.simulateMessage(config.Protocol, chunkSize)
					if err != nil {
						mu.Lock()
						errorCount++
						mu.Unlock()
						continue
					}

					mu.Lock()
					totalBytes += uint64(bytes)
					chunkCount++
					mu.Unlock()
				}
			}
		}(i)
	}

	wg.Wait()

	result.MessagesCount = chunkCount
	result.BytesTransferred = totalBytes
	result.ErrorCount = errorCount

	duration := time.Since(result.Timestamp).Seconds()
	if duration > 0 {
		result.Throughput = float64(totalBytes*8) / duration / 1000000 // Mbps
	}

	b.storeResult(*result)
	return result, nil
}

// Simulation functions (in real implementation, these would use actual network calls)

func (b *Benchmark) simulateQUICConnection() time.Duration {
	// QUIC 0-RTT or 1-RTT connection establishment
	base := 5 * time.Millisecond
	jitter := time.Duration(float64(base) * 0.2 * (2*rand.Float64() - 1))
	return base + jitter
}

func (b *Benchmark) simulateTCPConnection() time.Duration {
	// TCP 3-way handshake + TLS handshake
	base := 20 * time.Millisecond 
	jitter := time.Duration(float64(base) * 0.3 * (2*rand.Float64() - 1))
	return base + jitter
}

func (b *Benchmark) simulateMessage(protocol ProtocolType, size int) (time.Duration, int, error) {
	// Simulate network latency and processing time
	var baseLatency time.Duration
	if protocol == ProtocolQUIC {
		baseLatency = 2 * time.Millisecond
	} else {
		baseLatency = 5 * time.Millisecond
	}
	
	// Add jitter
	jitter := time.Duration(float64(baseLatency) * 0.4 * (2*rand.Float64() - 1))
	latency := baseLatency + jitter
	
	// Simulate processing time
	time.Sleep(latency)
	
	return latency, size, nil
}

// Statistical helper functions

func calculateAverage(values []time.Duration) time.Duration {
	if len(values) == 0 {
		return 0
	}
	var sum time.Duration
	for _, v := range values {
		sum += v
	}
	return sum / time.Duration(len(values))
}

func calculateMin(values []time.Duration) time.Duration {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
	}
	return min
}

func calculateMax(values []time.Duration) time.Duration {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values {
		if v > max {
			max = v
		}
	}
	return max
}

func calculatePercentile(values []time.Duration, percentile float64) time.Duration {
	if len(values) == 0 {
		return 0
	}
	
	// Simple percentile calculation (in production, use a proper sorting algorithm)
	sorted := make([]time.Duration, len(values))
	copy(sorted, values)
	
	// Bubble sort (simple but inefficient for large datasets)
	for i := 0; i < len(sorted); i++ {
		for j := 0; j < len(sorted)-1-i; j++ {
			if sorted[j] > sorted[j+1] {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}
	
	index := int(float64(len(sorted)) * percentile / 100.0)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

// storeResult stores a test result
func (b *Benchmark) storeResult(result TestResult) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.results = append(b.results, result)
}

// GetResults returns all test results
func (b *Benchmark) GetResults() []TestResult {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	results := make([]TestResult, len(b.results))
	copy(results, b.results)
	return results
}

// CompareProtocols compares results between QUIC and TCP for the same test type
func (b *Benchmark) CompareProtocols(testType TestType) (quicResult, tcpResult *TestResult, improvement map[string]float64) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	var quic, tcp *TestResult
	for i := range b.results {
		result := &b.results[i]
		if result.TestType == testType {
			if result.Protocol == ProtocolQUIC {
				quic = result
			} else if result.Protocol == ProtocolTCP {
				tcp = result
			}
		}
	}

	if quic == nil || tcp == nil {
		return quic, tcp, nil
	}

	improvement = make(map[string]float64)
	
	// Calculate improvements (positive values mean QUIC is better)
	if tcp.AvgLatency > 0 {
		improvement["latency"] = (tcp.AvgLatency.Seconds() - quic.AvgLatency.Seconds()) / tcp.AvgLatency.Seconds() * 100
	}
	
	if tcp.ConnectionTime > 0 {
		improvement["connection_time"] = (tcp.ConnectionTime.Seconds() - quic.ConnectionTime.Seconds()) / tcp.ConnectionTime.Seconds() * 100
	}
	
	if tcp.Throughput > 0 {
		improvement["throughput"] = (quic.Throughput - tcp.Throughput) / tcp.Throughput * 100
	}
	
	if tcp.ErrorCount > quic.ErrorCount {
		improvement["reliability"] = float64(tcp.ErrorCount-quic.ErrorCount) / float64(tcp.ErrorCount) * 100
	}

	return quic, tcp, improvement
}