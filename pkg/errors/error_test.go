package errors

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestTranscoderErrorImplementsErrorInterface(t *testing.T) {
	err := New(DownloadError, "Test error", "Test details", 123)

	// Check if it implements error interface
	var _ error = err

	// Check error message format
	expected := "[download_error] Test error: Test details"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

func TestTranscoderErrorJSON(t *testing.T) {
	err := New(TranscodingError, "JSON test", "Some details", 42)

	// Get JSON representation
	jsonStr, jsonErr := err.JSON()
	if jsonErr != nil {
		t.Fatalf("Failed to marshal error to JSON: %v", jsonErr)
	}

	// Parse it back to check fields
	var parsed map[string]interface{}
	if unmarshalErr := json.Unmarshal([]byte(jsonStr), &parsed); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", unmarshalErr)
	}

	// Check fields
	if parsed["type"] != string(TranscodingError) {
		t.Errorf("type = %q, want %q", parsed["type"], TranscodingError)
	}

	if parsed["message"] != "JSON test" {
		t.Errorf("message = %q, want %q", parsed["message"], "JSON test")
	}

	if parsed["details"] != "Some details" {
		t.Errorf("details = %q, want %q", parsed["details"], "Some details")
	}

	if parsed["code"].(float64) != 42 {
		t.Errorf("code = %v, want %v", parsed["code"], 42)
	}
}

func TestCaptureError(t *testing.T) {
	jsonStr, err := CaptureError(ValidationError, "Capture test", "Details here", 99)
	if err != nil {
		t.Fatalf("CaptureError failed: %v", err)
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if unmarshalErr := json.Unmarshal([]byte(jsonStr), &parsed); unmarshalErr != nil {
		t.Fatalf("Result is not valid JSON: %v", unmarshalErr)
	}

	// Check fields
	if parsed["type"] != string(ValidationError) {
		t.Errorf("type = %q, want %q", parsed["type"], ValidationError)
	}
}

func TestWrap(t *testing.T) {
	originalErr := errors.New("original error")
	wrapped := Wrap(originalErr, SystemError, "Wrapped error", 55)

	// Check error details
	if wrapped.Details != originalErr.Error() {
		t.Errorf("Details = %q, want %q", wrapped.Details, originalErr.Error())
	}

	if wrapped.Type != SystemError {
		t.Errorf("Type = %q, want %q", wrapped.Type, SystemError)
	}

	// Test wrapping nil
	nilWrapped := Wrap(nil, DownloadError, "Nil wrap", 1)
	if nilWrapped.Details != "" {
		t.Errorf("Details = %q, want empty string", nilWrapped.Details)
	}
}
