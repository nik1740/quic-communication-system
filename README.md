# QUIC Communication System

A comprehensive QUIC-based communication system demonstrating the advantages of HTTP/3 and QUIC protocol over traditional TCP/TLS for IoT devices and video streaming applications.

## 🚀 Features

### Core QUIC Implementation
- ✅ QUIC server with HTTP/3 support
- ✅ Connection multiplexing and migration support  
- ✅ Stream prioritization and flow control
- ✅ Self-signed certificate generation
- ✅ Comprehensive logging and monitoring

### IoT Communication Module
- ✅ Lightweight sensor data transmission protocols
- ✅ Command/response patterns for device control
- ✅ Support for multiple sensor types (temperature, humidity, motion, light)
- ✅ Real-time data streaming
- ✅ Device registration and management

### Video Streaming Module
- ✅ HTTP/3-based video streaming server
- ✅ Adaptive bitrate streaming (480p, 720p, 1080p)
- ✅ HLS manifest generation
- ✅ Segment-based video delivery
- ✅ Multi-quality stream support

### TCP/TLS Comparison
- ✅ Equivalent TCP/TLS server implementation
- ✅ Same API endpoints for fair comparison
- ✅ Performance baseline for benchmarking

### Benchmarking Framework
- ✅ Automated performance testing suite
- ✅ Latency, throughput, and reliability metrics
- ✅ Concurrent connection testing
- ✅ Results collection and JSON export
- ✅ QUIC vs TCP/TLS comparison

### Infrastructure
- ✅ Docker containerization
- ✅ Docker Compose orchestration
- ✅ Build and deployment scripts
- ✅ Configuration management

## 📁 Project Structure

```
quic-communication-system/
├── cmd/                    # Main applications
│   ├── server/            # QUIC server
│   ├── iot-client/        # IoT device simulator
│   ├── streaming-client/  # Video streaming client
│   └── benchmark/         # Benchmarking tools
├── internal/              # Internal packages
│   ├── quic/             # Core QUIC implementation
│   ├── iot/              # IoT protocol handlers
│   ├── streaming/        # Video streaming protocols
│   ├── tcp/              # TCP/TLS comparison
│   └── benchmark/        # Performance testing
├── pkg/                  # Public APIs
├── docker/               # Container configurations
├── scripts/              # Build and deployment scripts
├── test/                 # Test data and scenarios
├── docs/                 # Documentation
├── config.yaml           # Configuration file
├── docker-compose.yml    # Service orchestration
└── Dockerfile            # Container definition
```

## 🛠️ Prerequisites

