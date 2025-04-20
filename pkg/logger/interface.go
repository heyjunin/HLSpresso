package logger

// Logger defines a standard logging interface for the application
type Logger interface {
	Debug(message string, component string, data map[string]interface{})
	Info(message string, component string, data map[string]interface{})
	Warn(message string, component string, data map[string]interface{})
	Error(message string, component string, data map[string]interface{})
	Fatal(message string, component string, data map[string]interface{})
}

// DefaultLogger is the default implementation of the Logger interface
type DefaultLogger struct{}

// NewLogger creates a new instance of the default logger
func NewLogger() Logger {
	return &DefaultLogger{}
}

// Debug logs a debug event
func (l *DefaultLogger) Debug(message string, component string, data map[string]interface{}) {
	Debug(message, component, data)
}

// Info logs an info event
func (l *DefaultLogger) Info(message string, component string, data map[string]interface{}) {
	Info(message, component, data)
}

// Warn logs a warning event
func (l *DefaultLogger) Warn(message string, component string, data map[string]interface{}) {
	Warn(message, component, data)
}

// Error logs an error event
func (l *DefaultLogger) Error(message string, component string, data map[string]interface{}) {
	Error(message, component, data)
}

// Fatal logs a fatal event
func (l *DefaultLogger) Fatal(message string, component string, data map[string]interface{}) {
	Fatal(message, component, data)
}
