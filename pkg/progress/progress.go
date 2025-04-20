package progress

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/eusoujuninho/HLSpresso/pkg/logger"
	"github.com/schollz/progressbar/v3"
)

// ProgressEvent represents a progress update event
type ProgressEvent struct {
	Status     string  `json:"status"`
	Percentage float64 `json:"percentage"`
	Step       string  `json:"step"`
	Stage      string  `json:"stage"`
	Timestamp  string  `json:"timestamp"`
}

// Reporter interface defines progress reporting functions
type Reporter interface {
	Start(totalSteps int64)
	Update(current int64, step, stage string)
	Increment(step, stage string)
	Complete()
	JSON() (string, error)
}

// DefaultReporter implements the progress Reporter interface
type DefaultReporter struct {
	Total      int64
	Current    int64
	Started    time.Time
	Bar        *progressbar.ProgressBar
	LastUpdate time.Time
	Event      ProgressEvent
}

// NewReporter creates a new DefaultReporter
func NewReporter() *DefaultReporter {
	return &DefaultReporter{
		Event: ProgressEvent{
			Status:    "initialized",
			Timestamp: time.Now().Format(time.RFC3339),
		},
		LastUpdate: time.Now(),
	}
}

// Start initializes the progress tracking
func (r *DefaultReporter) Start(totalSteps int64) {
	r.Total = totalSteps
	r.Current = 0
	r.Started = time.Now()
	r.Event.Status = "started"
	r.Event.Percentage = 0
	r.Event.Timestamp = time.Now().Format(time.RFC3339)

	r.Bar = progressbar.NewOptions64(
		totalSteps,
		progressbar.OptionSetDescription("Transcoding..."),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowBytes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)

	// Log initial progress
	r.reportProgress()
}

// Update sets the current progress and reports it
func (r *DefaultReporter) Update(current int64, step, stage string) {
	r.Current = current
	r.Event.Percentage = float64(current) / float64(r.Total) * 100
	r.Event.Step = step
	r.Event.Stage = stage
	r.Event.Status = "processing"
	r.Event.Timestamp = time.Now().Format(time.RFC3339)

	_ = r.Bar.Set64(current)

	// Only report progress at most once per second to avoid flooding logs
	if time.Since(r.LastUpdate) >= time.Second {
		r.reportProgress()
		r.LastUpdate = time.Now()
	}
}

// Increment increases the progress by 1 and reports it
func (r *DefaultReporter) Increment(step, stage string) {
	r.Current++
	r.Event.Percentage = float64(r.Current) / float64(r.Total) * 100
	r.Event.Step = step
	r.Event.Stage = stage
	r.Event.Status = "processing"
	r.Event.Timestamp = time.Now().Format(time.RFC3339)

	_ = r.Bar.Add(1)

	// Only report progress at most once per second to avoid flooding logs
	if time.Since(r.LastUpdate) >= time.Second {
		r.reportProgress()
		r.LastUpdate = time.Now()
	}
}

// Complete marks the progress as complete
func (r *DefaultReporter) Complete() {
	_ = r.Bar.Finish()
	r.Current = r.Total
	r.Event.Percentage = 100
	r.Event.Status = "completed"
	r.Event.Timestamp = time.Now().Format(time.RFC3339)
	r.reportProgress()
}

// JSON returns the current progress event as JSON
func (r *DefaultReporter) JSON() (string, error) {
	data, err := json.Marshal(r.Event)
	if err != nil {
		return "", fmt.Errorf("failed to marshal progress event: %w", err)
	}
	return string(data), nil
}

// ReportProgress reports the current progress as a JSON message
func ReportProgress(reporter Reporter) (string, error) {
	return reporter.JSON()
}

// Internal function to report progress using the logger
func (r *DefaultReporter) reportProgress() {
	jsonData, err := r.JSON()
	if err != nil {
		logger.Error("Failed to generate progress JSON", "progress", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	data := map[string]interface{}{
		"progress": r.Event.Percentage,
		"step":     r.Event.Step,
		"stage":    r.Event.Stage,
		"status":   r.Event.Status,
		"json":     jsonData,
	}

	logger.Info("Transcoding progress", "progress", data)
}
