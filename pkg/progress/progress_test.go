package progress

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNewReporter(t *testing.T) {
	reporter := NewReporter()

	if reporter == nil {
		t.Fatal("NewReporter() returned nil")
	}

	if reporter.Event.Status != "initialized" {
		t.Errorf("Initial status = %q, want %q", reporter.Event.Status, "initialized")
	}

	if reporter.Event.Timestamp == "" {
		t.Error("Timestamp should not be empty")
	}
}

func TestReporterStart(t *testing.T) {
	reporter := NewReporter()
	reporter.Start(100)

	if reporter.Total != 100 {
		t.Errorf("Total = %d, want %d", reporter.Total, 100)
	}

	if reporter.Current != 0 {
		t.Errorf("Current = %d, want %d", reporter.Current, 0)
	}

	if reporter.Event.Status != "started" {
		t.Errorf("Status = %q, want %q", reporter.Event.Status, "started")
	}

	if reporter.Event.Percentage != 0 {
		t.Errorf("Percentage = %f, want %f", reporter.Event.Percentage, 0.0)
	}

	if reporter.Bar == nil {
		t.Error("Progress bar should be initialized")
	}
}

func TestReporterUpdate(t *testing.T) {
	reporter := NewReporter()
	reporter.Start(200)

	reporter.Update(50, "test-step", "test-stage")

	if reporter.Current != 50 {
		t.Errorf("Current = %d, want %d", reporter.Current, 50)
	}

	if reporter.Event.Percentage != 25.0 {
		t.Errorf("Percentage = %f, want %f", reporter.Event.Percentage, 25.0)
	}

	if reporter.Event.Step != "test-step" {
		t.Errorf("Step = %q, want %q", reporter.Event.Step, "test-step")
	}

	if reporter.Event.Stage != "test-stage" {
		t.Errorf("Stage = %q, want %q", reporter.Event.Stage, "test-stage")
	}

	if reporter.Event.Status != "processing" {
		t.Errorf("Status = %q, want %q", reporter.Event.Status, "processing")
	}
}

func TestReporterIncrement(t *testing.T) {
	reporter := NewReporter()
	reporter.Start(100)

	// Increment multiple times
	for i := 0; i < 5; i++ {
		reporter.Increment("increment-step", "increment-stage")
	}

	if reporter.Current != 5 {
		t.Errorf("Current = %d, want %d", reporter.Current, 5)
	}

	if reporter.Event.Percentage != 5.0 {
		t.Errorf("Percentage = %f, want %f", reporter.Event.Percentage, 5.0)
	}
}

func TestReporterComplete(t *testing.T) {
	reporter := NewReporter()
	reporter.Start(50)

	reporter.Complete()

	if reporter.Current != 50 {
		t.Errorf("Current = %d, want %d", reporter.Current, 50)
	}

	if reporter.Event.Percentage != 100.0 {
		t.Errorf("Percentage = %f, want %f", reporter.Event.Percentage, 100.0)
	}

	if reporter.Event.Status != "completed" {
		t.Errorf("Status = %q, want %q", reporter.Event.Status, "completed")
	}
}

