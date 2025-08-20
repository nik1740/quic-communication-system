# Multi-stage build for QUIC Communication System
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build all binaries
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/iot-client ./cmd/iot-client
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/streaming-client ./cmd/streaming-client
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/benchmark ./cmd/benchmark

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1001 -S quic && \
    adduser -u 1001 -S quic -G quic

# Set working directory
WORKDIR /app

# Copy binaries from builder
COPY --from=builder /build/bin/ ./bin/

# Create directories
RUN mkdir -p certs test-videos benchmark-results downloads logs && \
    chown -R quic:quic /app

# Switch to non-root user
USER quic

# Expose ports
EXPOSE 8443 8080

# Default command
CMD ["./bin/server"]