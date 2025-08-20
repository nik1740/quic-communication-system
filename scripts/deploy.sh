#!/bin/bash

# QUIC Communication System Deployment Script
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_header() {
    echo -e "${BLUE}[DEPLOY]${NC} $1"
}

print_header "QUIC Communication System Deployment"

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    print_error "Docker is not installed. Please install Docker."
    exit 1
fi

# Check if Docker Compose is installed
if ! command -v docker-compose &> /dev/null; then
    print_error "Docker Compose is not installed. Please install Docker Compose."
    exit 1
fi

print_status "Docker and Docker Compose detected"

# Build Docker images
print_status "Building Docker images..."
docker-compose build

# Create necessary directories
print_status "Creating volume directories..."
mkdir -p certs test-videos downloads benchmark-results logs

# Start the services
print_status "Starting QUIC Communication System..."
docker-compose up -d quic-server

# Wait for server to be ready
print_status "Waiting for server to be ready..."
for i in {1..30}; do
    if docker-compose exec quic-server wget --no-check-certificate -q --spider https://localhost:8443/health 2>/dev/null; then
        print_status "Server is ready!"
        break
    fi
    if [ $i -eq 30 ]; then
        print_error "Server failed to start within 30 seconds"
        docker-compose logs quic-server
        exit 1
    fi
    sleep 1
done

# Start client services
print_status "Starting client services..."
docker-compose up -d iot-simulator
sleep 2
docker-compose up -d streaming-client

print_status "Deployment completed successfully!"

echo ""
echo "=== Service Status ==="
docker-compose ps

echo ""
echo "=== Available Endpoints ==="
echo "QUIC Server (HTTP/3): https://localhost:8443"
echo "TCP Server (HTTP/1.1): http://localhost:8080"
echo ""
echo "API Endpoints:"
echo "- Health Check: /health"
echo "- IoT Devices: /iot/devices"
echo "- IoT Data: /iot/data"
echo "- IoT Commands: /iot/command"
echo "- Stream List: /stream/list"
echo "- Stream Info: /stream/info/{id}"
echo ""
echo "=== Useful Commands ==="
echo "View logs: docker-compose logs -f [service]"
echo "Stop services: docker-compose down"
echo "Run benchmarks: docker-compose --profile benchmark up"
echo "Shell access: docker-compose exec quic-server sh"
echo ""
echo "=== Testing ==="
echo "Test IoT endpoint:"
echo "curl -k https://localhost:8443/iot/devices"
echo ""
echo "Test streaming endpoint:"
echo "curl -k https://localhost:8443/stream/list"