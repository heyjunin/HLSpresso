package progress

import (
	"encoding/json"
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
