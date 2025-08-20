package logging

import (
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

// Logger wraps logrus logger
type Logger struct {
	*logrus.Logger
}

// NewLogger creates a new logger instance
func NewLogger(level, format, file string) (*Logger, error) {
	logger := logrus.New()

	// Set log level
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	logger.SetLevel(logLevel)

	// Set formatter
	if format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}

	// Set output
	if file != "" {
		f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}
		logger.SetOutput(io.MultiWriter(os.Stdout, f))
	} else {
		logger.SetOutput(os.Stdout)
	}

	return &Logger{Logger: logger}, nil
}

// WithFields creates a new logger entry with fields
func (l *Logger) WithFields(fields map[string]interface{}) *logrus.Entry {
	return l.Logger.WithFields(fields)
}

// WithComponent creates a new logger entry with component field
func (l *Logger) WithComponent(component string) *logrus.Entry {
	return l.Logger.WithField("component", component)
}

// WithProtocol creates a new logger entry with protocol field
func (l *Logger) WithProtocol(protocol string) *logrus.Entry {
	return l.Logger.WithField("protocol", protocol)
}