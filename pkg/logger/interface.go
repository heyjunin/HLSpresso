package logger

// Logger defines a standard logging interface used throughout HLSpresso.
// This allows different logging implementations to be potentially swapped in.
// Log messages include a level (Debug, Info, etc.), a message string,
// the component generating the log, and optional structured data.
type Logger interface {
	// Debug logs a message at the Debug level.
	Debug(message string, component string, data map[string]interface{})
	// Info logs a message at the Info level.
	Info(message string, component string, data map[string]interface{})
	// Warn logs a message at the Warning level.
	Warn(message string, component string, data map[string]interface{})
	// Error logs a message at the Error level.
	Error(message string, component string, data map[string]interface{})
	// Fatal logs a message at the Fatal level and typically terminates the application.
	Fatal(message string, component string, data map[string]interface{})
}

// DefaultLogger is the default concrete implementation of the Logger interface.
// It acts as a simple wrapper around the package-level logging functions (Debug, Info, etc.).
type DefaultLogger struct{}

// NewLogger creates and returns a new instance of DefaultLogger, which implements the Logger interface.
func NewLogger() Logger {
	return &DefaultLogger{}
}

// Debug logs a debug event using the package-level Debug function.
func (l *DefaultLogger) Debug(message string, component string, data map[string]interface{}) {
	Debug(message, component, data)
}

// Info logs an info event using the package-level Info function.
func (l *DefaultLogger) Info(message string, component string, data map[string]interface{}) {
	Info(message, component, data)
}

// Warn logs a warning event using the package-level Warn function.
func (l *DefaultLogger) Warn(message string, component string, data map[string]interface{}) {
	Warn(message, component, data)
}

// Error logs an error event using the package-level Error function.
func (l *DefaultLogger) Error(message string, component string, data map[string]interface{}) {
	Error(message, component, data)
}

// Fatal logs a fatal event using the package-level Fatal function.
func (l *DefaultLogger) Fatal(message string, component string, data map[string]interface{}) {
	Fatal(message, component, data)
}
