package main

import (
	"context"
	"os"
	"time"

	"github.com/nik1740/quic-communication-system/internal/benchmark"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	logger = logrus.New()
	rootCmd = &cobra.Command{
		Use:   "benchmark",
		Short: "QUIC vs TCP/TLS Performance Benchmark Tool",
		Long:  "Comprehensive benchmarking tool for comparing QUIC and TCP/TLS performance",
		Run:   runBenchmark,
	}
)

func init() {
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	rootCmd.Flags().StringP("server", "s", "localhost:4433", "Server address for testing")
	rootCmd.Flags().StringP("protocol", "p", "both", "Protocol to test (quic, tcp, tls, both)")
	rootCmd.Flags().StringP("test-type", "t", "all", "Test type (latency, throughput, connection, reliability, all)")
	rootCmd.Flags().StringP("network", "n", "all", "Network condition (local, broadband, mobile4g, mobile3g, satellite, poor, all)")
	rootCmd.Flags().IntP("duration", "d", 30, "Test duration in seconds")
	rootCmd.Flags().IntP("concurrency", "c", 1, "Number of concurrent connections")
	rootCmd.Flags().StringP("output", "o", "benchmark_results.json", "Output file for results")
	rootCmd.Flags().BoolP("verbose", "v", false, "Enable verbose logging")

	viper.BindPFlags(rootCmd.Flags())
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logger.WithError(err).Fatal("Failed to execute command")
		os.Exit(1)
	}
}

func runBenchmark(cmd *cobra.Command, args []string) {
	if viper.GetBool("verbose") {
		logger.SetLevel(logrus.DebugLevel)
	}

	serverAddr := viper.GetString("server")
	protocolFilter := viper.GetString("protocol")
	testTypeFilter := viper.GetString("test-type")
	networkFilter := viper.GetString("network")
	duration := time.Duration(viper.GetInt("duration")) * time.Second
	concurrency := viper.GetInt("concurrency")
	outputFile := viper.GetString("output")

	logger.Infof("Starting QUIC vs TCP/TLS Performance Benchmark")
	logger.Infof("Server: %s", serverAddr)
	logger.Infof("Protocol Filter: %s", protocolFilter)
	logger.Infof("Test Type Filter: %s", testTypeFilter)
	logger.Infof("Network Filter: %s", networkFilter)
	logger.Infof("Duration: %v", duration)
	logger.Infof("Concurrency: %d", concurrency)

	// Create benchmark suite
	suite := benchmark.NewBenchmarkSuite(logger)

	// Get test configurations
	configs := generateTestConfigs(suite, serverAddr, protocolFilter, testTypeFilter, networkFilter, duration, concurrency)

	logger.Infof("Generated %d test configurations", len(configs))

	ctx := context.Background()
	successfulTests := 0
	failedTests := 0

	// Run tests
	for i, config := range configs {
		logger.Infof("Running test %d/%d: %s over %s in %s conditions", 
			i+1, len(configs), config.TestType, config.Protocol, config.NetworkCondition.Name)

		result, err := suite.RunTest(ctx, config)
		if err != nil {
			logger.WithError(err).Errorf("Test failed: %s over %s", config.TestType, config.Protocol)
			failedTests++
		} else {
			logger.Infof("Test completed: %s over %s (%.2fs)", 
				config.TestType, config.Protocol, result.Duration.Seconds())
			successfulTests++
		}

		// Small delay between tests
		time.Sleep(100 * time.Millisecond)
	}

	// Export results
	if err := suite.ExportResults(outputFile); err != nil {
		logger.WithError(err).Errorf("Failed to export results to %s", outputFile)
	} else {
		logger.Infof("Results exported to %s", outputFile)
	}

	// Print summary
	printSummary(suite, successfulTests, failedTests)
}

