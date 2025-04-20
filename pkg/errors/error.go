package errors

import (
	"encoding/json"
	"fmt"
	"time"
)

// ErrorType represents different categories of errors
type ErrorType string

const (
	// DownloadError occurs during file download
	DownloadError ErrorType = "download_error"
	// TranscodingError occurs during video transcoding
	TranscodingError ErrorType = "transcoding_error"
	// HLSError occurs during HLS creation
	HLSError ErrorType = "hls_error"
	// ValidationError occurs when input parameters are invalid
	ValidationError ErrorType = "validation_error"
	// SystemError represents system-level errors
	SystemError ErrorType = "system_error"
)

// TranscoderError represents a structured error with JSON serialization
type TranscoderError struct {
	Type      ErrorType `json:"type"`
	Message   string    `json:"message"`
	Details   string    `json:"details,omitempty"`
	Timestamp string    `json:"timestamp"`
	Code      int       `json:"code"`
}

// Error implements the error interface
func (e *TranscoderError) Error() string {
	return fmt.Sprintf("[%s] %s: %s", e.Type, e.Message, e.Details)
}

// JSON returns the error as a JSON string
func (e *TranscoderError) JSON() (string, error) {
	data, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// New creates a new TranscoderError
func New(errorType ErrorType, message, details string, code int) *TranscoderError {
	return &TranscoderError{
		Type:      errorType,
		Message:   message,
		Details:   details,
		Timestamp: time.Now().Format(time.RFC3339),
		Code:      code,
	}
}

// CaptureError creates and returns a structured error as JSON string
func CaptureError(errorType ErrorType, message string, details string, code int) (string, error) {
	err := New(errorType, message, details, code)
	return err.JSON()
}

// Wrap wraps an existing error in a TranscoderError
func Wrap(err error, errorType ErrorType, message string, code int) *TranscoderError {
	details := ""
	if err != nil {
		details = err.Error()
	}
	return New(errorType, message, details, code)
}
