# API Documentation

## QUIC Communication System API

### IoT Protocol

The IoT protocol uses QUIC streams with a 4-byte protocol header `IOT\x00`.

#### Message Format

```json
{
  "type": "sensor_data|command|response|heartbeat",
  "device_id": "string",
  "timestamp": "2024-01-01T00:00:00Z",
  "data": {},
  "reliable": true|false
}
```

#### Sensor Data

```json
{
  "type": "sensor_data",
  "device_id": "temp-sensor-01",
  "timestamp": "2024-01-01T00:00:00Z",
  "data": {
    "device_id": "temp-sensor-01",
    "type": "temperature",
    "value": 25.5,
    "unit": "°C",
    "timestamp": "2024-01-01T00:00:00Z",
    "quality": 0.95
  },
  "reliable": false
}
```

#### Command

```json
{
  "type": "command",
  "device_id": "light-01",
  "timestamp": "2024-01-01T00:00:00Z",
  "data": {
    "id": "cmd-123",
    "device_id": "light-01",
    "type": "set_brightness",
    "params": {"brightness": 80},
    "priority": 5
  },
  "reliable": true
}
```

#### Heartbeat

```json
{
  "type": "heartbeat",
  "device_id": "device-01",
  "timestamp": "2024-01-01T00:00:00Z",
  "data": {
    "status": "alive",
    "uptime": 1640995200
  },
  "reliable": true
}
```

### Streaming Protocol

The streaming protocol uses QUIC streams with a 4-byte protocol header `STRM`.

#### Stream Request

```json
{
  "stream_id": "my-stream",
  "quality": {
    "name": "720p",
    "width": 1280,
    "height": 720,
    "bitrate": 2000,
    "framerate": 30
  },
  "buffer_size": 30,
  "start_time": "2024-01-01T00:00:00Z",
  "adaptive_bitrate": true
}
```

#### Stream Response

```json
{
  "stream_id": "my-stream",
  "status": "accepted|error",
  "available_qualities": [
    {"name": "360p", "width": 640, "height": 360, "bitrate": 500, "framerate": 30},
    {"name": "720p", "width": 1280, "height": 720, "bitrate": 2000, "framerate": 30}
  ],
  "chunk_duration": "2s"
}
```

#### Video Chunk Format

1. Header size (4 bytes, big-endian uint32)
2. JSON header with metadata
3. Raw video data

Header JSON:
```json
{
  "stream_id": "my-stream",
  "sequence_num": 123,
  "timestamp": "2024-01-01T00:00:00Z",
  "quality": {"name": "720p", "width": 1280, "height": 720, "bitrate": 2000, "framerate": 30},
  "size": 65536,
  "is_keyframe": true,
  "duration": "2s"
}
```

## Configuration Format

### Server Configuration

```yaml
server:
  host: "0.0.0.0"          # Bind address
  port: 4433               # QUIC port
  read_timeout: "30s"      # Read timeout
  write_timeout: "30s"     # Write timeout

iot:
  max_devices: 1000        # Maximum concurrent devices
  heartbeat_timeout: "60s" # Device timeout
  buffer_size: 8192        # Message buffer size

streaming:
  max_streams: 100         # Maximum concurrent streams
  bitrates: [500, 1000, 2000, 4000]  # Available bitrates (kbps)
  chunk_size_kb: 64        # Video chunk size
  max_resolution: "1080p"  # Maximum resolution
  codec_profiles: ["h264", "h265"]  # Supported codecs

benchmark:
  test_duration: "60s"     # Default test duration
  sample_interval: "1s"    # Metrics sample interval
  packet_loss_rate: 0.0    # Simulated packet loss
  latency_ms: 0           # Simulated latency
  bandwidth_mbps: 100     # Simulated bandwidth

log_level: "info"         # Log level (debug, info, warn, error)
```

## Metrics and Monitoring

### Prometheus Metrics

The server exposes the following metrics:

- `quic_connections_total`: Total number of QUIC connections
- `quic_connections_active`: Currently active connections
- `quic_streams_total`: Total number of streams
- `quic_bytes_sent_total`: Total bytes sent
- `quic_bytes_received_total`: Total bytes received
- `iot_devices_total`: Total registered IoT devices
- `iot_messages_total`: Total IoT messages processed
- `streaming_sessions_total`: Total streaming sessions
- `streaming_chunks_sent_total`: Total video chunks sent

### Health Checks

- `GET /health`: Server health status
- `GET /metrics`: Prometheus metrics endpoint
- `GET /stats`: JSON statistics endpoint

## Performance Benchmarks

### Test Types

1. **Latency Test**: Measures connection establishment and message round-trip time
2. **Throughput Test**: Measures maximum data transfer rate
3. **IoT Test**: Simulates realistic IoT device communication patterns
4. **Streaming Test**: Simulates video streaming with adaptive bitrate

### Benchmark Results Format

```json
{
  "test_type": "latency",
  "protocol": "quic",
  "duration": "30s",
  "connections": 10,
  "bytes_transferred": 1048576,
  "messages_count": 1000,
  "avg_latency": "5ms",
  "min_latency": "2ms",
  "max_latency": "15ms",
  "p95_latency": "12ms",
  "p99_latency": "14ms",
  "throughput_mbps": 125.5,
  "packet_loss_rate": 0.001,
  "connection_time": "8ms",
  "error_count": 2,
  "timestamp": "2024-01-01T00:00:00Z"
}
```