func generateTestConfigs(suite *benchmark.BenchmarkSuite, serverAddr, protocolFilter, testTypeFilter, networkFilter string, duration time.Duration, concurrency int) []benchmark.TestConfig {
	var configs []benchmark.TestConfig

	// Determine protocols to test
	protocols := []benchmark.Protocol{}
	switch protocolFilter {
	case "quic":
		protocols = []benchmark.Protocol{benchmark.ProtocolQUIC}
	case "tcp":
		protocols = []benchmark.Protocol{benchmark.ProtocolTCP}
	case "tls":
		protocols = []benchmark.Protocol{benchmark.ProtocolTLS}
	case "both", "all":
		protocols = []benchmark.Protocol{benchmark.ProtocolQUIC, benchmark.ProtocolTCP, benchmark.ProtocolTLS}
	default:
		protocols = []benchmark.Protocol{benchmark.ProtocolQUIC, benchmark.ProtocolTCP}
	}

	// Determine test types
	testTypes := []benchmark.TestType{}
	switch testTypeFilter {
	case "latency":
		testTypes = []benchmark.TestType{benchmark.TestTypeLatency}
	case "throughput":
		testTypes = []benchmark.TestType{benchmark.TestTypeThroughput}
	case "connection":
		testTypes = []benchmark.TestType{benchmark.TestTypeConnection}
	case "reliability":
		testTypes = []benchmark.TestType{benchmark.TestTypeReliability}
	case "all":
		testTypes = []benchmark.TestType{
			benchmark.TestTypeLatency,
			benchmark.TestTypeThroughput,
			benchmark.TestTypeConnection,
			benchmark.TestTypeReliability,
		}
	default:
		testTypes = []benchmark.TestType{benchmark.TestTypeLatency, benchmark.TestTypeThroughput}
	}

	// Get network conditions
	allConditions := suite.GetNetworkConditions()
	conditions := []benchmark.NetworkCondition{}

	if networkFilter == "all" {
		conditions = allConditions
	} else {
		for _, condition := range allConditions {
			if matchesNetworkFilter(condition.Name, networkFilter) {
				conditions = append(conditions, condition)
			}
		}
		if len(conditions) == 0 {
			// Default to first condition if no match
			conditions = []benchmark.NetworkCondition{allConditions[0]}
		}
	}

	// Generate all combinations
	for _, protocol := range protocols {
		for _, testType := range testTypes {
			for _, condition := range conditions {
				config := benchmark.TestConfig{
					Protocol:         protocol,
					TestType:         testType,
					NetworkCondition: condition,
					Duration:         duration,
					Concurrency:      concurrency,
					ServerAddr:       serverAddr,
					DataSize:         getDataSizeForTest(testType),
				}
				configs = append(configs, config)
			}
		}
	}

	return configs
}

func matchesNetworkFilter(conditionName, filter string) bool {
	switch filter {
	case "local":
		return conditionName == "Local Network"
	case "broadband":
		return conditionName == "Broadband"
	case "mobile4g":
		return conditionName == "Mobile 4G"
	case "mobile3g":
		return conditionName == "Mobile 3G"
	case "satellite":
		return conditionName == "Satellite"
	case "poor":
		return conditionName == "Poor Network"
	default:
		return false
	}
}

func getDataSizeForTest(testType benchmark.TestType) int64 {
	switch testType {
	case benchmark.TestTypeLatency:
		return 1024 // 1 KB
	case benchmark.TestTypeThroughput:
		return 10 * 1024 * 1024 // 10 MB
	case benchmark.TestTypeConnection:
		return 0 // No data transfer
	case benchmark.TestTypeReliability:
		return 100 * 1024 // 100 KB
	default:
		return 1024 * 1024 // 1 MB
	}
}

