package benchmark

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Protocol represents the transport protocol being tested
type Protocol string

const (
	ProtocolQUIC Protocol = "quic"
	ProtocolTCP  Protocol = "tcp"
	ProtocolTLS  Protocol = "tls"
)

// TestType represents different types of performance tests
type TestType string

const (
	TestTypeLatency    TestType = "latency"
	TestTypeThroughput TestType = "throughput"
	TestTypeConnection TestType = "connection"
	TestTypeReliability TestType = "reliability"
)

// NetworkCondition represents simulated network conditions
type NetworkCondition struct {
	Name         string        `json:"name"`
	Latency      time.Duration `json:"latency"`
	PacketLoss   float64       `json:"packet_loss"`   // 0.0 to 1.0
	Bandwidth    int64         `json:"bandwidth"`     // bytes per second
	Jitter       time.Duration `json:"jitter"`
}

// TestResult represents the result of a single test
type TestResult struct {
	ID              string           `json:"id"`
	Protocol        Protocol         `json:"protocol"`
	TestType        TestType         `json:"test_type"`
	NetworkCondition NetworkCondition `json:"network_condition"`
	StartTime       time.Time        `json:"start_time"`
	EndTime         time.Time        `json:"end_time"`
	Duration        time.Duration    `json:"duration"`
	Success         bool             `json:"success"`
	ErrorMessage    string           `json:"error_message,omitempty"`
	Metrics         TestMetrics      `json:"metrics"`
}

// TestMetrics contains detailed performance metrics
type TestMetrics struct {
	// Connection metrics
	ConnectionTime    time.Duration `json:"connection_time"`
	HandshakeTime     time.Duration `json:"handshake_time"`
	FirstByteTime     time.Duration `json:"first_byte_time"`
	
	// Throughput metrics
	BytesSent         int64         `json:"bytes_sent"`
	BytesReceived     int64         `json:"bytes_received"`
	Throughput        float64       `json:"throughput"`        // bytes per second
	ThroughputMbps    float64       `json:"throughput_mbps"`   // megabits per second
	
	// Latency metrics
	MinLatency        time.Duration `json:"min_latency"`
	MaxLatency        time.Duration `json:"max_latency"`
	AvgLatency        time.Duration `json:"avg_latency"`
	MedianLatency     time.Duration `json:"median_latency"`
	P95Latency        time.Duration `json:"p95_latency"`
	P99Latency        time.Duration `json:"p99_latency"`
	
	// Reliability metrics
	PacketsSent       int64         `json:"packets_sent"`
	PacketsReceived   int64         `json:"packets_received"`
	PacketLossRate    float64       `json:"packet_loss_rate"`
	ErrorCount        int           `json:"error_count"`
	RetransmissionCount int         `json:"retransmission_count"`
	
	// Resource usage
	CPUUsage          float64       `json:"cpu_usage"`
	MemoryUsage       int64         `json:"memory_usage"`
	
	// Custom metrics
	CustomMetrics     map[string]interface{} `json:"custom_metrics,omitempty"`
}

// BenchmarkSuite manages and executes performance tests
type BenchmarkSuite struct {
	logger       *logrus.Logger
	results      []TestResult
	resultsMux   sync.RWMutex
	networkConditions []NetworkCondition
}

// NewBenchmarkSuite creates a new benchmark suite
func NewBenchmarkSuite(logger *logrus.Logger) *BenchmarkSuite {
	suite := &BenchmarkSuite{
		logger:  logger,
		results: make([]TestResult, 0),
	}
	
	// Initialize default network conditions
	suite.initDefaultNetworkConditions()
	
	return suite
}

