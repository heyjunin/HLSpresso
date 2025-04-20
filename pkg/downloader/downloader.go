package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/eusoujuninho/HLSpresso/pkg/errors"
	"github.com/eusoujuninho/HLSpresso/pkg/logger"
	"github.com/eusoujuninho/HLSpresso/pkg/progress"
)

// Options represents download options
type Options struct {
	URL           string
	OutputPath    string
	Timeout       time.Duration
	Progress      progress.Reporter
	AllowOverride bool
}

// Downloader downloads video files from URLs
type Downloader struct {
	client  *http.Client
	options Options
}

// New creates a new Downloader
func New(options Options) *Downloader {
	// Set default timeout if not specified
	if options.Timeout == 0 {
		options.Timeout = 30 * time.Minute
	}

	client := &http.Client{
		Timeout: options.Timeout,
	}

	return &Downloader{
		client:  client,
		options: options,
	}
}

// Download downloads a file from URL to the specified output path
func (d *Downloader) Download(ctx context.Context) (string, error) {
	// Create output directory if it doesn't exist
	outputDir := filepath.Dir(d.options.OutputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", errors.Wrap(err, errors.SystemError, "Failed to create output directory", 1)
	}

	// Check if file already exists
	if _, err := os.Stat(d.options.OutputPath); err == nil && !d.options.AllowOverride {
		logger.Info("File already exists, skipping download", "downloader", map[string]interface{}{
			"path": d.options.OutputPath,
		})
		return d.options.OutputPath, nil
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.options.URL, nil)
	if err != nil {
		return "", errors.Wrap(err, errors.DownloadError, "Failed to create HTTP request", 2)
	}

	// Log download start
	logger.Info("Starting download", "downloader", map[string]interface{}{
		"url":  d.options.URL,
		"path": d.options.OutputPath,
	})

	// Send request
	resp, err := d.client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, errors.DownloadError, "Failed to download file", 3)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return "", errors.New(errors.DownloadError, "HTTP request failed", fmt.Sprintf("Status: %s", resp.Status), 4)
	}

	// Create output file
	file, err := os.Create(d.options.OutputPath)
	if err != nil {
		return "", errors.Wrap(err, errors.SystemError, "Failed to create output file", 5)
	}
	defer file.Close()

	// Get content length for progress reporting
	contentLength := resp.ContentLength
	if contentLength > 0 && d.options.Progress != nil {
		d.options.Progress.Start(contentLength)
	}

	// Create a proxy reader to track download progress
	var reader io.Reader
	if d.options.Progress != nil && contentLength > 0 {
		reader = &progressReader{
			reader:   resp.Body,
			reporter: d.options.Progress,
			size:     contentLength,
		}
	} else {
		reader = resp.Body
	}

	// Copy data from response to file
	if _, err := io.Copy(file, reader); err != nil {
		return "", errors.Wrap(err, errors.DownloadError, "Failed to write file", 6)
	}

	// Complete progress
	if d.options.Progress != nil {
		d.options.Progress.Complete()
	}

	logger.Info("Download completed", "downloader", map[string]interface{}{
		"path": d.options.OutputPath,
	})

	return d.options.OutputPath, nil
}

// progressReader is a reader wrapper that reports download progress
type progressReader struct {
	reader   io.Reader
	reporter progress.Reporter
	size     int64
	read     int64
}

// Read reads data and updates progress
func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	if n > 0 {
		pr.read += int64(n)
		pr.reporter.Update(pr.read, "downloading", "Downloading file")
	}
	return n, err
}
