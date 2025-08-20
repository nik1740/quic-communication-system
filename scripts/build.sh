#!/bin/bash

# QUIC Communication System Build Script
set -e

echo "Building QUIC Communication System..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
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

# Check if Go is installed
if ! command -v go &> /dev/null; then
    print_error "Go is not installed. Please install Go 1.19 or later."
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
REQUIRED_VERSION="1.19"

if ! printf '%s\n%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V -C; then
    print_error "Go version $GO_VERSION is too old. Please install Go $REQUIRED_VERSION or later."
    exit 1
fi

print_status "Go version $GO_VERSION detected"

# Create necessary directories
print_status "Creating directories..."
mkdir -p bin
mkdir -p certs
mkdir -p test-videos
mkdir -p downloads
mkdir -p benchmark-results
mkdir -p logs

# Download dependencies
print_status "Downloading dependencies..."
go mod tidy
go mod download

# Build all components
print_status "Building server..."
go build -o bin/server ./cmd/server

print_status "Building IoT client..."
go build -o bin/iot-client ./cmd/iot-client

print_status "Building streaming client..."
go build -o bin/streaming-client ./cmd/streaming-client

print_status "Building benchmark tool..."
go build -o bin/benchmark ./cmd/benchmark

# Run tests if any exist
if find . -name "*_test.go" -type f | grep -q .; then
    print_status "Running tests..."
    go test ./...
else
    print_warning "No tests found"
fi

# Check if binaries were created
for binary in server iot-client streaming-client benchmark; do
    if [ -f "bin/$binary" ]; then
        print_status "✓ $binary built successfully"
    else
        print_error "✗ Failed to build $binary"
        exit 1
    fi
done

print_status "Build completed successfully!"
print_status "Binaries are available in the 'bin/' directory"

echo ""
echo "Next steps:"
echo "1. Start the server: ./bin/server"
echo "2. Run IoT client: ./bin/iot-client --server https://localhost:8443"
echo "3. Run streaming client: ./bin/streaming-client --server https://localhost:8443"
echo "4. Run benchmarks: ./bin/benchmark --server https://localhost:8443"
echo ""
echo "Or use Docker:"
echo "docker-compose up"