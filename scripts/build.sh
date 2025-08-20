#!/bin/bash

set -e

echo "Building QUIC Communication System..."

# Create bin directory
mkdir -p bin

# Build all components
echo "Building QUIC server..."
go build -o bin/server ./cmd/server

echo "Building TCP server..."
go build -o bin/tcp-server ./cmd/tcp-server

echo "Building IoT client..."
go build -o bin/iot-client ./cmd/iot-client

echo "Building streaming client..."
go build -o bin/streaming-client ./cmd/streaming-client

echo "Building benchmark tool..."
go build -o bin/benchmark ./cmd/benchmark

echo "Build completed successfully!"
echo "Binaries are available in the bin/ directory:"
ls -la bin/