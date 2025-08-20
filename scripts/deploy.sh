#!/bin/bash

set -e

echo "Deploying QUIC Communication System with Docker..."

# Build Docker images
echo "Building Docker images..."
cd docker
docker-compose build

# Create logs directory
mkdir -p ../logs

# Start services
echo "Starting services..."
docker-compose up -d

echo "Services started successfully!"
echo "QUIC server: https://localhost:8443"
echo "TCP server: https://localhost:8080"
echo ""
echo "To view logs:"
echo "  docker-compose logs -f"
echo ""
echo "To stop services:"
echo "  docker-compose down"
echo ""
echo "To run benchmark:"
echo "  docker-compose run benchmark"