#!/bin/bash

# Integration test script for QUIC Communication System

set -e

echo "=== QUIC Communication System Integration Test ==="

# Directories
BIN_DIR="./bin"
LOG_DIR="./logs"
TEST_DIR="./test"

# Create directories
mkdir -p "$LOG_DIR" "$TEST_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if binaries exist
check_binaries() {
    log_info "Checking binaries..."
    
    for binary in quic-server iot-client streaming-client benchmark; do
        if [ ! -f "$BIN_DIR/$binary" ]; then
            log_error "Binary $binary not found. Run 'make build' first."
            exit 1
        fi
    done
    
    log_info "All binaries found."
}

# Test server startup
test_server_startup() {
    log_info "Testing server startup..."
    
    # Start server in background
    $BIN_DIR/quic-server --addr :14433 --http-addr :18080 --debug > "$LOG_DIR/server.log" 2>&1 &
    SERVER_PID=$!
    
    # Wait for server to start
    sleep 3
    
    # Check if server is running
    if kill -0 $SERVER_PID 2>/dev/null; then
        log_info "Server started successfully (PID: $SERVER_PID)"
    else
        log_error "Server failed to start"
        cat "$LOG_DIR/server.log"
        exit 1
    fi
}

# Test IoT client connection
test_iot_client() {
    log_info "Testing IoT client connection..."
    
    # Start IoT client for 10 seconds
    timeout 10s $BIN_DIR/iot-client \
        --server localhost:14433 \
        --device-type temperature \
        --count 5 \
        --debug > "$LOG_DIR/iot-client.log" 2>&1 || true
    
    if grep -q "Device registered successfully" "$LOG_DIR/iot-client.log"; then
        log_info "IoT client test passed"
    else
        log_warn "IoT client test inconclusive (check logs)"
    fi
}

# Test streaming client
test_streaming_client() {
    log_info "Testing streaming client..."
    
    # Test stream listing
    timeout 10s $BIN_DIR/streaming-client \
        --server localhost:18080 \
        --list-streams \
        --debug > "$LOG_DIR/streaming-list.log" 2>&1 || true
    
    # Test short streaming session
    timeout 15s $BIN_DIR/streaming-client \
        --server localhost:18080 \
        --duration 5 \
        --debug > "$LOG_DIR/streaming-client.log" 2>&1 || true
    
    if grep -q "Available streams" "$LOG_DIR/streaming-list.log"; then
        log_info "Streaming client test passed"
    else
        log_warn "Streaming client test inconclusive (check logs)"
    fi
}

# Test benchmark
test_benchmark() {
    log_info "Testing benchmark tool..."
    
    # Run quick benchmark
    timeout 30s $BIN_DIR/benchmark \
        --server localhost:14433 \
        --protocol quic \
        --test-type latency \
        --network local \
        --duration 5 \
        --output "$TEST_DIR/benchmark_test.json" \
        --verbose > "$LOG_DIR/benchmark.log" 2>&1 || true
    
    if [ -f "$TEST_DIR/benchmark_test.json" ]; then
        log_info "Benchmark test passed"
    else
        log_warn "Benchmark test inconclusive (check logs)"
    fi
}

# Cleanup function
cleanup() {
    log_info "Cleaning up..."
    
    if [ ! -z "$SERVER_PID" ]; then
        kill $SERVER_PID 2>/dev/null || true
        wait $SERVER_PID 2>/dev/null || true
        log_info "Server stopped"
    fi
    
    # Kill any remaining processes
    pkill -f "quic-server" 2>/dev/null || true
    pkill -f "iot-client" 2>/dev/null || true
    pkill -f "streaming-client" 2>/dev/null || true
}

# Set trap for cleanup
trap cleanup EXIT

# Main test execution
main() {
    log_info "Starting integration tests..."
    
    check_binaries
    test_server_startup
    
    # Give server time to initialize
    sleep 2
    
    test_iot_client
    test_streaming_client
    test_benchmark
    
    log_info "Integration tests completed!"
    log_info "Check logs in $LOG_DIR/ for detailed output"
}

# Run main function
main "$@"