// initDefaultNetworkConditions sets up common network test scenarios
func (bs *BenchmarkSuite) initDefaultNetworkConditions() {
	bs.networkConditions = []NetworkCondition{
		{
			Name:       "Local Network",
			Latency:    1 * time.Millisecond,
			PacketLoss: 0.0,
			Bandwidth:  1000 * 1024 * 1024, // 1 Gbps
			Jitter:     100 * time.Microsecond,
		},
		{
			Name:       "Broadband",
			Latency:    20 * time.Millisecond,
			PacketLoss: 0.001, // 0.1%
			Bandwidth:  100 * 1024 * 1024, // 100 Mbps
			Jitter:     2 * time.Millisecond,
		},
		{
			Name:       "Mobile 4G",
			Latency:    50 * time.Millisecond,
			PacketLoss: 0.01, // 1%
			Bandwidth:  50 * 1024 * 1024, // 50 Mbps
			Jitter:     10 * time.Millisecond,
		},
		{
			Name:       "Mobile 3G",
			Latency:    150 * time.Millisecond,
			PacketLoss: 0.02, // 2%
			Bandwidth:  5 * 1024 * 1024, // 5 Mbps
			Jitter:     25 * time.Millisecond,
		},
		{
			Name:       "Satellite",
			Latency:    600 * time.Millisecond,
			PacketLoss: 0.005, // 0.5%
			Bandwidth:  25 * 1024 * 1024, // 25 Mbps
			Jitter:     50 * time.Millisecond,
		},
		{
			Name:       "Poor Network",
			Latency:    200 * time.Millisecond,
			PacketLoss: 0.05, // 5%
			Bandwidth:  1 * 1024 * 1024, // 1 Mbps
			Jitter:     100 * time.Millisecond,
		},
	}
}

// TestConfig represents configuration for a test run
type TestConfig struct {
	Protocol         Protocol         `json:"protocol"`
	TestType         TestType         `json:"test_type"`
	NetworkCondition NetworkCondition `json:"network_condition"`
	Duration         time.Duration    `json:"duration"`
	Concurrency      int              `json:"concurrency"`
	DataSize         int64            `json:"data_size"`
	ServerAddr       string           `json:"server_addr"`
	CustomParams     map[string]interface{} `json:"custom_params,omitempty"`
}

// RunTest executes a single performance test
func (bs *BenchmarkSuite) RunTest(ctx context.Context, config TestConfig) (*TestResult, error) {
	testID := fmt.Sprintf("test_%d", time.Now().UnixNano())
	
	result := TestResult{
		ID:               testID,
		Protocol:         config.Protocol,
		TestType:         config.TestType,
		NetworkCondition: config.NetworkCondition,
		StartTime:        time.Now(),
		Metrics:          TestMetrics{
			CustomMetrics: make(map[string]interface{}),
		},
	}
	
	bs.logger.Infof("Starting test %s: %s over %s in %s conditions", 
		testID, config.TestType, config.Protocol, config.NetworkCondition.Name)
	
	var err error
	
	switch config.TestType {
	case TestTypeLatency:
		err = bs.runLatencyTest(ctx, &config, &result)
	case TestTypeThroughput:
		err = bs.runThroughputTest(ctx, &config, &result)
	case TestTypeConnection:
		err = bs.runConnectionTest(ctx, &config, &result)
	case TestTypeReliability:
		err = bs.runReliabilityTest(ctx, &config, &result)
	default:
		err = fmt.Errorf("unknown test type: %s", config.TestType)
	}
	
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	
	if err != nil {
		result.Success = false
		result.ErrorMessage = err.Error()
		bs.logger.WithError(err).Errorf("Test %s failed", testID)
	} else {
		result.Success = true
		bs.logger.Infof("Test %s completed successfully in %v", testID, result.Duration)
	}
	
	// Store result
	bs.resultsMux.Lock()
	bs.results = append(bs.results, result)
	bs.resultsMux.Unlock()
	
	return &result, err
}

// runLatencyTest measures round-trip latency
func (bs *BenchmarkSuite) runLatencyTest(ctx context.Context, config *TestConfig, result *TestResult) error {
	// Simulate latency test implementation
	latencies := make([]time.Duration, 0, 100)
	
	for i := 0; i < 100; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			start := time.Now()
			
			// Simulate network request with configured latency
			time.Sleep(config.NetworkCondition.Latency + 
				time.Duration(float64(config.NetworkCondition.Jitter) * (0.5 - rand.Float64())))
			
			latency := time.Since(start)
			latencies = append(latencies, latency)
		}
	}
	
	// Calculate latency statistics
	result.Metrics.MinLatency = minDuration(latencies)
	result.Metrics.MaxLatency = maxDuration(latencies)
	result.Metrics.AvgLatency = avgDuration(latencies)
	result.Metrics.MedianLatency = medianDuration(latencies)
	result.Metrics.P95Latency = percentileDuration(latencies, 0.95)
	result.Metrics.P99Latency = percentileDuration(latencies, 0.99)
	
	return nil
}