- **Go 1.19+** - [Install Go](https://golang.org/doc/install)
- **Docker** - [Install Docker](https://docs.docker.com/get-docker/)
- **Docker Compose** - [Install Docker Compose](https://docs.docker.com/compose/install/)

## 🚀 Quick Start

### Option 1: Docker (Recommended)

1. **Clone the repository:**
   ```bash
   git clone https://github.com/nik1740/quic-communication-system.git
   cd quic-communication-system
   ```

2. **Deploy with Docker Compose:**
   ```bash
   ./scripts/deploy.sh
   ```

3. **Test the endpoints:**
   ```bash
   # Test IoT endpoint
   curl -k https://localhost:8443/iot/devices
   
   # Test streaming endpoint
   curl -k https://localhost:8443/stream/list
   
   # Health check
   curl -k https://localhost:8443/health
   ```

### Option 2: Local Build

1. **Build the project:**
   ```bash
   ./scripts/build.sh
   ```

2. **Start the server:**
   ```bash
   ./bin/server
   ```

3. **Run clients (in separate terminals):**
   ```bash
   # IoT client simulator
   ./bin/iot-client --server https://localhost:8443 --type temperature
   
   # Streaming client
   ./bin/streaming-client --server https://localhost:8443 --stream demo1 --quality 720p
   
   # Benchmark tool
   ./bin/benchmark --server https://localhost:8443 --test iot --duration 60
   ```

## 📊 API Endpoints

### Server Health
- `GET /health` - Server health check

### IoT Endpoints
- `GET /iot/devices` - List registered devices
- `POST /iot/devices` - Register new device
- `GET /iot/data?limit=100` - Get recent sensor readings
- `POST /iot/data` - Submit sensor data
- `POST /iot/command` - Send device command
- `GET /iot/health` - IoT service health

### Streaming Endpoints
- `GET /stream/list` - List available streams
- `GET /stream/info/{id}` - Get stream information
- `GET /stream/manifest/{id}/master.m3u8` - Master playlist
- `GET /stream/manifest/{id}/{quality}.m3u8` - Quality playlist
- `GET /stream/video/{id}/{quality}/{segment}.ts` - Video segments
- `GET /stream/health` - Streaming service health

## 🔧 Configuration

Edit `config.yaml` to customize settings:

```yaml
server:
  quic_addr: ":8443"          # QUIC server address
  tcp_addr: ":8080"           # TCP server address
  cert_file: "./certs/server.crt"
  key_file: "./certs/server.key"
  timeout: "30s"

iot:
  sensor_update_interval: "1s"
  max_sensors: 100
  data_retention: "24h"

streaming:
  video_dir: "./test-videos"
  qualities: ["480p", "720p", "1080p"]
  segment_length: 10
  buffer_size: 4096

benchmark:
  duration: "60s"
  connections: 10
  data_size: 1024
  results_dir: "./benchmark-results"

logging:
  level: "info"              # debug, info, warn, error
  format: "json"             # json, text
  file: ""                   # empty for stdout
```

## 📈 Benchmarking

### Running Benchmarks

```bash
# QUIC IoT benchmark
./bin/benchmark --server https://localhost:8443 --test iot --duration 60 --connections 10 --quic

# TCP IoT benchmark
./bin/benchmark --server http://localhost:8080 --test iot --duration 60 --connections 10 --quic=false

# Streaming benchmark
./bin/benchmark --server https://localhost:8443 --test streaming --duration 60 --connections 5

# Mixed workload
./bin/benchmark --server https://localhost:8443 --test mixed --duration 120 --connections 15
```

### Using Docker for Benchmarks

```bash
# Run QUIC and TCP benchmarks
docker-compose --profile benchmark up

# Results will be saved in ./benchmark-results/
```

### Sample Benchmark Results

```json
{
  "test_type": "iot",
  "protocol": "QUIC",
  "duration": "1m0s",
  "total_requests": 3000,
  "successful_requests": 2998,
  "failed_requests": 2,
  "requests_per_second": 49.97,
  "avg_response_time": "15.2ms",
  "min_response_time": "8.1ms", 
  "max_response_time": "145.7ms",
  "p50_response_time": "12.4ms",
  "p95_response_time": "28.9ms",
  "p99_response_time": "67.3ms",
  "throughput_mbps": 2.34,
  "total_bytes": 156738942
}
```

## 🔬 Testing Scenarios

### IoT Device Simulation

```bash
# Temperature sensor
./bin/iot-client --server https://localhost:8443 --type temperature --interval 2 --duration 300

# Multiple sensor types
./bin/iot-client --type humidity --interval 1 &
./bin/iot-client --type motion --interval 5 &
./bin/iot-client --type light --interval 3 &
```

### Video Streaming Test

```bash
# Download stream segments
./bin/streaming-client --server https://localhost:8443 --stream demo1 --quality 1080p --output ./downloads

# Test different qualities
for quality in 480p 720p 1080p; do
  ./bin/streaming-client --quality $quality --output ./downloads/$quality
done
```

## 🐳 Docker Commands

```bash
# Build and start all services
docker-compose up -d

# View logs
docker-compose logs -f quic-server
docker-compose logs -f iot-simulator

# Stop services
docker-compose down

# Run benchmark profile
docker-compose --profile benchmark up

# Scale IoT simulators
docker-compose up -d --scale iot-simulator=3

# Shell access
docker-compose exec quic-server sh
```

## 🔍 Monitoring & Logs

### Log Locations
- **Local**: `./logs/`
- **Docker**: View with `docker-compose logs [service]`

### Health Checks
- Server: `https://localhost:8443/health`
- IoT Service: `https://localhost:8443/iot/health`
- Streaming Service: `https://localhost:8443/stream/health`

### Metrics
- Request/response times
- Throughput measurements
- Connection statistics
- Error rates
- Data transfer volumes

## 🎯 Performance Goals

This implementation demonstrates:

- **Faster Connection Establishment**: QUIC's 0-RTT and 1-RTT handshakes
- **Better Multiplexing**: No head-of-line blocking
- **Connection Migration**: Seamless network switching
- **Improved Reliability**: Built-in error recovery
- **Reduced Latency**: Optimized for modern networks

### Expected Benefits
- 20-50% reduction in connection establishment time
- 10-30% improvement in throughput under packet loss
- Better performance on high-latency networks
- Improved mobile network performance

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Troubleshooting

### Common Issues

**Certificate errors:**
```bash
# Regenerate certificates
rm -rf certs/
./bin/server  # Will auto-generate new certificates
```

**Port conflicts:**
```bash
# Check port usage
lsof -i :8443
lsof -i :8080

# Use different ports
./bin/server --quic-addr :9443 --tcp-addr :9080
```

**Docker issues:**
```bash
# Reset Docker environment
docker-compose down -v
docker system prune -f
docker-compose up --build
```

### Getting Help

- Check the logs: `docker-compose logs [service]`
- Verify endpoints: `curl -k https://localhost:8443/health`
- Review configuration: `cat config.yaml`
- Test connectivity: `telnet localhost 8443`

## 🔗 References

- [QUIC Protocol (RFC 9000)](https://tools.ietf.org/html/rfc9000)
- [HTTP/3 (RFC 9114)](https://tools.ietf.org/html/rfc9114)
- [quic-go Library](https://github.com/quic-go/quic-go)
- [Docker Documentation](https://docs.docker.com/)
- [Go Documentation](https://golang.org/doc/)