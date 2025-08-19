package tui

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// DebugLogger handles debug logging for TUI operations
type DebugLogger struct {
	enabled bool
	logger  *log.Logger
	file    *os.File
}

// NewDebugLogger creates a new debug logger
func NewDebugLogger(enabled bool) *DebugLogger {
	if !enabled {
		return &DebugLogger{enabled: false}
	}

	// Create .goingenv directory if it doesn't exist
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return &DebugLogger{enabled: false}
	}

	debugDir := filepath.Join(homeDir, ".goingenv", "debug")
	if err := os.MkdirAll(debugDir, 0755); err != nil {
		return &DebugLogger{enabled: false}
	}

	// Create debug log file with timestamp
	timestamp := time.Now().Format("20060102_150405")
	logPath := filepath.Join(debugDir, fmt.Sprintf("tui_debug_%s.log", timestamp))
	
	file, err := os.Create(logPath)
	if err != nil {
		return &DebugLogger{enabled: false}
	}

	logger := log.New(file, "", log.LstdFlags|log.Lmicroseconds)
	
	return &DebugLogger{
		enabled: true,
		logger:  logger,
		file:    file,
	}
}

// Log writes a debug message if logging is enabled
func (d *DebugLogger) Log(format string, args ...interface{}) {
	if !d.enabled || d.logger == nil {
		return
	}
	
	d.logger.Printf("[DEBUG] "+format, args...)
}

// LogScreen logs screen transitions
func (d *DebugLogger) LogScreen(from, to Screen) {
	d.Log("Screen transition: %s -> %s", from, to)
}

// LogKeypress logs key press events
func (d *DebugLogger) LogKeypress(key string, screen Screen) {
	d.Log("Keypress: %s (screen: %s)", key, screen)
}

// LogMessage logs message events
func (d *DebugLogger) LogMessage(msgType, content string) {
	d.Log("Message [%s]: %s", msgType, content)
}

// LogError logs error events
func (d *DebugLogger) LogError(operation string, err error) {
	d.Log("Error in %s: %v", operation, err)
}

// LogOperation logs general operations
func (d *DebugLogger) LogOperation(operation, details string) {
	d.Log("Operation: %s - %s", operation, details)
}

// LogModelUpdate logs model updates with detailed information
func (d *DebugLogger) LogModelUpdate(updateType string, details map[string]interface{}) {
	d.Log("Model Update [%s]: %+v", updateType, details)
}

// LogProgress logs progress updates
func (d *DebugLogger) LogProgress(operation string, progress float64) {
	d.Log("Progress [%s]: %.2f%%", operation, progress*100)
}

// LogFileOperation logs file operations
func (d *DebugLogger) LogFileOperation(operation, path string, size int64) {
	d.Log("File Operation [%s]: %s (size: %d bytes)", operation, path, size)
}

// Close closes the debug logger and its file
func (d *DebugLogger) Close() {
	if d.enabled && d.file != nil {
		d.Log("Debug session ended")
		d.file.Close()
	}
}

// IsEnabled returns whether debug logging is enabled
func (d *DebugLogger) IsEnabled() bool {
	return d.enabled
}

// GetLogPath returns the path to the current log file
func (d *DebugLogger) GetLogPath() string {
	if !d.enabled || d.file == nil {
		return ""
	}
	return d.file.Name()
}