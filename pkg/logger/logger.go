package logger

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// LogLevel represents the severity level of a log
type LogLevel string

const (
	// DebugLevel logs detailed information for debugging
	DebugLevel LogLevel = "debug"
	// InfoLevel logs general information
	InfoLevel LogLevel = "info"
	// WarnLevel logs warnings
	WarnLevel LogLevel = "warn"
	// ErrorLevel logs errors
	ErrorLevel LogLevel = "error"
	// FatalLevel logs fatal errors
	FatalLevel LogLevel = "fatal"
)

// Init initializes the logger with JSON formatting
func Init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

// LogEvent represents a structured log event
type LogEvent struct {
	Level     LogLevel               `json:"level"`
	Message   string                 `json:"message"`
	Component string                 `json:"component"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// Log logs an event with the specified level
func Log(level LogLevel, message, component string, data map[string]interface{}) {
	logger := log.With().
		Str("component", component).
		Fields(data).
		Logger()

	switch level {
	case DebugLevel:
		logger.Debug().Msg(message)
	case InfoLevel:
		logger.Info().Msg(message)
	case WarnLevel:
		logger.Warn().Msg(message)
	case ErrorLevel:
		logger.Error().Msg(message)
	case FatalLevel:
		logger.Fatal().Msg(message)
	}
}

// Debug logs a debug event
func Debug(message, component string, data map[string]interface{}) {
	Log(DebugLevel, message, component, data)
}

// Info logs an info event
func Info(message, component string, data map[string]interface{}) {
	Log(InfoLevel, message, component, data)
}

// Warn logs a warning event
func Warn(message, component string, data map[string]interface{}) {
	Log(WarnLevel, message, component, data)
}

// Error logs an error event
func Error(message, component string, data map[string]interface{}) {
	Log(ErrorLevel, message, component, data)
}

// Fatal logs a fatal event and exits
func Fatal(message, component string, data map[string]interface{}) {
	Log(FatalLevel, message, component, data)
}