func TestReporterJSON(t *testing.T) {
	reporter := NewReporter()
	reporter.Start(100)
	reporter.Update(25, "json-step", "json-stage")

	jsonStr, err := reporter.JSON()
	if err != nil {
		t.Fatalf("JSON() failed: %v", err)
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if unmarshalErr := json.Unmarshal([]byte(jsonStr), &parsed); unmarshalErr != nil {
		t.Fatalf("Result is not valid JSON: %v", unmarshalErr)
	}

	// Check fields
	if parsed["status"] != "processing" {
		t.Errorf("status = %q, want %q", parsed["status"], "processing")
	}

	if parsed["percentage"].(float64) != 25.0 {
		t.Errorf("percentage = %f, want %f", parsed["percentage"], 25.0)
	}

	if parsed["step"] != "json-step" {
		t.Errorf("step = %q, want %q", parsed["step"], "json-step")
	}

	if parsed["stage"] != "json-stage" {
		t.Errorf("stage = %q, want %q", parsed["stage"], "json-stage")
	}
}

func TestReporterWithProgressFile(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("FormatText", func(t *testing.T) {
		progressFilePath := filepath.Join(tempDir, "progress_text.txt")

		// 1. Test Initialization (File Creation/Truncation)
		initialContent := "initial data"
		if err := os.WriteFile(progressFilePath, []byte(initialContent), 0644); err != nil {
			t.Fatalf("Failed to create initial progress file: %v", err)
		}

		// Default format is text
		reporter := NewReporter(WithProgressFile(progressFilePath))
		reporter.Start(100) // Start should write initial state (0.00)

		contentBytes, err := os.ReadFile(progressFilePath)
		if err != nil {
			t.Fatalf("Failed to read progress file after Start: %v", err)
		}
		content := string(contentBytes)
		expected := "0.00"
		if content != expected {
			t.Errorf("Progress file content after Start = %q, want %q", content, expected)
		}

		// 2. Test Update
		reporter.Update(55, "step", "stage")

		contentBytes, err = os.ReadFile(progressFilePath)
		if err != nil {
			t.Fatalf("Failed to read progress file after Update: %v", err)
		}
		content = string(contentBytes)
		expected = "55.00"
		if content != expected {
			t.Errorf("Progress file content after Update = %q, want %q", content, expected)
		}

		// 3. Test Complete
		reporter.Complete()

		contentBytes, err = os.ReadFile(progressFilePath)
		if err != nil {
			t.Fatalf("Failed to read progress file after Complete: %v", err)
		}
		content = string(contentBytes)
		expected = "100.00"
		if content != expected {
			t.Errorf("Progress file content after Complete = %q, want %q", content, expected)
		}
	})

	t.Run("FormatJSON", func(t *testing.T) {
		progressFilePath := filepath.Join(tempDir, "progress_json.json")

		// 1. Test Initialization (File Creation/Truncation)
		initialContent := "{}"
		if err := os.WriteFile(progressFilePath, []byte(initialContent), 0644); err != nil {
			t.Fatalf("Failed to create initial progress file: %v", err)
		}

		reporter := NewReporter(WithProgressFile(progressFilePath), WithProgressFileFormat("json"))

		// Capture the event state *before* calling Start to compare later
		initialEventState := reporter.Event
		initialEventState.Status = "started" // Expected status after Start
		initialEventState.Percentage = 0.0   // Expected percentage after Start

		reporter.Start(200)

		contentBytes, err := os.ReadFile(progressFilePath)
		if err != nil {
			t.Fatalf("Failed to read progress file after Start: %v", err)
		}

		var eventFromFileStart ProgressEvent
		if err := json.Unmarshal(contentBytes, &eventFromFileStart); err != nil {
			t.Fatalf("Failed to unmarshal JSON after Start: %v\nContent:\n%s", err, string(contentBytes))
		}

		// Compare fields individually (ignoring Timestamp)
		if eventFromFileStart.Status != initialEventState.Status {
			t.Errorf("JSON Start Status = %q, want %q", eventFromFileStart.Status, initialEventState.Status)
		}
		if eventFromFileStart.Percentage != initialEventState.Percentage {
			t.Errorf("JSON Start Percentage = %.2f, want %.2f", eventFromFileStart.Percentage, initialEventState.Percentage)
		}
		// Compare other relevant fields if needed (Step, Stage are usually empty on Start)

		// 2. Test Update
		reporter.Update(110, "json-step", "json-stage")

		updatedEventState := reporter.Event // Capture state after Update

		contentBytes, err = os.ReadFile(progressFilePath)
		if err != nil {
			t.Fatalf("Failed to read progress file after Update: %v", err)
		}
		var eventFromFileUpdate ProgressEvent
		if err := json.Unmarshal(contentBytes, &eventFromFileUpdate); err != nil {
			t.Fatalf("Failed to unmarshal JSON after Update: %v\nContent:\n%s", err, string(contentBytes))
		}

		// Compare fields individually (ignoring Timestamp)
		if eventFromFileUpdate.Status != updatedEventState.Status {
			t.Errorf("JSON Update Status = %q, want %q", eventFromFileUpdate.Status, updatedEventState.Status)
		}
		if eventFromFileUpdate.Percentage != updatedEventState.Percentage {
			t.Errorf("JSON Update Percentage = %.2f, want %.2f", eventFromFileUpdate.Percentage, updatedEventState.Percentage)
		}
		if eventFromFileUpdate.Step != updatedEventState.Step {
			t.Errorf("JSON Update Step = %q, want %q", eventFromFileUpdate.Step, updatedEventState.Step)
		}
		if eventFromFileUpdate.Stage != updatedEventState.Stage {
			t.Errorf("JSON Update Stage = %q, want %q", eventFromFileUpdate.Stage, updatedEventState.Stage)
		}

		// 3. Test Complete
		reporter.Complete()

		completedEventState := reporter.Event // Capture state after Complete

		contentBytes, err = os.ReadFile(progressFilePath)
		if err != nil {
			t.Fatalf("Failed to read progress file after Complete: %v", err)
		}
		var eventFromFileComplete ProgressEvent
		if err := json.Unmarshal(contentBytes, &eventFromFileComplete); err != nil {
			t.Fatalf("Failed to unmarshal JSON after Complete: %v\nContent:\n%s", err, string(contentBytes))
		}

		// Compare fields individually (ignoring Timestamp)
		if eventFromFileComplete.Status != completedEventState.Status {
			t.Errorf("JSON Complete Status = %q, want %q", eventFromFileComplete.Status, completedEventState.Status)
		}
		if eventFromFileComplete.Percentage != completedEventState.Percentage {
			t.Errorf("JSON Complete Percentage = %.2f, want %.2f", eventFromFileComplete.Percentage, completedEventState.Percentage)
		}
		// Compare other relevant fields if needed
	})

	// Test for empty path remains the same
	t.Run("EmptyPath", func(t *testing.T) {
		emptyPathReporter := NewReporter(WithProgressFile(""))
		otherFilePath := filepath.Join(tempDir, "should_not_exist.txt")

		// Directly check the internal option
		if emptyPathReporter.opts.progressFilePath != "" {
			t.Fatalf("Expected progressFilePath option to be empty, but got %q", emptyPathReporter.opts.progressFilePath)
		}

		// Call methods to ensure no panic and no file is created
		emptyPathReporter.Start(10)
		emptyPathReporter.Update(5, "s", "s")
		emptyPathReporter.Complete()

		if _, err := os.Stat(otherFilePath); !os.IsNotExist(err) {
			t.Errorf("Progress file seems to have been created unexpectedly at: %s", otherFilePath)
		}
	})
}

/* // Teste removido pois a função ReportProgress foi depreciada.
func TestReportProgress(t *testing.T) {
	reporter := NewReporter()
	reporter.Start(100)
	reporter.Update(50, "report-step", "report-stage")

	jsonStr, err := ReportProgress(reporter)
	if err != nil {
		t.Fatalf("ReportProgress() failed: %v", err)
	}

	// Check if it contains expected data
	if !strings.Contains(jsonStr, "report-step") || !strings.Contains(jsonStr, "report-stage") {
		t.Errorf("ReportProgress output does not contain expected data: %s", jsonStr)
	}
}
*/
