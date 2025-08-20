#!/bin/bash

# QUIC Communication System Docker Deployment Script
set -e

echo "🐳 Deploying QUIC Communication System with Docker..."

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed. Please install Docker first."
    exit 1
fi

# Check if Docker Compose is installed
if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    echo "❌ Docker Compose is not installed. Please install Docker Compose first."
    exit 1
fi

# Create necessary directories
mkdir -p docker/logs docker/results

echo "🏗️ Building Docker images..."
cd docker

# Build the main application image
docker build -t quic-communication-system -f Dockerfile ..

echo "🚀 Starting services..."
docker-compose up -d

echo "⏳ Waiting for services to start..."
sleep 10

echo "📊 Service status:"
docker-compose ps

echo ""
echo "✅ Deployment completed successfully!"
echo ""
echo "🌐 Available services:"
echo "  - QUIC Server     : UDP port 4433"
echo "  - TCP Server      : TCP port 4434" 
echo "  - Prometheus      : http://localhost:9090"
echo "  - Grafana         : http://localhost:3000 (admin/admin)"
echo ""
echo "🔧 Management commands:"
echo "  View logs         : docker-compose logs -f [service-name]"
echo "  Stop services     : docker-compose down"
echo "  Restart services  : docker-compose restart"
echo "  View benchmarks   : docker-compose logs benchmark"
echo ""
echo "📈 Benchmark results will be saved to: docker/results/"