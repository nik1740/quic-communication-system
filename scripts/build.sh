#!/bin/bash

# QUIC Communication System Build Script
set -e

echo "🔨 Building QUIC Communication System..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.21 or later."
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
REQUIRED_VERSION="1.21"

if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V | head -n1)" != "$REQUIRED_VERSION" ]; then
    echo "❌ Go version $REQUIRED_VERSION or later is required. Found: $GO_VERSION"
    exit 1
fi

echo "✅ Go version: $GO_VERSION"

# Create output directory
mkdir -p bin

# Build all components
echo "📦 Building server..."
go build -o bin/server ./cmd/server

echo "📦 Building IoT client..."
go build -o bin/iot-client ./cmd/iot-client

echo "📦 Building streaming client..."
go build -o bin/streaming-client ./cmd/streaming-client

echo "📦 Building benchmark tool..."
go build -o bin/benchmark ./cmd/benchmark

echo "🧪 Running tests..."
go test ./...

echo "✅ Build completed successfully!"
echo ""
echo "🚀 Available binaries:"
echo "  - bin/server           : QUIC communication server"
echo "  - bin/iot-client       : IoT device simulator"
echo "  - bin/streaming-client : Video streaming client"
echo "  - bin/benchmark        : Performance benchmarking tool"
echo ""
echo "💡 Usage examples:"
echo "  ./bin/server --help"
echo "  ./bin/iot-client --help"
echo "  ./bin/streaming-client --help"
echo "  ./bin/benchmark --help"