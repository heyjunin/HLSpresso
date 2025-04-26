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
	// JSON returns the latest ProgressEvent as a JSON string.
	// Deprecated: Use the Updates() channel for receiving events.
	JSON() (string, error)
}

// reporterOptions holds configuration for the DefaultReporter.
type reporterOptions struct {
	throttle           time.Duration
	progressFilePath   string // New option: path to the progress file
	progressFileFormat string // New option: "text" or "json" (default: "text")
	description        string // Option for progress bar description
	showBytes          bool   // Option to show bytes in progress bar
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

// WithProgressFile sets the file path where the current progress should be written.
// The format is controlled by WithProgressFileFormat (defaults to "text").
// The file is created or truncated on Start and overwritten on each Update and Complete.
// If the path is empty (default), no file will be written.
func WithProgressFile(path string) ReporterOption {
	return func(opts *reporterOptions) {
		opts.progressFilePath = path
	}
}

// WithProgressFileFormat sets the format for the progress file ("text" or "json").
// Defaults to "text" (only percentage) if not specified.
// Requires WithProgressFile to be set with a non-empty path.
// If "json" is selected, the entire ProgressEvent struct is marshaled and written.
func WithProgressFileFormat(format string) ReporterOption {
	return func(opts *reporterOptions) {
		// Basic validation, could be stricter (enum?)
		if format == "json" || format == "text" {
			opts.progressFileFormat = format
		} else {
			// Log a warning or default to "text"? Defaulting for now.
			logger.Warn("Invalid progress file format specified, defaulting to 'text'", "progress", map[string]interface{}{
				"format": format,
			})
			opts.progressFileFormat = "text"
		}
	}
}

// WithDescription sets the description text for the console progress bar.
func WithDescription(desc string) ReporterOption {
	return func(opts *reporterOptions) {
		opts.description = desc
	}
}

// WithShowBytes configures the console progress bar to display progress in bytes.
func WithShowBytes(show bool) ReporterOption {
	return func(opts *reporterOptions) {
		opts.showBytes = show
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
	opts       reporterOptions
	updatesCh  chan ProgressEvent
	lastUpdate time.Time
	Event      ProgressEvent
	mu         sync.Mutex // Protects access to shared fields
}

// NewReporter creates a new DefaultReporter.
// It accepts optional configuration functions like WithThrottle, WithProgressFile, WithDescription, and WithShowBytes.
func NewReporter(opts ...ReporterOption) *DefaultReporter {
	// Default options
	options := reporterOptions{
		description:        "Processing...",
		showBytes:          true,   // Default to showing bytes
		progressFileFormat: "text", // Default format
	}
	// Apply provided functional options
	for _, opt := range opts {
		opt(&options)
	}

	r := &DefaultReporter{
		opts: options,
		Event: ProgressEvent{
			Status:    "initialized",
			Timestamp: time.Now().Format(time.RFC3339),
		},
		lastUpdate: time.Now(),
		updatesCh:  make(chan ProgressEvent, 10), // Buffered channel
	}
	return r
}

// Start initializes the progress tracking for the DefaultReporter.
// It sets the total number of steps and starts the progress bar.
func (r *DefaultReporter) Start(total int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Total = total
	r.Current = 0
	r.Started = time.Now()
	r.Event.Status = "started"
	r.Event.Percentage = 0
	r.Event.Timestamp = time.Now().Format(time.RFC3339)

	barOpts := []progressbar.Option{
		progressbar.OptionSetDescription(r.opts.description),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowCount(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	}
	if r.opts.showBytes {
		barOpts = append(barOpts, progressbar.OptionShowBytes(true))
	}

	r.Bar = progressbar.NewOptions64(total, barOpts...)

	// Send initial event and write initial file state
	r.sendUpdateInternal(true)
	r.writeProgressFileInternal()
}

// Update sets the current progress and reports it via the progress bar and Updates channel.
// Updates to the channel may be throttled based on the WithThrottle option.
func (r *DefaultReporter) Update(current int64, step, stage string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Bar == nil {
		return
	} // Not started
	if current > r.Total {
		current = r.Total
	} // Cap progress
	r.Current = current

	percentage := 0.0
	if r.Total > 0 {
		percentage = float64(current) / float64(r.Total) * 100
	}
	r.Event.Percentage = percentage
	r.Event.Step = step
	r.Event.Stage = stage
	r.Event.Status = "processing"
	r.Event.Timestamp = time.Now().Format(time.RFC3339)

	_ = r.Bar.Set64(current)

	r.sendUpdateInternal(false)   // Throttle updates channel
	r.writeProgressFileInternal() // Write file on every update
}

// Increment increases the progress by 1 and reports it.
// Updates to the channel may be throttled.
func (r *DefaultReporter) Increment(step, stage string) {
	r.Update(r.Current+1, step, stage) // Reuse Update logic
}

// Complete marks the progress as complete, finishes the progress bar, and sends a final update.
func (r *DefaultReporter) Complete() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Bar == nil {
		return
	} // Not started or already completed

	_ = r.Bar.Finish()
	r.Current = r.Total
	r.Event.Percentage = 100
	r.Event.Status = "completed"
	r.Event.Timestamp = time.Now().Format(time.RFC3339)

	r.sendUpdateInternal(true)    // Send final update regardless of throttle
	r.writeProgressFileInternal() // Write final state
	r.Bar = nil                   // Mark as finished to prevent further updates
	close(r.updatesCh)            // Close deprecated channel
}

// Updates returns the channel for receiving ProgressEvent updates.
func (r *DefaultReporter) Updates() <-chan ProgressEvent {
	return r.updatesCh
}

// JSON returns the current progress event as a JSON string.
// Deprecated: Use the Updates() channel for receiving events instead.
func (r *DefaultReporter) JSON() (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	data, err := json.Marshal(r.Event)
	if err != nil {
		return "", fmt.Errorf("failed to marshal progress event: %w", err)
	}
	return string(data), nil
}

// sendUpdateInternal handles sending updates to the deprecated channel with throttling.
// Requires lock to be held by caller.
func (r *DefaultReporter) sendUpdateInternal(force bool) {
	now := time.Now()
	if !force && now.Sub(r.lastUpdate) < r.opts.throttle {
		return // Throttled
	}
	r.lastUpdate = now

	// Non-blocking send to deprecated channel
	select {
	case r.updatesCh <- r.Event:
	default:
	}
}

// writeProgressFileInternal writes the current progress to the configured file,
// respecting the specified format ("text" or "json").
// Requires lock to be held by caller.
func (r *DefaultReporter) writeProgressFileInternal() {
	if r.opts.progressFilePath == "" {
		return // No file path configured
	}

	var content []byte
	var err error

	switch r.opts.progressFileFormat {
	case "json":
		content, err = json.MarshalIndent(r.Event, "", "  ") // Use MarshalIndent for readability
		if err != nil {
			logger.Warn("Failed to marshal progress event to JSON", "progress", map[string]interface{}{
				"path":  r.opts.progressFilePath,
				"error": err.Error(),
			})
			return // Don't write partial/corrupt data
		}
	case "text":
		fallthrough // Fallthrough to default
	default: // Default to text format (percentage)
		content = []byte(fmt.Sprintf("%.2f", r.Event.Percentage))
	}

	err = os.WriteFile(r.opts.progressFilePath, content, 0644)
	if err != nil {
		// Log the error but don't stop the whole process
		logger.Warn("Failed to write progress file", "progress", map[string]interface{}{
			"path":   r.opts.progressFilePath,
			"format": r.opts.progressFileFormat,
			"error":  err.Error(),
		})
	}
}

// ReportProgress is deprecated. Consume events from the Reporter.Updates() channel instead.
func ReportProgress(reporter Reporter) (string, error) {
	// Deprecated: use reporter.Updates() channel.
	return "", fmt.Errorf("ReportProgress is deprecated, consume the Updates() channel instead")
}
