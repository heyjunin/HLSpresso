package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/heyjunin/HLSpresso/pkg/errors"
	"github.com/heyjunin/HLSpresso/pkg/logger"
	"github.com/heyjunin/HLSpresso/pkg/progress"
)

// Options represents configuration options for the Downloader.
type Options struct {
	// URL is the web address of the file to be downloaded.
	URL string
	// OutputPath is the local file system path where the downloaded file will be saved.
	OutputPath string
	// Timeout sets the maximum time allowed for the HTTP download operation.
	// Defaults to 30 minutes if not specified.
	Timeout time.Duration
	// Progress is an optional progress.Reporter to receive updates on the download progress.
	Progress progress.Reporter
	// AllowOverride, if true, allows the downloader to overwrite an existing file
	// at the OutputPath. If false and the file exists, the download is skipped.
	AllowOverride bool
}

// Downloader handles the process of downloading files from a given URL.
// It supports progress reporting and timeouts.
// Create instances using New().
type Downloader struct {
	client  *http.Client
	options Options
}

// New creates a new Downloader instance configured with the provided options.
// It sets a default timeout of 30 minutes if Options.Timeout is zero.
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

// Download initiates the file download from the URL specified in the Downloader's options
// and saves it to the specified OutputPath.
// The context can be used to cancel the download operation.
// It handles directory creation, checks for existing files (based on AllowOverride),
// reports progress (if a reporter is provided), and handles potential errors.
// Returns the final output path upon successful download, or an error.
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

// progressReader is an internal io.Reader wrapper used to track download progress
// by reporting the number of bytes read via a progress.Reporter.
type progressReader struct {
	reader   io.Reader
	reporter progress.Reporter
	size     int64
	read     int64
}

// Read implements the io.Reader interface for progressReader.
// It reads from the underlying reader and updates the progress reporter.
func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	if n > 0 {
		pr.read += int64(n)
		pr.reporter.Update(pr.read, "downloading", "Downloading file")
	}
	return n, err
}
