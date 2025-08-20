#!/bin/bash

# QUIC Communication System Demo Script
set -e

echo "🎯 QUIC Communication System Demo"
echo "================================="

# Check if binaries exist
if [ ! -f "bin/server" ] || [ ! -f "bin/iot-client" ] || [ ! -f "bin/streaming-client" ] || [ ! -f "bin/benchmark" ]; then
    echo "❌ Binaries not found. Please run './scripts/build.sh' first."
    exit 1
fi

echo "🚀 Starting QUIC server..."
./bin/server --addr localhost:4433 --log-level info &
SERVER_PID=$!

# Wait for server to start
sleep 3

echo "📡 Starting IoT device simulation..."
./bin/iot-client --server localhost:4433 --device-id demo-temp-sensor --device-type temperature --interval 2s --duration 30s &
IOT_PID=$!

echo "🎬 Starting video streaming simulation..."
./bin/streaming-client --server localhost:4433 --stream-id demo-stream --quality 720p --duration 30s &
STREAM_PID=$!

# Wait for clients to run
sleep 35

echo "📊 Running performance benchmarks..."
echo ""
echo "🔬 Latency Test:"
./bin/benchmark --test latency --protocol both --server localhost:4433 --connections 5 --duration 15s --compare

echo ""
echo "🚀 Throughput Test:"
./bin/benchmark --test throughput --protocol both --server localhost:4433 --connections 3 --duration 15s --compare

echo ""
echo "🏠 IoT Simulation Test:"
./bin/benchmark --test iot --protocol both --server localhost:4433 --connections 10 --duration 20s --compare

echo ""
echo "🎥 Streaming Simulation Test:"
./bin/benchmark --test streaming --protocol both --server localhost:4433 --connections 2 --duration 20s --compare

# Cleanup
echo ""
echo "🧹 Cleaning up..."
kill $SERVER_PID $IOT_PID $STREAM_PID 2>/dev/null || true
wait 2>/dev/null || true

echo ""
echo "✅ Demo completed successfully!"
echo ""
echo "📋 Summary:"
echo "  - QUIC server handled IoT and streaming traffic"
echo "  - Performance comparison between QUIC and TCP/TLS"
echo "  - Multiple concurrent connections tested"
echo "  - Latency, throughput, and application-specific metrics measured"
echo ""
echo "💡 For more detailed analysis, run individual components with --help"