# QUIC Communication System

A comprehensive QUIC-based communication system demonstrating the advantages of QUIC protocol over traditional TCP/TLS for IoT devices and video streaming applications.

## Features

- **QUIC Server**: HTTP/3 server implementation with connection multiplexing and stream prioritization
- **IoT Communication**: Lightweight sensor data transmission with reliable/unreliable stream support
- **Video Streaming**: Adaptive bitrate streaming with low latency and frame prioritization
- **TCP/TLS Comparison**: Equivalent TCP implementations for performance benchmarking
- **Benchmarking Framework**: Automated performance testing with comprehensive metrics
- **Docker Support**: Containerized deployment for consistent testing environments

## Architecture

```
quic-communication-system/
├── cmd/                    # Main applications
│   ├── server/            # QUIC/TCP server
│   ├── iot-client/        # IoT device simulator
│   ├── streaming-client/  # Video streaming client
│   └── benchmark/         # Performance testing tool
├── internal/              # Internal packages
│   ├── quic/             # QUIC utilities and configuration
│   ├── iot/              # IoT protocol handlers
│   ├── streaming/        # Video streaming protocols
│   ├── tcp/              # TCP/TLS comparison implementations
│   └── benchmark/        # Performance testing framework
├── docker/               # Container configurations
├── scripts/              # Build and deployment scripts
└── docs/                 # Documentation
```

## Quick Start

### Prerequisites

- Go 1.21 or later
- Docker and Docker Compose (optional)

### Local Development

1. **Clone and build:**
   ```bash
   git clone https://github.com/nik1740/quic-communication-system.git
   cd quic-communication-system
   go mod tidy
   ./scripts/build.sh
   ```

2. **Start the QUIC server:**
   ```bash
   ./bin/server
   ```

3. **Test IoT communication:**
   ```bash
   ./bin/iot-client -server https://localhost:8443 -device temp_sensor_01 -sensor temperature
   ```

4. **Test video streaming:**
   ```bash
   ./bin/streaming-client -server https://localhost:8443 -stream stream_001 -quality medium
   ```

5. **Run benchmarks:**
   ```bash
   ./bin/benchmark -test latency -duration 30s -clients 10
   ```

### Docker Deployment

1. **Deploy with Docker Compose:**
   ```bash
   ./scripts/deploy.sh
   ```

2. **View logs:**
   ```bash
   cd docker && docker-compose logs -f
   ```

3. **Run benchmarks:**
   ```bash
   cd docker && docker-compose run benchmark
   ```

## API Endpoints

### QUIC Server (Port 8443)

#### Health Check
- `GET /health` - Server health status

#### IoT Endpoints
- `GET /iot/sensor` - Get simulated sensor data
- `POST /iot/sensor` - Submit sensor readings
- `POST /iot/command` - Send device commands
- `GET /iot/devices` - List connected devices
- `GET /iot/simulate?devices=N&duration=Xs` - Start IoT simulation

#### Streaming Endpoints
- `GET /stream/list` - List available streams
- `GET /stream/info/{stream_id}` - Get stream metadata
- `GET /stream/chunk/{stream_id}?quality=X&chunk=N` - Get video chunk
- `GET /stream/stats/{stream_id}` - Get streaming statistics
- `GET /stream/live` - Live stream (Server-Sent Events)

### TCP Server (Port 8080)

Same endpoints as QUIC server for comparison testing.

## Performance Testing

### Test Types

1. **Latency Test**: Measures round-trip time for small requests
2. **Throughput Test**: Measures requests per second under load
3. **IoT Test**: Simulates sensor data transmission patterns
4. **Streaming Test**: Simulates video chunk delivery patterns

### Example Benchmark Commands

```bash
# Latency comparison
./bin/benchmark -test latency -duration 60s -clients 5

# Throughput test
./bin/benchmark -test throughput -duration 30s -clients 20 -size 4096

# IoT simulation
./bin/benchmark -test iot -duration 120s -clients 50

# Streaming performance
./bin/benchmark -test streaming -duration 60s -clients 10
```

### Sample Results

