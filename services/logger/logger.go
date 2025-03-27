package logger

import "log"

// Level định nghĩa các mức độ log
type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	ErrorLevel
)

// Logger interface định nghĩa các phương thức logging
type Logger interface {
	Info(format string, v ...interface{})
	Error(format string, v ...interface{})
	Debug(format string, v ...interface{})
}

// DefaultLogger implement Logger interface sử dụng log package
type DefaultLogger struct {
	level Level
}

// NewDefaultLogger tạo một instance mới của DefaultLogger
func NewDefaultLogger(level Level) *DefaultLogger {
	return &DefaultLogger{
		level: level,
	}
}

// Info log thông tin
func (l *DefaultLogger) Info(format string, v ...interface{}) {
	if l.level <= InfoLevel {
		log.Printf("[INFO] "+format, v...)
	}
}

// Error log lỗi
func (l *DefaultLogger) Error(format string, v ...interface{}) {
	if l.level <= ErrorLevel {
		log.Printf("[ERROR] "+format, v...)
	}
}

// Debug log debug
func (l *DefaultLogger) Debug(format string, v ...interface{}) {
	if l.level <= DebugLevel {
		log.Printf("[DEBUG] "+format, v...)
	}
}