func printSummary(suite *benchmark.BenchmarkSuite, successful, failed int) {
	results := suite.GetResults()

	logger.Infof("\n=== BENCHMARK SUMMARY ===")
	logger.Infof("Total Tests: %d", len(results))
	logger.Infof("Successful: %d", successful)
	logger.Infof("Failed: %d", failed)
	logger.Infof("Success Rate: %.1f%%", float64(successful)/float64(len(results))*100)

	// Group results by protocol and test type
	protocolResults := make(map[benchmark.Protocol][]benchmark.TestResult)
	for _, result := range results {
		if result.Success {
			protocolResults[result.Protocol] = append(protocolResults[result.Protocol], result)
		}
	}

	logger.Infof("\n=== PERFORMANCE COMPARISON ===")

	for protocol, results := range protocolResults {
		logger.Infof("\n--- %s Results ---", protocol)

		// Calculate averages by test type
		testTypeMetrics := make(map[benchmark.TestType][]benchmark.TestMetrics)
		for _, result := range results {
			testTypeMetrics[result.TestType] = append(testTypeMetrics[result.TestType], result.Metrics)
		}

		for testType, metrics := range testTypeMetrics {
			logger.Infof("%s Tests: %d", testType, len(metrics))

			switch testType {
			case benchmark.TestTypeLatency:
				avgLatency := avgLatency(metrics)
				logger.Infof("  Average Latency: %v", avgLatency)

			case benchmark.TestTypeThroughput:
				avgThroughput := avgThroughput(metrics)
				logger.Infof("  Average Throughput: %.2f Mbps", avgThroughput)

			case benchmark.TestTypeConnection:
				avgConnTime := avgConnectionTime(metrics)
				logger.Infof("  Average Connection Time: %v", avgConnTime)

			case benchmark.TestTypeReliability:
				avgPacketLoss := avgPacketLoss(metrics)
				logger.Infof("  Average Packet Loss: %.2f%%", avgPacketLoss*100)
			}
		}
	}

	logger.Infof("\n=== RECOMMENDATIONS ===")
	generateRecommendations(protocolResults)
}

func avgLatency(metrics []benchmark.TestMetrics) time.Duration {
	if len(metrics) == 0 {
		return 0
	}
	total := time.Duration(0)
	for _, m := range metrics {
		total += m.AvgLatency
	}
	return total / time.Duration(len(metrics))
}

func avgThroughput(metrics []benchmark.TestMetrics) float64 {
	if len(metrics) == 0 {
		return 0
	}
	total := 0.0
	for _, m := range metrics {
		total += m.ThroughputMbps
	}
	return total / float64(len(metrics))
}

func avgConnectionTime(metrics []benchmark.TestMetrics) time.Duration {
	if len(metrics) == 0 {
		return 0
	}
	total := time.Duration(0)
	for _, m := range metrics {
		total += m.ConnectionTime
	}
	return total / time.Duration(len(metrics))
}

func avgPacketLoss(metrics []benchmark.TestMetrics) float64 {
	if len(metrics) == 0 {
		return 0
	}
	total := 0.0
	for _, m := range metrics {
		total += m.PacketLossRate
	}
	return total / float64(len(metrics))
}

func generateRecommendations(protocolResults map[benchmark.Protocol][]benchmark.TestResult) {
	logger.Infof("Based on the benchmark results:")

	// Compare QUIC vs TCP/TLS
	quicResults := protocolResults[benchmark.ProtocolQUIC]
	tcpResults := protocolResults[benchmark.ProtocolTCP]

	if len(quicResults) > 0 && len(tcpResults) > 0 {
		logger.Infof("• QUIC shows advantages in connection establishment due to 0-RTT/1-RTT handshake")
		logger.Infof("• QUIC provides better handling of network changes and packet loss")
		logger.Infof("• TCP/TLS may show better throughput in stable, low-latency networks")
		logger.Infof("• QUIC is recommended for mobile and high-latency environments")
		logger.Infof("• TCP/TLS is suitable for stable data center environments")
	}

	logger.Infof("\nFor detailed analysis, review the exported JSON results file.")
}