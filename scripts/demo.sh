#!/bin/bash

# QUIC Communication System Demo Script
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Function to print colored output
print_header() {
    echo -e "${BLUE}=== $1 ===${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_info() {
    echo -e "${CYAN}ℹ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

# Demo functions
demo_health_check() {
    print_header "Health Check Demo"
    
    print_info "Testing QUIC server health endpoint..."
    if curl -k -s https://localhost:8443/health | jq . > /dev/null 2>&1; then
        print_success "QUIC server health check passed"
        curl -k -s https://localhost:8443/health | jq .
    else
        print_error "QUIC server health check failed"
        return 1
    fi
    
    echo
}

demo_iot_endpoints() {
    print_header "IoT Endpoints Demo"
    
    print_info "1. Getting list of registered devices..."
    curl -k -s https://localhost:8443/iot/devices | jq .
    
    print_info "2. Registering a new test device..."
    cat << EOF | curl -k -s -X POST https://localhost:8443/iot/devices -H "Content-Type: application/json" -d @- | jq .
{
    "id": "demo-sensor-001",
    "name": "Demo Temperature Sensor",
    "type": "temperature",
    "location": "demo-room"
}
EOF

    print_info "3. Submitting test sensor data..."
    cat << EOF | curl -k -s -X POST https://localhost:8443/iot/data -H "Content-Type: application/json" -d @- | jq .
{
    "id": "demo-sensor-001",
    "type": "temperature",
    "value": 23.5,
    "unit": "°C",
    "location": "demo-room",
    "quality": "good"
}
EOF

    print_info "4. Getting recent sensor readings..."
    curl -k -s "https://localhost:8443/iot/data?limit=5" | jq .
    
    print_info "5. Sending a test command..."
    cat << EOF | curl -k -s -X POST https://localhost:8443/iot/command -H "Content-Type: application/json" -d @- | jq .
{
    "device_id": "demo-sensor-001",
    "type": "ping",
    "payload": {
        "message": "test command from demo"
    },
    "priority": 1
}
EOF

    print_info "6. IoT service health check..."
    curl -k -s https://localhost:8443/iot/health | jq .
    
    echo
}

demo_streaming_endpoints() {
    print_header "Streaming Endpoints Demo"
    
    print_info "1. Getting list of available streams..."
    curl -k -s https://localhost:8443/stream/list | jq .
    
    print_info "2. Getting stream information for demo1..."
    curl -k -s https://localhost:8443/stream/info/demo1 | jq .
    
    print_info "3. Getting master playlist for demo1..."
    echo "Master playlist content:"
    curl -k -s https://localhost:8443/stream/manifest/demo1/master.m3u8
    
    print_info "4. Getting 720p quality playlist..."
    echo "720p playlist content:"
    curl -k -s https://localhost:8443/stream/manifest/demo1/720p.m3u8
    
    print_info "5. Testing video segment download (first 100 bytes)..."
    curl -k -s https://localhost:8443/stream/video/demo1/720p/0.ts | head -c 100 | xxd
    
    print_info "6. Streaming service health check..."
    curl -k -s https://localhost:8443/stream/health | jq .
    
    echo
}

demo_clients() {
    print_header "Client Applications Demo"
    
    print_info "1. Running IoT client for 10 seconds..."
    timeout 10s ./bin/iot-client --server https://localhost:8443 --type humidity --interval 2 --duration 10 &
    IOT_PID=$!
    
    print_info "2. Running streaming client to download a few segments..."
    mkdir -p /tmp/demo-downloads
    timeout 15s ./bin/streaming-client --server https://localhost:8443 --stream demo1 --quality 480p --output /tmp/demo-downloads || true
    
    wait $IOT_PID 2>/dev/null || true
    
    if [ -d "/tmp/demo-downloads" ]; then
        print_success "Downloaded files:"
        ls -la /tmp/demo-downloads/
    fi
    
    echo
}

demo_benchmark() {
    print_header "Benchmark Demo"
    
    print_info "Running a quick QUIC benchmark (15 seconds)..."
    ./bin/benchmark --server https://localhost:8443 --test iot --duration 15 --connections 3 --rate 2 --output /tmp/demo-quic-results.json
    
    if [ -f "/tmp/demo-quic-results.json" ]; then
        print_success "QUIC benchmark results:"
        cat /tmp/demo-quic-results.json | jq .
    fi
    
    echo
}

# Main demo execution
main() {
    print_header "QUIC Communication System Demo"
    
    # Check if server is running
    if ! curl -k -s https://localhost:8443/health > /dev/null 2>&1; then
        print_error "QUIC server is not running on https://localhost:8443"
        print_info "Please start the server first: ./bin/server"
        exit 1
    fi
    
    print_success "QUIC server is running!"
    
    # Check required tools
    for tool in curl jq; do
        if ! command -v $tool &> /dev/null; then
            print_warning "$tool is not installed. Some demo features may not work properly."
        fi
    done
    
    # Run demos
    demo_health_check
    demo_iot_endpoints
    demo_streaming_endpoints
    demo_clients
    demo_benchmark
    
    print_header "Demo Completed!"
    print_success "All demo scenarios executed successfully"
    
    echo
    print_info "To explore more:"
    echo "- Check the comprehensive README.md for detailed usage"
    echo "- Run longer benchmarks with different parameters"
    echo "- Try the Docker Compose setup for full orchestration"
    echo "- Compare QUIC vs TCP/TLS performance"
}

# Run the demo
main "$@"