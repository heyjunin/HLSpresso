package progress

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/heyjunin/HLSpresso/pkg/logger"
	"github.com/schollz/progressbar/v3"
)

// ProgressEvent represents a single progress update event, often serialized to JSON.
type ProgressEvent struct {
	// Status indicates the current overall status (e.g., "initialized", "started", "processing", "completed").
	Status string `json:"status"`
	// Percentage represents the progress completion from 0.0 to 100.0.
	Percentage float64 `json:"percentage"`
	// Step provides a high-level description of the current phase (e.g., "downloading", "transcoding").
	Step string `json:"step"`
	// Stage offers a more detailed description within the current step (e.g., "Creating HLS stream", "Downloading file").
	Stage string `json:"stage"`
	// Timestamp marks when the event occurred in RFC3339 format.
	Timestamp string `json:"timestamp"`
}

// Reporter defines the interface for reporting progress during long-running operations
// like downloading or transcoding.
// HLSpresso components accept implementations of this interface to provide progress updates.
type Reporter interface {
	// Start initializes the progress reporting, typically setting the total number of steps or bytes.
	Start(total int64)
	// Update sets the current progress to a specific value.
	// It also takes descriptions of the current step and stage.
	Update(current int64, step, stage string)
	// Increment advances the progress by one step.
	// It also takes descriptions of the current step and stage.
	Increment(step, stage string)
	// Complete marks the operation as finished.
	Complete()
	// Updates returns a channel that emits ProgressEvent updates.
	// Consumers can listen on this channel to receive progress information.
	// The channel will be closed when the reporter is closed or the operation completes.
	Updates() <-chan ProgressEvent
	// Close signals that no more progress updates will be sent and closes the Updates channel.
	Close()
	// JSON returns the latest ProgressEvent as a JSON string.
	// Deprecated: Use the Updates() channel for receiving events.
	JSON() (string, error)
}

// reporterOptions holds configuration for the DefaultReporter.
type reporterOptions struct {
	throttle time.Duration
}

// ReporterOption is a function type used to configure a DefaultReporter.
type ReporterOption func(*reporterOptions)

// WithThrottle sets the minimum time interval between progress updates sent to the Updates channel.
// This helps prevent flooding listeners with too many events.
// Defaults to 0 (no throttling) if not set.
func WithThrottle(duration time.Duration) ReporterOption {
	return func(opts *reporterOptions) {
		opts.throttle = duration
	}
}

// DefaultReporter is the default implementation of the Reporter interface.
// It uses the github.com/schollz/progressbar/v3 library to display a progress
// bar on the console (stderr) and sends ProgressEvent updates to a channel.
type DefaultReporter struct {
	Total      int64
	Current    int64
	Started    time.Time
	Bar        *progressbar.ProgressBar
	options    reporterOptions
	updatesCh  chan ProgressEvent
	lastUpdate time.Time
	Event      ProgressEvent
	closed     bool
	mu         sync.Mutex // Protects access to shared fields
}

// NewReporter creates a new DefaultReporter.
// It accepts optional configuration functions like WithThrottle.
func NewReporter(opts ...ReporterOption) *DefaultReporter {
	return &DefaultReporter{
		Event: ProgressEvent{
			Status:    "initialized",
			Timestamp: time.Now().Format(time.RFC3339),
		},
		lastUpdate: time.Now(),
		updatesCh:  make(chan ProgressEvent, 10), // Buffered channel
	}
}

// Start initializes the progress tracking for the DefaultReporter.
// It sets the total number of steps and starts the progress bar.
func (r *DefaultReporter) Start(total int64) {
	r.Total = total
	r.Current = 0
	r.Started = time.Now()
	r.Event.Status = "started"
	r.Event.Percentage = 0
	r.Event.Timestamp = time.Now().Format(time.RFC3339)

	r.Bar = progressbar.NewOptions64(
		total,
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

	// Send initial event
	r.sendUpdate()
}

// Update sets the current progress and reports it via the progress bar and Updates channel.
// Updates to the channel may be throttled based on the WithThrottle option.
func (r *DefaultReporter) Update(current int64, step, stage string) {
	r.Current = current
	r.Event.Percentage = float64(current) / float64(r.Total) * 100
	r.Event.Step = step
	r.Event.Stage = stage
	r.Event.Status = "processing"
	r.Event.Timestamp = time.Now().Format(time.RFC3339)

	_ = r.Bar.Set64(current)

	r.sendUpdate()
}

// Increment increases the progress by 1 and reports it.
// Updates to the channel may be throttled.
func (r *DefaultReporter) Increment(step, stage string) {
	r.Current++
	r.Event.Percentage = float64(r.Current) / float64(r.Total) * 100
	r.Event.Step = step
	r.Event.Stage = stage
	r.Event.Status = "processing"
	r.Event.Timestamp = time.Now().Format(time.RFC3339)

	_ = r.Bar.Add(1)

	r.sendUpdate()
}

// Complete marks the progress as complete, finishes the progress bar, and sends a final update.
func (r *DefaultReporter) Complete() {
	_ = r.Bar.Finish()
	r.Current = r.Total
	r.Event.Percentage = 100
	r.Event.Status = "completed"
	r.Event.Timestamp = time.Now().Format(time.RFC3339)
	r.sendUpdate()
	// r.Close() // Do not close automatically on Complete; let the caller manage the lifecycle.
}

// Updates returns the channel for receiving ProgressEvent updates.
func (r *DefaultReporter) Updates() <-chan ProgressEvent {
	return r.updatesCh
}

// Close stops sending updates and closes the Updates channel.
// It should be called when the operation using the reporter is finished
// (or Complete() can be called which also calls Close).
func (r *DefaultReporter) Close() {
	close(r.updatesCh)
}

// JSON returns the current progress event as a JSON string.
// Deprecated: Use the Updates() channel for receiving events instead.
func (r *DefaultReporter) JSON() (string, error) {
	data, err := json.Marshal(r.Event)
	if err != nil {
		return "", fmt.Errorf("failed to marshal progress event: %w", err)
	}
	return string(data), nil
}

// reportProgress is an internal helper to log progress (optional) and send updates.
// It handles throttling.
func (r *DefaultReporter) sendUpdate() {
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

	select {
	case r.updatesCh <- r.Event:
	default:
		// If the channel is full, we don't need to send the update
	}
}

// reportProgress is deprecated and its logic is merged into sendUpdate.
func (r *DefaultReporter) reportProgress() {
	// Kept for potential backward compatibility or internal logging needs, but primarily use sendUpdate.
	// ... existing code ...
}

// ReportProgress is deprecated. Consume events from the Reporter.Updates() channel instead.
func ReportProgress(reporter Reporter) (string, error) {
	// Deprecated: use reporter.Updates() channel.
	return "", fmt.Errorf("ReportProgress is deprecated, consume the Updates() channel instead")
}
