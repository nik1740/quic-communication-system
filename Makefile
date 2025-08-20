# Makefile for QUIC Communication System

# Build variables
BINARY_DIR := bin
CMD_DIR := cmd
DOCKER_DIR := docker

# Binary names
QUIC_SERVER := $(BINARY_DIR)/quic-server
IOT_CLIENT := $(BINARY_DIR)/iot-client
STREAMING_CLIENT := $(BINARY_DIR)/streaming-client
BENCHMARK := $(BINARY_DIR)/benchmark

# Go build flags
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
CGO_ENABLED := 0
BUILD_FLAGS := -a -installsuffix cgo -ldflags '-w -s'

# Git information
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
VERSION := $(shell git describe --tags 2>/dev/null || echo "dev")

# Inject build information
LDFLAGS := -X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.BuildTime=$(BUILD_TIME)

.PHONY: all build clean test lint docker docker-build docker-push help

# Default target
all: build

# Build all binaries
build: $(QUIC_SERVER) $(IOT_CLIENT) $(STREAMING_CLIENT) $(BENCHMARK)

$(BINARY_DIR):
	mkdir -p $(BINARY_DIR)

$(QUIC_SERVER): $(BINARY_DIR)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
	go build $(BUILD_FLAGS) -ldflags '$(LDFLAGS)' \
	-o $(QUIC_SERVER) ./$(CMD_DIR)/server

$(IOT_CLIENT): $(BINARY_DIR)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
	go build $(BUILD_FLAGS) -ldflags '$(LDFLAGS)' \
	-o $(IOT_CLIENT) ./$(CMD_DIR)/iot-client

$(STREAMING_CLIENT): $(BINARY_DIR)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
	go build $(BUILD_FLAGS) -ldflags '$(LDFLAGS)' \
	-o $(STREAMING_CLIENT) ./$(CMD_DIR)/streaming-client

$(BENCHMARK): $(BINARY_DIR)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
	go build $(BUILD_FLAGS) -ldflags '$(LDFLAGS)' \
	-o $(BENCHMARK) ./$(CMD_DIR)/benchmark

# Clean build artifacts
clean:
	rm -rf $(BINARY_DIR)
	go clean ./...

# Install dependencies
deps:
	go mod download
	go mod tidy

# Run tests
test:
	go test -v -race -coverprofile=coverage.out ./...

# Run tests with coverage report
test-coverage: test
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run linting
lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

# Format code
fmt:
	go fmt ./...
	@which goimports > /dev/null || go install golang.org/x/tools/cmd/goimports@latest
	goimports -w .

# Run security scan
sec:
	@which gosec > /dev/null || go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	gosec ./...

# Docker targets
docker-build:
	docker build -f $(DOCKER_DIR)/Dockerfile -t quic-communication-system:$(VERSION) .
	docker tag quic-communication-system:$(VERSION) quic-communication-system:latest

docker-compose-build:
	docker-compose -f $(DOCKER_DIR)/docker-compose.yml build

docker-compose-up:
	docker-compose -f $(DOCKER_DIR)/docker-compose.yml up -d

docker-compose-down:
	docker-compose -f $(DOCKER_DIR)/docker-compose.yml down

docker-compose-logs:
	docker-compose -f $(DOCKER_DIR)/docker-compose.yml logs -f

# Development targets
dev-server: $(QUIC_SERVER)
	./$(QUIC_SERVER) --debug

dev-iot: $(IOT_CLIENT)
	./$(IOT_CLIENT) --debug --device-type temperature --interval 5

dev-streaming: $(STREAMING_CLIENT)
	./$(STREAMING_CLIENT) --debug --server localhost:8080 --list-streams

dev-benchmark: $(BENCHMARK)
	./$(BENCHMARK) --verbose --protocol both --test-type latency

# Quick development setup
dev-setup: deps build
	@echo "Development environment ready!"
	@echo "Run 'make dev-server' to start the server"

# Performance testing
perf-test: $(BENCHMARK)
	./$(BENCHMARK) --protocol both --test-type all --output results/perf-$(shell date +%Y%m%d-%H%M%S).json

# Integration tests
test-integration: build
	@echo "Running integration tests..."
	./scripts/integration-test.sh

