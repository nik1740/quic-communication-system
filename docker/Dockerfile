# Build stage
FROM golang:1.21-alpine AS builder

# Install dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o quic-server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o iot-client ./cmd/iot-client
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o streaming-client ./cmd/streaming-client
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o benchmark ./cmd/benchmark

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binaries from builder stage
COPY --from=builder /app/quic-server .
COPY --from=builder /app/iot-client .
COPY --from=builder /app/streaming-client .
COPY --from=builder /app/benchmark .

# Create directories for data and configs
RUN mkdir -p /data /config

# Expose ports
EXPOSE 4433/udp 8080/tcp

# Default command
CMD ["./quic-server"]