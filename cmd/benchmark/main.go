package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nik1740/quic-communication-system/internal/benchmark"
	"go.uber.org/zap"
)

func main() {
	var (
		testType     = flag.String("test", "latency", "Test type (latency, throughput, iot, streaming)")
		protocol     = flag.String("protocol", "both", "Protocol to test (quic, tcp, both)")
		serverAddr   = flag.String("server", "localhost:4433", "Server address")
		connections  = flag.Int("connections", 10, "Number of concurrent connections")
		duration     = flag.Duration("duration", 30*time.Second, "Test duration")
		messageSize  = flag.Int("message-size", 1024, "Message size in bytes")
		messageRate  = flag.Int("message-rate", 10, "Messages per second per connection")
		outputFile   = flag.String("output", "", "Output file for results (JSON)")
		comparison   = flag.Bool("compare", false, "Compare QUIC vs TCP results")
		verbose      = flag.Bool("verbose", false, "Verbose logging")
	)
	flag.Parse()

	// Initialize logger
	var logger *zap.Logger
	var err error
	if *verbose {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = zap.NewProduction()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Validate test type
	testTypeEnum := benchmark.TestType(*testType)
	if !isValidTestType(testTypeEnum) {
		logger.Fatal("Invalid test type", zap.String("test_type", *testType))
	}

	// Create benchmark instance
	bench := benchmark.NewBenchmark(logger)

	// Determine protocols to test
	protocols := make([]benchmark.ProtocolType, 0)
	switch strings.ToLower(*protocol) {
	case "quic":
		protocols = append(protocols, benchmark.ProtocolQUIC)
	case "tcp":
		protocols = append(protocols, benchmark.ProtocolTCP)
	case "both":
		protocols = append(protocols, benchmark.ProtocolQUIC, benchmark.ProtocolTCP)
	default:
		logger.Fatal("Invalid protocol", zap.String("protocol", *protocol))
	}

	logger.Info("Starting benchmark suite",
		zap.String("test_type", *testType),
		zap.Strings("protocols", protocolsToStrings(protocols)),
		zap.String("server", *serverAddr),
		zap.Int("connections", *connections),
		zap.Duration("duration", *duration),
	)

	ctx := context.Background()
	results := make([]*benchmark.TestResult, 0)

	// Run tests for each protocol
	for _, proto := range protocols {
		config := benchmark.BenchmarkConfig{
			TestType:    testTypeEnum,
			Protocol:    proto,
			ServerAddr:  *serverAddr,
			Connections: *connections,
			Duration:    *duration,
			MessageSize: *messageSize,
			MessageRate: *messageRate,
		}

		logger.Info("Running test", 
			zap.String("protocol", string(proto)),
			zap.String("test_type", string(testTypeEnum)),
		)

		result, err := bench.RunTest(ctx, config)
		if err != nil {
			logger.Error("Test failed",
				zap.String("protocol", string(proto)),
				zap.Error(err),
			)
			continue
		}

		results = append(results, result)
		printResult(result, logger)
	}

	// Compare protocols if requested and both were tested
	if *comparison && len(protocols) == 2 {
		compareProtocols(bench, testTypeEnum, logger)
	}

	// Save results to file if specified
	if *outputFile != "" {
		if err := saveResults(results, *outputFile); err != nil {
			logger.Error("Failed to save results", zap.Error(err))
		} else {
			logger.Info("Results saved", zap.String("file", *outputFile))
		}
	}

	logger.Info("Benchmark completed successfully")
}

// isValidTestType checks if the test type is valid
func isValidTestType(testType benchmark.TestType) bool {
	switch testType {
	case benchmark.TestLatency, benchmark.TestThroughput, benchmark.TestIoT, benchmark.TestStreaming:
		return true
	default:
		return false
	}
}

// protocolsToStrings converts protocol enums to strings
func protocolsToStrings(protocols []benchmark.ProtocolType) []string {
	result := make([]string, len(protocols))
	for i, p := range protocols {
		result[i] = string(p)
	}
	return result
}

// printResult prints test results in a formatted way
func printResult(result *benchmark.TestResult, logger *zap.Logger) {
	logger.Info("Test completed",
		zap.String("protocol", string(result.Protocol)),
		zap.String("test_type", string(result.TestType)),
		zap.Duration("duration", result.Duration),
		zap.Int("connections", result.Connections),
		zap.Uint64("messages", result.MessagesCount),
		zap.Uint64("bytes_transferred", result.BytesTransferred),
		zap.Float64("throughput_mbps", result.Throughput),
		zap.Duration("avg_latency", result.AvgLatency),
		zap.Duration("min_latency", result.MinLatency),
		zap.Duration("max_latency", result.MaxLatency),
		zap.Duration("p95_latency", result.P95Latency),
		zap.Duration("p99_latency", result.P99Latency),
		zap.Duration("connection_time", result.ConnectionTime),
		zap.Uint64("errors", result.ErrorCount),
		zap.Float64("packet_loss", result.PacketLoss),
	)

	// Print human-readable summary
	fmt.Printf("\n=== %s Test Results (%s) ===\n", strings.ToUpper(string(result.Protocol)), result.TestType)
	fmt.Printf("Duration: %v\n", result.Duration)
	fmt.Printf("Connections: %d\n", result.Connections)
	fmt.Printf("Messages: %d\n", result.MessagesCount)
	fmt.Printf("Data Transferred: %.2f MB\n", float64(result.BytesTransferred)/(1024*1024))
	fmt.Printf("Throughput: %.2f Mbps\n", result.Throughput)
	
	if result.AvgLatency > 0 {
		fmt.Printf("Average Latency: %v\n", result.AvgLatency)
		fmt.Printf("Min Latency: %v\n", result.MinLatency)
		fmt.Printf("Max Latency: %v\n", result.MaxLatency)
		fmt.Printf("P95 Latency: %v\n", result.P95Latency)
		fmt.Printf("P99 Latency: %v\n", result.P99Latency)
	}
	
	if result.ConnectionTime > 0 {
		fmt.Printf("Connection Time: %v\n", result.ConnectionTime)
	}
	
	if result.ErrorCount > 0 {
		fmt.Printf("Errors: %d\n", result.ErrorCount)
	}
	
	if result.PacketLoss > 0 {
		fmt.Printf("Packet Loss: %.2f%%\n", result.PacketLoss)
	}
	fmt.Printf("Timestamp: %v\n", result.Timestamp.Format(time.RFC3339))
	fmt.Println()
}

// compareProtocols compares QUIC and TCP results
func compareProtocols(bench *benchmark.Benchmark, testType benchmark.TestType, logger *zap.Logger) {
	quicResult, tcpResult, improvements := bench.CompareProtocols(testType)
	
	if quicResult == nil || tcpResult == nil {
		logger.Warn("Cannot compare protocols - missing results")
		return
	}

	fmt.Printf("\n=== Protocol Comparison (%s) ===\n", testType)
	fmt.Printf("                    QUIC         TCP         Improvement\n")
	fmt.Printf("                    ----         ---         -----------\n")
	
	if quicResult.AvgLatency > 0 && tcpResult.AvgLatency > 0 {
		fmt.Printf("Avg Latency:        %-12v %-12v %+.1f%%\n", 
			quicResult.AvgLatency, tcpResult.AvgLatency, improvements["latency"])
	}
	
	if quicResult.ConnectionTime > 0 && tcpResult.ConnectionTime > 0 {
		fmt.Printf("Connection Time:    %-12v %-12v %+.1f%%\n", 
			quicResult.ConnectionTime, tcpResult.ConnectionTime, improvements["connection_time"])
	}
	
	fmt.Printf("Throughput:         %-12.2f %-12.2f %+.1f%%\n", 
		quicResult.Throughput, tcpResult.Throughput, improvements["throughput"])
	
	fmt.Printf("Error Count:        %-12d %-12d", 
		quicResult.ErrorCount, tcpResult.ErrorCount)
	if imp, exists := improvements["reliability"]; exists {
		fmt.Printf(" %+.1f%%", imp)
	}
	fmt.Println()
	
	fmt.Printf("Messages:           %-12d %-12d\n", 
		quicResult.MessagesCount, tcpResult.MessagesCount)
	
	fmt.Printf("Data Transferred:   %-12.2f %-12.2f\n", 
		float64(quicResult.BytesTransferred)/(1024*1024), 
		float64(tcpResult.BytesTransferred)/(1024*1024))

	fmt.Println()

	// Summary
	if improvements["latency"] > 0 {
		fmt.Printf("✅ QUIC shows %.1f%% better latency\n", improvements["latency"])
	} else if improvements["latency"] < 0 {
		fmt.Printf("❌ QUIC shows %.1f%% worse latency\n", -improvements["latency"])
	}

	if improvements["connection_time"] > 0 {
		fmt.Printf("✅ QUIC shows %.1f%% faster connection establishment\n", improvements["connection_time"])
	} else if improvements["connection_time"] < 0 {
		fmt.Printf("❌ QUIC shows %.1f%% slower connection establishment\n", -improvements["connection_time"])
	}

	if improvements["throughput"] > 0 {
		fmt.Printf("✅ QUIC shows %.1f%% better throughput\n", improvements["throughput"])
	} else if improvements["throughput"] < 0 {
		fmt.Printf("❌ QUIC shows %.1f%% worse throughput\n", -improvements["throughput"])
	}

	fmt.Println()
}

// saveResults saves test results to a JSON file
func saveResults(results []*benchmark.TestResult, filename string) error {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write results file: %w", err)
	}

	return nil
}