```
=== QUIC vs TCP Comparison ===
Throughput:        QUIC 1247.50 vs TCP 1089.33 RPS (14.52% improvement)
Average Latency:   QUIC 12.45 vs TCP 18.67 ms (33.32% improvement)
Bandwidth:         QUIC 15.23 vs TCP 13.87 Mbps (9.81% improvement)
95th Percentile:   QUIC 28.91 vs TCP 42.15 ms (31.41% improvement)
```

## Configuration

### Server Configuration

Environment variables:
- `SERVER_ADDR`: Server listen address (default: `:8443`)
- `LOG_LEVEL`: Logging level (default: `info`)

### Client Configuration

IoT Client flags:
- `-server`: Server address
- `-device`: Device ID
- `-sensor`: Sensor type (temperature, humidity, motion, pressure, light)
- `-interval`: Data transmission interval
- `-duration`: Total runtime

Streaming Client flags:
- `-server`: Server address
- `-stream`: Stream ID
- `-quality`: Video quality (low, medium, high, ultra)
- `-duration`: Playback duration

## QUIC Advantages Demonstrated

### 1. **Connection Establishment**
- QUIC: 1-RTT connection setup
- TCP+TLS: 3-RTT connection setup (TCP handshake + TLS handshake)

### 2. **Stream Multiplexing**
- QUIC: Native stream multiplexing without head-of-line blocking
- TCP: Single stream per connection, HOL blocking issues

### 3. **Connection Migration**
- QUIC: Seamless connection migration across networks
- TCP: Connections break on network changes

### 4. **Loss Recovery**
- QUIC: Per-stream loss recovery
- TCP: Connection-wide impact of packet loss

### 5. **Flow Control**
- QUIC: Stream-level and connection-level flow control
- TCP: Connection-level only

## Development

### Adding New Features

1. **IoT Sensors**: Add new sensor types in `internal/iot/handler.go`
2. **Streaming Formats**: Extend formats in `internal/streaming/handler.go`
3. **Benchmark Tests**: Add test types in `internal/benchmark/benchmarker.go`

### Testing

```bash
# Run unit tests
go test ./...

# Run integration tests
go test -tags=integration ./...

# Run benchmarks
go test -bench=. ./...
```

### Code Structure

- **Handler Pattern**: HTTP handlers for different protocols
- **Middleware**: Common functionality like logging, metrics
- **Configuration**: Environment-based configuration
- **Error Handling**: Structured error responses
- **Logging**: Structured logging with context

## Monitoring and Metrics

### Available Metrics

- Request latency (min, max, avg, p95, p99)
- Throughput (requests/second)
- Bandwidth utilization (Mbps)
- Success/failure rates
- Connection statistics
- Stream statistics

### Log Format

```json
{
  "timestamp": "2024-08-20T10:52:00Z",
  "level": "info",
  "message": "Request processed",
  "protocol": "quic",
  "endpoint": "/iot/sensor",
  "latency_ms": 12.34,
  "status": 200
}
```

## Troubleshooting

### Common Issues

1. **Certificate Errors**: The system uses self-signed certificates for testing
   - Solution: Use `-k` flag with curl or set `InsecureSkipVerify: true`

2. **Port Conflicts**: Default ports 8443 (QUIC) and 8080 (TCP)
   - Solution: Use different ports with `-addr` flag

3. **Docker Network Issues**: Services not connecting
   - Solution: Check network configuration in docker-compose.yml

### Debug Commands

```bash
# Check server health
curl -k https://localhost:8443/health

# List available streams
curl -k https://localhost:8443/stream/list

# Get device list
curl -k https://localhost:8443/iot/devices

# Check Docker services
docker-compose ps
docker-compose logs server
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make changes with tests
4. Submit a pull request

## License

MIT License - see LICENSE file for details.

## References

- [QUIC Protocol Specification](https://tools.ietf.org/html/rfc9000)
- [HTTP/3 Specification](https://tools.ietf.org/html/rfc9114)
- [quic-go Library](https://github.com/quic-go/quic-go)
- [QUIC Performance Studies](https://blog.cloudflare.com/the-road-to-quic/)