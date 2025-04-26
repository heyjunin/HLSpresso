package errors

import (
	"encoding/json"
	"fmt"
	"time"
)

// ErrorType defines distinct categories for errors originating from HLSpresso components.
type ErrorType string

const (
	// DownloadError represents errors occurring during the file download process.
	DownloadError ErrorType = "download_error"
	// TranscodingError represents errors occurring during the core video transcoding phase (FFmpeg execution).
	TranscodingError ErrorType = "transcoding_error"
	// HLSError represents errors specific to HLS manifest or segment generation.
	HLSError ErrorType = "hls_error"
	// ValidationError represents errors caused by invalid input parameters or configuration.
	ValidationError ErrorType = "validation_error"
	// SystemError represents underlying system issues, such as file I/O errors or command execution problems (excluding FFmpeg transcoding itself).
	SystemError ErrorType = "system_error"
)

// StructuredError represents a detailed error originating from HLSpresso operations.
// It includes a type, message, optional details, timestamp, and a specific error code.
// It implements the standard Go `error` interface.
type StructuredError struct {
	// Type categorizes the error (e.g., DownloadError, TranscodingError).
	Type ErrorType `json:"type"`
	// Message provides a concise, human-readable description of the error.
	Message string `json:"message"`
	// Details offers additional context or the underlying error message, if available.
	Details string `json:"details,omitempty"`
	// Timestamp marks when the error occurred in RFC3339 format.
	Timestamp string `json:"timestamp"`
	// Code provides a specific integer code unique to the error source within its type.
	Code int `json:"code"`
}

// Error implements the standard `error` interface for StructuredError.
// It returns a formatted string including the error type, message, and details.
func (e *StructuredError) Error() string {
	return fmt.Sprintf("[%s] %s: %s", e.Type, e.Message, e.Details)
}

// JSON returns the StructuredError serialized as a JSON string.
// Returns an empty string and an error if marshalling fails.
func (e *StructuredError) JSON() (string, error) {
	data, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// New creates a new StructuredError instance.
// It automatically sets the Timestamp to the current time.
func New(errorType ErrorType, message, details string, code int) *StructuredError {
	return &StructuredError{
		Type:      errorType,
		Message:   message,
		Details:   details,
		Timestamp: time.Now().Format(time.RFC3339),
		Code:      code,
	}
}

// CaptureError creates a new StructuredError and returns it serialized as a JSON string.
// Deprecated: It's generally better to return the StructuredError directly and let the caller decide on serialization.
func CaptureError(errorType ErrorType, message string, details string, code int) (string, error) {
	err := New(errorType, message, details, code)
	return err.JSON()
}

// Wrap creates a new StructuredError, using the message from an existing standard Go error
// as the Details field.
// If the input error `err` is nil, Details will be empty.
func Wrap(err error, errorType ErrorType, message string, code int) *StructuredError {
	details := ""
	if err != nil {
		details = err.Error()
	}
	return New(errorType, message, details, code)
}
