package utils

import (
	"fmt"
	"io"
	"os"
	"time"
)

// Logger is a tiny console logger.
type Logger struct {
	writer   io.Writer
	debugLog bool
}

// NewLogger creates a Logger. If debugMode is true, Debug logs are enabled.
func NewLogger(debugMode bool) *Logger {
	return &Logger{
		writer:   os.Stdout,
		debugLog: debugMode,
	}
}

// Info prints an informational message.
func (l *Logger) Info(msg string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Fprintf(l.writer, "[%s] ℹ️  INFO: %s\n", timestamp, fmt.Sprintf(msg, args...))
}

// Warn prints a warning message.
func (l *Logger) Warn(msg string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Fprintf(l.writer, "[%s] ⚠️  WARN: %s\n", timestamp, fmt.Sprintf(msg, args...))
}

// Error prints an error message to stderr.
func (l *Logger) Error(msg string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Fprintf(os.Stderr, "[%s] ❌ ERROR: %s\n", timestamp, fmt.Sprintf(msg, args...))
}

// Debug prints a debug message when debug mode is enabled.
func (l *Logger) Debug(msg string, args ...interface{}) {
	if !l.debugLog {
		return
	}
	timestamp := time.Now().Format("15:04:05")
	fmt.Fprintf(l.writer, "[%s] 🐛 DEBUG: %s\n", timestamp, fmt.Sprintf(msg, args...))
}

// Success prints a success message.
func (l *Logger) Success(msg string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Fprintf(l.writer, "[%s] ✅ SUCCESS: %s\n", timestamp, fmt.Sprintf(msg, args...))
}