// runThroughputTest measures data throughput
func (bs *BenchmarkSuite) runThroughputTest(ctx context.Context, config *TestConfig, result *TestResult) error {
	dataSize := config.DataSize
	if dataSize == 0 {
		dataSize = 10 * 1024 * 1024 // 10 MB default
	}
	
	start := time.Now()
	
	// Simulate data transfer
	transferTime := time.Duration(float64(dataSize) / float64(config.NetworkCondition.Bandwidth) * float64(time.Second))
	
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(transferTime):
		// Transfer completed
	}
	
	duration := time.Since(start)
	
	result.Metrics.BytesSent = dataSize
	result.Metrics.BytesReceived = dataSize
	result.Metrics.Throughput = float64(dataSize) / duration.Seconds()
	result.Metrics.ThroughputMbps = (result.Metrics.Throughput * 8) / (1024 * 1024)
	
	return nil
}

// runConnectionTest measures connection establishment time
func (bs *BenchmarkSuite) runConnectionTest(ctx context.Context, config *TestConfig, result *TestResult) error {
	// Simulate connection establishment
	start := time.Now()
	
	// Different protocols have different handshake overhead
	var handshakeTime time.Duration
	switch config.Protocol {
	case ProtocolQUIC:
		handshakeTime = config.NetworkCondition.Latency // 1-RTT handshake
	case ProtocolTCP:
		handshakeTime = config.NetworkCondition.Latency * 3 // 3-way handshake
	case ProtocolTLS:
		handshakeTime = config.NetworkCondition.Latency * 4 // TCP + TLS handshake
	}
	
	time.Sleep(handshakeTime)
	
	result.Metrics.ConnectionTime = time.Since(start)
	result.Metrics.HandshakeTime = handshakeTime
	
	return nil
}

// runReliabilityTest measures connection reliability under adverse conditions
func (bs *BenchmarkSuite) runReliabilityTest(ctx context.Context, config *TestConfig, result *TestResult) error {
	packetCount := 1000
	packetsReceived := int64(packetCount)
	
	// Simulate packet loss
	lostPackets := int64(float64(packetCount) * config.NetworkCondition.PacketLoss)
	packetsReceived -= lostPackets
	
	result.Metrics.PacketsSent = int64(packetCount)
	result.Metrics.PacketsReceived = packetsReceived
	result.Metrics.PacketLossRate = float64(lostPackets) / float64(packetCount)
	result.Metrics.RetransmissionCount = int(lostPackets) // Simplified
	
	return nil
}

// GetResults returns all test results
func (bs *BenchmarkSuite) GetResults() []TestResult {
	bs.resultsMux.RLock()
	defer bs.resultsMux.RUnlock()
	
	results := make([]TestResult, len(bs.results))
	copy(results, bs.results)
	return results
}

// ExportResults exports results to JSON
func (bs *BenchmarkSuite) ExportResults(filename string) error {
	results := bs.GetResults()
	
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}
	
	return os.WriteFile(filename, data, 0644)
}

// GetNetworkConditions returns available network conditions
func (bs *BenchmarkSuite) GetNetworkConditions() []NetworkCondition {
	return bs.networkConditions
}

// Helper functions for statistics

func minDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	min := durations[0]
	for _, d := range durations[1:] {
		if d < min {
			min = d
		}
	}
	return min
}

func maxDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	max := durations[0]
	for _, d := range durations[1:] {
		if d > max {
			max = d
		}
	}
	return max
}

func avgDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	total := time.Duration(0)
	for _, d := range durations {
		total += d
	}
	return total / time.Duration(len(durations))
}

func medianDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	
	n := len(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}

func percentileDuration(durations []time.Duration, percentile float64) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	
	index := int(float64(len(sorted)) * percentile)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}