# Release build for multiple platforms
release:
	@mkdir -p releases
	# Linux amd64
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -ldflags '$(LDFLAGS)' -o releases/quic-server-linux-amd64 ./$(CMD_DIR)/server
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -ldflags '$(LDFLAGS)' -o releases/iot-client-linux-amd64 ./$(CMD_DIR)/iot-client
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -ldflags '$(LDFLAGS)' -o releases/streaming-client-linux-amd64 ./$(CMD_DIR)/streaming-client
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -ldflags '$(LDFLAGS)' -o releases/benchmark-linux-amd64 ./$(CMD_DIR)/benchmark
	# Linux arm64
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -ldflags '$(LDFLAGS)' -o releases/quic-server-linux-arm64 ./$(CMD_DIR)/server
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -ldflags '$(LDFLAGS)' -o releases/iot-client-linux-arm64 ./$(CMD_DIR)/iot-client
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -ldflags '$(LDFLAGS)' -o releases/streaming-client-linux-arm64 ./$(CMD_DIR)/streaming-client
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -ldflags '$(LDFLAGS)' -o releases/benchmark-linux-arm64 ./$(CMD_DIR)/benchmark
	# macOS amd64
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -ldflags '$(LDFLAGS)' -o releases/quic-server-darwin-amd64 ./$(CMD_DIR)/server
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -ldflags '$(LDFLAGS)' -o releases/iot-client-darwin-amd64 ./$(CMD_DIR)/iot-client
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -ldflags '$(LDFLAGS)' -o releases/streaming-client-darwin-amd64 ./$(CMD_DIR)/streaming-client
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -ldflags '$(LDFLAGS)' -o releases/benchmark-darwin-amd64 ./$(CMD_DIR)/benchmark
	# macOS arm64
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -ldflags '$(LDFLAGS)' -o releases/quic-server-darwin-arm64 ./$(CMD_DIR)/server
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -ldflags '$(LDFLAGS)' -o releases/iot-client-darwin-arm64 ./$(CMD_DIR)/iot-client
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -ldflags '$(LDFLAGS)' -o releases/streaming-client-darwin-arm64 ./$(CMD_DIR)/streaming-client
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -ldflags '$(LDFLAGS)' -o releases/benchmark-darwin-arm64 ./$(CMD_DIR)/benchmark
	# Windows amd64
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -ldflags '$(LDFLAGS)' -o releases/quic-server-windows-amd64.exe ./$(CMD_DIR)/server
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -ldflags '$(LDFLAGS)' -o releases/iot-client-windows-amd64.exe ./$(CMD_DIR)/iot-client
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -ldflags '$(LDFLAGS)' -o releases/streaming-client-windows-amd64.exe ./$(CMD_DIR)/streaming-client
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -ldflags '$(LDFLAGS)' -o releases/benchmark-windows-amd64.exe ./$(CMD_DIR)/benchmark

# Create directories
dirs:
	mkdir -p $(BINARY_DIR) results logs config data

# Help target
help:
	@echo "QUIC Communication System - Build Commands"
	@echo ""
	@echo "Build Commands:"
	@echo "  build              Build all binaries"
	@echo "  clean              Clean build artifacts"
	@echo "  deps               Install/update dependencies"
	@echo "  release            Build for multiple platforms"
	@echo ""
	@echo "Testing Commands:"
	@echo "  test               Run unit tests"
	@echo "  test-coverage      Run tests with coverage report"
	@echo "  test-integration   Run integration tests"
	@echo "  perf-test          Run performance benchmarks"
	@echo ""
	@echo "Code Quality:"
	@echo "  lint               Run linting"
	@echo "  fmt                Format code"
	@echo "  sec                Run security scan"
	@echo ""
	@echo "Docker Commands:"
	@echo "  docker-build       Build Docker image"
	@echo "  docker-compose-build   Build with docker-compose"
	@echo "  docker-compose-up      Start services with docker-compose"
	@echo "  docker-compose-down    Stop services"
	@echo "  docker-compose-logs    View logs"
	@echo ""
	@echo "Development Commands:"
	@echo "  dev-setup          Setup development environment"
	@echo "  dev-server         Run server in development mode"
	@echo "  dev-iot            Run IoT client in development mode"
	@echo "  dev-streaming      Run streaming client in development mode"
	@echo "  dev-benchmark      Run benchmark in development mode"
	@echo ""
	@echo "Build info:"
	@echo "  Version: $(VERSION)"
	@echo "  Git Commit: $(GIT_COMMIT)"
	@echo "  Build Time: $(BUILD_TIME)"