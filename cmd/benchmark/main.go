package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/nik1740/quic-communication-system/internal/benchmark"
)

func main() {
	var (
		quicAddr    = flag.String("quic", "https://localhost:8443", "QUIC server address")
		tcpAddr     = flag.String("tcp", "https://localhost:8080", "TCP server address")
		testType    = flag.String("test", "latency", "Test type (latency, throughput, iot, streaming)")
		duration    = flag.Duration("duration", 30*time.Second, "Test duration")
		clients     = flag.Int("clients", 10, "Number of concurrent clients")
		requestSize = flag.Int("size", 1024, "Request payload size in bytes")
		output      = flag.String("output", "", "Output file for results (JSON)")
		compare     = flag.Bool("compare", true, "Compare QUIC vs TCP performance")
	)
	flag.Parse()

	log.Printf("Starting benchmark tool")
	log.Printf("Test type: %s", *testType)
	log.Printf("Duration: %v", *duration)
	log.Printf("Clients: %d", *clients)
	log.Printf("Request size: %d bytes", *requestSize)

	ctx := context.Background()

	var results []benchmark.TestResult

	// Test QUIC
	log.Println("Testing QUIC protocol...")
	quicConfig := benchmark.TestConfig{
		Protocol:    "quic",
		Endpoint:    *quicAddr,
		TestType:    *testType,
		Duration:    *duration,
		Clients:     *clients,
		RequestSize: *requestSize,
	}

	quicBench := benchmark.NewBenchmarker(quicConfig)
	quicResult, err := quicBench.Run(ctx)
	if err != nil {
		log.Printf("QUIC test failed: %v", err)
	} else {
		results = append(results, *quicResult)
		printResult("QUIC", quicResult)
	}

	if *compare {
		// Test TCP
		log.Println("Testing TCP protocol...")
		tcpConfig := benchmark.TestConfig{
			Protocol:    "tcp",
			Endpoint:    *tcpAddr,
			TestType:    *testType,
			Duration:    *duration,
			Clients:     *clients,
			RequestSize: *requestSize,
		}

		tcpBench := benchmark.NewBenchmarker(tcpConfig)
		tcpResult, err := tcpBench.Run(ctx)
		if err != nil {
			log.Printf("TCP test failed: %v", err)
		} else {
			results = append(results, *tcpResult)
			printResult("TCP", tcpResult)
		}

		// Compare results
		if len(results) == 2 {
			compareResults(&results[0], &results[1])
		}
	}

	// Save results to file if specified
	if *output != "" {
		if err := saveResults(*output, results); err != nil {
			log.Printf("Failed to save results: %v", err)
		} else {
			log.Printf("Results saved to %s", *output)
		}
	}
}

func printResult(protocol string, result *benchmark.TestResult) {
	fmt.Printf("\n=== %s Results ===\n", protocol)
	fmt.Printf("Total Requests:    %d\n", result.TotalRequests)
	fmt.Printf("Success Rate:      %.2f%%\n", float64(result.SuccessRequests)/float64(result.TotalRequests)*100)
	fmt.Printf("Throughput:        %.2f requests/sec\n", result.Throughput)
	fmt.Printf("Bandwidth:         %.2f Mbps\n", result.Bandwidth)
	fmt.Printf("Average Latency:   %.2f ms\n", result.AvgLatency)
	fmt.Printf("Min Latency:       %.2f ms\n", result.MinLatency)
	fmt.Printf("Max Latency:       %.2f ms\n", result.MaxLatency)
	fmt.Printf("95th Percentile:   %.2f ms\n", result.P95Latency)
	fmt.Printf("99th Percentile:   %.2f ms\n", result.P99Latency)
	fmt.Printf("Bytes Sent:        %d\n", result.BytesSent)
	fmt.Printf("Bytes Received:    %d\n", result.BytesReceived)
	
	if len(result.Errors) > 0 {
		fmt.Printf("Errors:            %d\n", len(result.Errors))
		for i, err := range result.Errors {
			if i < 5 { // Show first 5 errors
				fmt.Printf("  - %s\n", err)
			}
		}
		if len(result.Errors) > 5 {
			fmt.Printf("  ... and %d more\n", len(result.Errors)-5)
		}
	}
}

func compareResults(quicResult, tcpResult *benchmark.TestResult) {
	fmt.Printf("\n=== QUIC vs TCP Comparison ===\n")
	
	// Throughput comparison
	throughputImprovement := (quicResult.Throughput - tcpResult.Throughput) / tcpResult.Throughput * 100
	fmt.Printf("Throughput:        QUIC %.2f vs TCP %.2f RPS (%.2f%% improvement)\n",
		quicResult.Throughput, tcpResult.Throughput, throughputImprovement)
	
	// Latency comparison
	latencyImprovement := (tcpResult.AvgLatency - quicResult.AvgLatency) / tcpResult.AvgLatency * 100
	fmt.Printf("Average Latency:   QUIC %.2f vs TCP %.2f ms (%.2f%% improvement)\n",
		quicResult.AvgLatency, tcpResult.AvgLatency, latencyImprovement)
	
	// Bandwidth comparison
	bandwidthImprovement := (quicResult.Bandwidth - tcpResult.Bandwidth) / tcpResult.Bandwidth * 100
	fmt.Printf("Bandwidth:         QUIC %.2f vs TCP %.2f Mbps (%.2f%% improvement)\n",
		quicResult.Bandwidth, tcpResult.Bandwidth, bandwidthImprovement)
	
	// Success rate comparison
	quicSuccessRate := float64(quicResult.SuccessRequests) / float64(quicResult.TotalRequests) * 100
	tcpSuccessRate := float64(tcpResult.SuccessRequests) / float64(tcpResult.TotalRequests) * 100
	fmt.Printf("Success Rate:      QUIC %.2f%% vs TCP %.2f%%\n", quicSuccessRate, tcpSuccessRate)
	
	// P95 latency comparison
	p95Improvement := (tcpResult.P95Latency - quicResult.P95Latency) / tcpResult.P95Latency * 100
	fmt.Printf("95th Percentile:   QUIC %.2f vs TCP %.2f ms (%.2f%% improvement)\n",
		quicResult.P95Latency, tcpResult.P95Latency, p95Improvement)
	
	// Summary
	fmt.Printf("\nSummary:\n")
	if throughputImprovement > 0 {
		fmt.Printf("✓ QUIC shows %.2f%% better throughput\n", throughputImprovement)
	} else {
		fmt.Printf("✗ TCP shows %.2f%% better throughput\n", -throughputImprovement)
	}
	
	if latencyImprovement > 0 {
		fmt.Printf("✓ QUIC shows %.2f%% lower latency\n", latencyImprovement)
	} else {
		fmt.Printf("✗ TCP shows %.2f%% lower latency\n", -latencyImprovement)
	}
	
	if bandwidthImprovement > 0 {
		fmt.Printf("✓ QUIC shows %.2f%% better bandwidth utilization\n", bandwidthImprovement)
	} else {
		fmt.Printf("✗ TCP shows %.2f%% better bandwidth utilization\n", -bandwidthImprovement)
	}
}

func saveResults(filename string, results []benchmark.TestResult) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(map[string]interface{}{
		"timestamp": time.Now(),
		"results":   results,
	})
}