# API Documentation

## QUIC Server API (Port 8443)

Base URL: `https://localhost:8443`

### Health Check

#### GET /health
Returns server health status.

**Response:**
```
Status: 200 OK
Content-Type: text/plain

QUIC server is running
```

---

## IoT Endpoints

### GET /iot/sensor
Get simulated sensor data.

**Response:**
```json
[
  {
    "device_id": "temp_01",
    "sensor_type": "temperature", 
    "value": 23.5,
    "unit": "celsius",
    "timestamp": "2024-08-20T10:30:00Z",
    "quality": "reliable"
  }
]
```

### POST /iot/sensor
Submit sensor readings from IoT devices.

**Request:**
```json
{
  "device_id": "temp_sensor_001",
  "sensor_type": "temperature",
  "value": 25.3,
  "unit": "celsius", 
  "timestamp": "2024-08-20T10:30:00Z",
  "quality": "reliable"
}
```

**Response:**
```json
{
  "status": "success",
  "message": "Sensor data received"
}
```

### POST /iot/command
Send commands to IoT devices.

**Request:**
```json
{
  "device_id": "device_001",
  "action": "set_temperature",
  "parameters": {
    "target_temp": 22.0,
    "mode": "auto"
  },
  "priority": "high"
}
```

**Response:**
```json
{
  "command_id": "cmd_1692531600",
  "status": "executed", 
  "message": "Command set_temperature executed on device device_001"
}
```

### GET /iot/devices
List connected IoT devices.

**Response:**
```json
{
  "devices": [
    {
      "id": "temp_01",
      "type": "temperature",
      "status": "online",
      "location": "room_a"
    }
  ],
  "count": 4
}
```

### GET /iot/simulate
Start IoT device simulation.

**Query Parameters:**
- `devices` (int): Number of devices to simulate (default: 10)
- `duration` (string): Simulation duration, e.g., "60s", "5m" (default: "60s")

**Response:**
```json
{
  "status": "started",
  "devices": 10,
  "duration": "60s"
}
```

---

## Video Streaming Endpoints

### GET /stream/list
List available video streams.

**Response:**
```json
{
  "streams": [
    {
      "stream_id": "stream_001",
      "title": "Sample Video 1",
      "duration": 120,
      "bitrates": [
        {
          "quality": "low",
          "bitrate": 500,
          "resolution": "640x360",
          "url": "/stream/chunk/stream_001?quality=low"
        }
      ],
      "format": "h264",
      "resolution": "1920x1080",
      "frame_rate": 30,
      "created_at": "2024-08-20T10:00:00Z"
    }
  ],
  "count": 2
}
```

### GET /stream/info/{stream_id}
Get metadata for a specific stream.

**Response:**
```json
{
  "stream_id": "stream_001",
  "title": "Stream stream_001",
  "duration": 300,
  "bitrates": [
    {
      "quality": "low",
      "bitrate": 500,
      "resolution": "640x360", 
      "url": "/stream/chunk/stream_001?quality=low"
    },
    {
      "quality": "medium",
      "bitrate": 1500,
      "resolution": "1280x720",
      "url": "/stream/chunk/stream_001?quality=medium"
    },
    {
      "quality": "high", 
      "bitrate": 3000,
      "resolution": "1920x1080",
      "url": "/stream/chunk/stream_001?quality=high"
    }
  ],
  "format": "h264",
  "resolution": "1920x1080",
  "frame_rate": 30,
  "created_at": "2024-08-20T10:00:00Z"
}
```

### GET /stream/chunk/{stream_id}
Get video chunk data.

**Query Parameters:**
- `quality` (string): Video quality - "low", "medium", "high", "ultra" (default: "medium")
- `chunk` (int): Chunk index (default: 0)

**Headers:**
- `Accept: application/json` - Return metadata instead of binary data

**Response (Binary):**
```
Content-Type: video/mp4
X-Stream-ID: stream_001
X-Chunk-Index: 0
X-Quality: medium

[Binary video data]
```

**Response (JSON):**
```json
{
  "stream_id": "stream_001",
  "chunk_index": 0,
  "quality": "medium",
  "size": 150000,
  "duration": 2000,
  "timestamp": 1692531600000,
  "is_keyframe": true
}
```

### GET /stream/stats/{stream_id}
Get streaming statistics.

**Response:**
```json
{
  "stream_id": "stream_001",
  "bytes_sent": 5000000,
  "chunks_sent": 150,
  "latency_ms": 25.5,
  "bandwidth_mbps": 15.3,
  "packet_loss_percent": 0.5,
  "active_clients": 10,
  "uptime_seconds": 3600
}
```

### GET /stream/live
Live video stream using Server-Sent Events.

**Response:**
```
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive

data: {"type":"frame","timestamp":1692531600000,"frame_id":0,"size":45000,"quality":"medium"}

data: {"type":"frame","timestamp":1692531601000,"frame_id":1,"size":47000,"quality":"medium"}
```

---

## TCP Server API (Port 8080)

Base URL: `https://localhost:8080`

The TCP server provides identical endpoints to the QUIC server for performance comparison.

### Additional Benchmark Endpoint

#### GET /benchmark/
Get connection information for benchmarking.

**Response:**
```json
{
  "protocol": "TCP/TLS",
  "connection": "HTTP/1.1 or HTTP/2", 
  "timestamp": 1692531600,
  "server": "tcp-comparison"
}
```

#### POST /benchmark/
Echo test for latency measurement.

**Request:**
```
[Binary payload for latency testing]
```

**Response:**
```json
{
  "protocol": "TCP/TLS",
  "bytes_read": 1024,
  "latency_ns": 5000000,
  "latency_ms": 5.0,
  "timestamp": 1692531600
}
```

---

## Error Responses

All endpoints return structured error responses:

```json
{
  "error": "Invalid request",
  "code": "INVALID_REQUEST",
  "details": "Missing required parameter: device_id"
}
```

### Common Status Codes

- `200` - OK
- `400` - Bad Request
- `404` - Not Found  
- `405` - Method Not Allowed
- `500` - Internal Server Error

---

## Rate Limiting

The server implements basic rate limiting:
- 100 requests per minute per IP for IoT endpoints
- 50 requests per minute per IP for streaming endpoints
- No rate limiting for health check

---

## Authentication

Currently, no authentication is required for testing purposes. In production:
- API keys for IoT devices
- JWT tokens for streaming clients
- mTLS for high-security scenarios

---

## WebSocket Support

Future versions will include WebSocket support for:
- Real-time IoT command/control
- Live streaming protocols
- Bidirectional communication patterns