package config

import (
	"crypto/tls"
	"time"
)

// ServerConfig holds the main server configuration
type ServerConfig struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	TLSConfig    *tls.Config   `yaml:"-"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

// IoTConfig holds IoT-specific configuration
type IoTConfig struct {
	MaxDevices       int           `yaml:"max_devices"`
	HeartbeatTimeout time.Duration `yaml:"heartbeat_timeout"`
	BufferSize       int           `yaml:"buffer_size"`
}

// StreamingConfig holds video streaming configuration
type StreamingConfig struct {
	MaxStreams     int      `yaml:"max_streams"`
	Bitrates       []int    `yaml:"bitrates"`
	ChunkSizeKB    int      `yaml:"chunk_size_kb"`
	MaxResolution  string   `yaml:"max_resolution"`
	CodecProfiles  []string `yaml:"codec_profiles"`
}

// BenchmarkConfig holds benchmarking configuration
type BenchmarkConfig struct {
	TestDuration    time.Duration `yaml:"test_duration"`
	SampleInterval  time.Duration `yaml:"sample_interval"`
	PacketLossRate  float64       `yaml:"packet_loss_rate"`
	LatencyMs       int           `yaml:"latency_ms"`
	BandwidthMbps   int           `yaml:"bandwidth_mbps"`
}

// Config holds the complete application configuration
type Config struct {
	Server      ServerConfig      `yaml:"server"`
	IoT         IoTConfig         `yaml:"iot"`
	Streaming   StreamingConfig   `yaml:"streaming"`
	Benchmark   BenchmarkConfig   `yaml:"benchmark"`
	LogLevel    string            `yaml:"log_level"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:         "localhost",
			Port:         4433,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		IoT: IoTConfig{
			MaxDevices:       1000,
			HeartbeatTimeout: 60 * time.Second,
			BufferSize:       8192,
		},
		Streaming: StreamingConfig{
			MaxStreams:    100,
			Bitrates:      []int{500, 1000, 2000, 4000},
			ChunkSizeKB:   64,
			MaxResolution: "1080p",
			CodecProfiles: []string{"h264", "h265"},
		},
		Benchmark: BenchmarkConfig{
			TestDuration:   60 * time.Second,
			SampleInterval: time.Second,
			PacketLossRate: 0.0,
			LatencyMs:      0,
			BandwidthMbps:  100,
		},
		LogLevel: "info",
	}
}