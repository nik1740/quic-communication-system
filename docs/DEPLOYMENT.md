# Deployment Guide

## System Requirements

### Minimum Requirements
- CPU: 2 cores
- RAM: 4 GB
- Storage: 10 GB free space
- Network: UDP port 4433 accessible

### Recommended Requirements
- CPU: 4+ cores
- RAM: 8+ GB
- Storage: 20+ GB free space
- Network: Low-latency, high-bandwidth connection

## Installation Methods

### Method 1: Binary Installation

1. Download the latest release from GitHub
2. Extract the archive
3. Run the build script:
```bash
./scripts/build.sh
```

### Method 2: Docker Deployment

1. Clone the repository
2. Run the deployment script:
```bash
./scripts/deploy.sh
```

### Method 3: Manual Build

1. Install Go 1.21+
2. Clone the repository
3. Build manually:
```bash
go mod download
go build -o bin/server ./cmd/server
go build -o bin/iot-client ./cmd/iot-client
go build -o bin/streaming-client ./cmd/streaming-client
go build -o bin/benchmark ./cmd/benchmark
```

## Configuration

### Environment Variables

- `LOG_LEVEL`: Set logging level (debug, info, warn, error)
- `QUIC_ADDR`: Server bind address (default: localhost:4433)
- `CONFIG_FILE`: Path to YAML configuration file

### Configuration File

Create `config.yaml`:

```yaml
server:
  host: "0.0.0.0"
  port: 4433
  read_timeout: "30s"
  write_timeout: "30s"

iot:
  max_devices: 1000
  heartbeat_timeout: "60s"
  buffer_size: 8192

streaming:
  max_streams: 100
  bitrates: [500, 1000, 2000, 4000]
  chunk_size_kb: 64
  max_resolution: "1080p"

log_level: "info"
```

## Production Deployment

### Load Balancing

For high availability, deploy multiple server instances behind a UDP load balancer:

```
Internet → UDP Load Balancer → QUIC Server Instance 1
                            → QUIC Server Instance 2
                            → QUIC Server Instance 3
```

### Monitoring Setup

1. Deploy Prometheus for metrics collection
2. Configure Grafana dashboards
3. Set up alerting rules

Example Prometheus configuration:
```yaml
scrape_configs:
  - job_name: 'quic-server'
    static_configs:
      - targets: ['localhost:8080']
```

### Security Considerations

1. **TLS Certificates**: Use proper certificates in production
2. **Firewall Rules**: Restrict access to necessary ports only
3. **Rate Limiting**: Implement rate limiting for DoS protection
4. **Authentication**: Add authentication for administrative endpoints

### Backup and Recovery

1. **Configuration Backup**: Regularly backup configuration files
2. **Metrics Data**: Backup Prometheus data for historical analysis
3. **Log Archival**: Implement log rotation and archival

## Scaling Guidelines

### Vertical Scaling
- Increase CPU cores for better concurrent connection handling
- Increase RAM for larger device/stream counts
- Use SSD storage for better I/O performance

### Horizontal Scaling
- Deploy multiple server instances
- Use connection-based load balancing
- Implement proper service discovery

### Performance Tuning

#### System-Level Tuning

```bash
# Increase UDP buffer sizes
echo 'net.core.rmem_max = 134217728' >> /etc/sysctl.conf
echo 'net.core.rmem_default = 134217728' >> /etc/sysctl.conf
echo 'net.core.wmem_max = 134217728' >> /etc/sysctl.conf
echo 'net.core.wmem_default = 134217728' >> /etc/sysctl.conf

# Apply changes
sysctl -p
```

#### Application-Level Tuning

1. **Connection Limits**: Adjust `max_devices` and `max_streams`
2. **Buffer Sizes**: Tune `buffer_size` and `chunk_size_kb`
3. **Timeouts**: Optimize `read_timeout` and `heartbeat_timeout`

## Monitoring and Alerting

### Key Metrics to Monitor

1. **Connection Metrics**
   - Active connections count
   - Connection establishment rate
   - Connection errors

2. **Performance Metrics**
   - Message throughput
   - Latency percentiles
   - Error rates

3. **Resource Metrics**
   - CPU usage
   - Memory usage
   - Network I/O

### Sample Alerting Rules

```yaml
groups:
  - name: quic-server
    rules:
      - alert: HighConnectionCount
        expr: quic_connections_active > 800
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: High number of active connections

      - alert: HighErrorRate
        expr: rate(quic_errors_total[5m]) > 0.1
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: High error rate detected
```

## Troubleshooting

### Common Issues

1. **Connection Timeouts**
   - Check firewall rules
   - Verify UDP port accessibility
   - Review network configuration

2. **High Memory Usage**
   - Check for connection leaks
   - Review buffer size configuration
   - Monitor garbage collection

3. **Performance Issues**
   - Check system resource usage
   - Review network latency
   - Analyze connection patterns

### Debug Commands

```bash
# Check server status
curl -s http://localhost:8080/health

# View metrics
curl -s http://localhost:8080/metrics

# Check logs
tail -f logs/server.log

# Test connectivity
./bin/iot-client -server localhost:4433 -duration 10s
```

### Log Analysis

Key log patterns to watch:

- `Failed to accept connection`: Network or resource issues
- `Stream handler error`: Application-level errors
- `Device went offline`: IoT connectivity issues
- `High temperature alert`: Application-specific alerts