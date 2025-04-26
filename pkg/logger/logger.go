package logger

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// LogLevel defines the severity level for log events.
type LogLevel string

const (
	// DebugLevel indicates detailed tracing information, typically only useful during development.
	DebugLevel LogLevel = "debug"
	// InfoLevel indicates general operational information.
	InfoLevel LogLevel = "info"
	// WarnLevel indicates potentially harmful situations or unexpected events.
	WarnLevel LogLevel = "warn"
	// ErrorLevel indicates error events that might still allow the application to continue running.
	ErrorLevel LogLevel = "error"
	// FatalLevel indicates severe error events that will presumably lead the application to abort.
	FatalLevel LogLevel = "fatal"
)

// Init initializes the global logger provided by the zerolog library.
// It configures the logger to output JSON formatted logs to stderr with Unix timestamps.
// This should typically be called once at application startup.
func Init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	// Use JSON output instead of ConsoleWriter for structured logs
	// log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	log.Logger = log.Output(os.Stderr)
}

// LogEvent represents the structure of a log entry, primarily used for understanding the JSON output.
// This struct itself is not directly used for logging via the exported functions.
type LogEvent struct {
	// Level is the severity level of the log event (e.g., "info", "error").
	Level LogLevel `json:"level"`
	// Message is the main human-readable log message.
	Message string `json:"message"`
	// Component indicates the part of the application that generated the log (e.g., "transcoder", "downloader").
	Component string `json:"component"`
	// Data contains optional additional structured key-value data associated with the log event.
	Data map[string]interface{} `json:"data,omitempty"`
}

// Log is the core logging function.
// It takes the level, message, component, and optional data, and logs it using the globally configured zerolog logger.
// Use the specific level functions (Debug, Info, Warn, Error, Fatal) instead of calling Log directly.
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

// Debug logs a message at the Debug level with the specified component and optional data.
func Debug(message, component string, data map[string]interface{}) {
	Log(DebugLevel, message, component, data)
}

// Info logs a message at the Info level with the specified component and optional data.
func Info(message, component string, data map[string]interface{}) {
	Log(InfoLevel, message, component, data)
}

// Warn logs a message at the Warn level with the specified component and optional data.
func Warn(message, component string, data map[string]interface{}) {
	Log(WarnLevel, message, component, data)
}

// Error logs a message at the Error level with the specified component and optional data.
func Error(message, component string, data map[string]interface{}) {
	Log(ErrorLevel, message, component, data)
}

// Fatal logs a message at the Fatal level with the specified component and optional data,
// and then calls os.Exit(1).
func Fatal(message, component string, data map[string]interface{}) {
	Log(FatalLevel, message, component, data)
}
