# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the applications
RUN go build -o bin/server ./cmd/server
RUN go build -o bin/iot-client ./cmd/iot-client
RUN go build -o bin/streaming-client ./cmd/streaming-client
RUN go build -o bin/benchmark ./cmd/benchmark

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binaries from builder
COPY --from=builder /app/bin/ ./bin/

# Create directories for data
RUN mkdir -p /app/test /app/logs

# Expose ports
EXPOSE 8443 8080

# Default command
CMD ["./bin/server"]