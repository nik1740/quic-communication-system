package config

import (
	"time"

	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	IoT       IoTConfig       `mapstructure:"iot"`
	Streaming StreamingConfig `mapstructure:"streaming"`
	Benchmark BenchmarkConfig `mapstructure:"benchmark"`
	Logging   LoggingConfig   `mapstructure:"logging"`
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	QUICAddr string        `mapstructure:"quic_addr"`
	TCPAddr  string        `mapstructure:"tcp_addr"`
	CertFile string        `mapstructure:"cert_file"`
	KeyFile  string        `mapstructure:"key_file"`
	Timeout  time.Duration `mapstructure:"timeout"`
}

// IoTConfig holds IoT-specific configuration
type IoTConfig struct {
	SensorUpdateInterval time.Duration `mapstructure:"sensor_update_interval"`
	MaxSensors          int           `mapstructure:"max_sensors"`
	DataRetention       time.Duration `mapstructure:"data_retention"`
}

// StreamingConfig holds streaming-specific configuration
type StreamingConfig struct {
	VideoDir      string   `mapstructure:"video_dir"`
	Qualities     []string `mapstructure:"qualities"`
	SegmentLength int      `mapstructure:"segment_length"`
	BufferSize    int      `mapstructure:"buffer_size"`
}

// BenchmarkConfig holds benchmarking configuration
type BenchmarkConfig struct {
	Duration    time.Duration `mapstructure:"duration"`
	Connections int           `mapstructure:"connections"`
	DataSize    int           `mapstructure:"data_size"`
	ResultsDir  string        `mapstructure:"results_dir"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	File   string `mapstructure:"file"`
}

// LoadConfig loads configuration from file or environment variables
func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configPath)
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	// Set defaults
	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func setDefaults() {
	// Server defaults
	viper.SetDefault("server.quic_addr", ":8443")
	viper.SetDefault("server.tcp_addr", ":8080")
	viper.SetDefault("server.cert_file", "certs/server.crt")
	viper.SetDefault("server.key_file", "certs/server.key")
	viper.SetDefault("server.timeout", "30s")

	// IoT defaults
	viper.SetDefault("iot.sensor_update_interval", "1s")
	viper.SetDefault("iot.max_sensors", 100)
	viper.SetDefault("iot.data_retention", "24h")

	// Streaming defaults
	viper.SetDefault("streaming.video_dir", "./test-videos")
	viper.SetDefault("streaming.qualities", []string{"480p", "720p", "1080p"})
	viper.SetDefault("streaming.segment_length", 10)
	viper.SetDefault("streaming.buffer_size", 4096)

	// Benchmark defaults
	viper.SetDefault("benchmark.duration", "60s")
	viper.SetDefault("benchmark.connections", 10)
	viper.SetDefault("benchmark.data_size", 1024)
	viper.SetDefault("benchmark.results_dir", "./benchmark-results")

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.file", "")
}