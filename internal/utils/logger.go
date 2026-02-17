package utils

import (
	"fmt"
	"io"
	"os"
	"time"
)

// Logger 简单日志工具
type Logger struct {
	writer   io.Writer
	debugLog bool
}

// NewLogger 创建新的logger
func NewLogger(debugMode bool) *Logger {
	return &Logger{
		writer:   os.Stdout,
		debugLog: debugMode,
	}
}

// Info 输出信息日志
func (l *Logger) Info(msg string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Fprintf(l.writer, "[%s] ℹ️  INFO: %s\n", timestamp, fmt.Sprintf(msg, args...))
}

// Warn 输出警告日志
func (l *Logger) Warn(msg string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Fprintf(l.writer, "[%s] ⚠️  WARN: %s\n", timestamp, fmt.Sprintf(msg, args...))
}

// Error 输出错误日志
func (l *Logger) Error(msg string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Fprintf(os.Stderr, "[%s] ❌ ERROR: %s\n", timestamp, fmt.Sprintf(msg, args...))
}

// Debug 输出调试日志（仅在调试模式下输出）
func (l *Logger) Debug(msg string, args ...interface{}) {
	if !l.debugLog {
		return
	}
	timestamp := time.Now().Format("15:04:05")
	fmt.Fprintf(l.writer, "[%s] 🐛 DEBUG: %s\n", timestamp, fmt.Sprintf(msg, args...))
}

// Success 输出成功日志
func (l *Logger) Success(msg string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Fprintf(l.writer, "[%s] ✅ SUCCESS: %s\n", timestamp, fmt.Sprintf(msg, args...))
}
