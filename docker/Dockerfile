# Multi-stage build for QUIC Communication System
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the applications
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/iot-client ./cmd/iot-client
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/streaming-client ./cmd/streaming-client
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/benchmark ./cmd/benchmark

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

# Copy binaries from builder
COPY --from=builder /app/bin/ ./bin/
COPY --from=builder /app/docs/ ./docs/

# Copy configuration files
COPY docker/config/ ./config/

# Change ownership
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose ports
EXPOSE 4433/udp
EXPOSE 8080/tcp

# Default command
CMD ["./bin/server"]