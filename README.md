# ☕ HLSpresso - Video Transcoder

A powerful Go-based video transcoding tool that converts videos to HLS adaptive streaming format and MP4. Supports downloading videos from remote URLs and generates JSON logs for monitoring progress and errors.

## ✨ Features

- **Input**: Process local video files or download from remote URLs.
- **Output**: Generate HLS adaptive streaming or simple MP4 output.
- **HLS Adaptive Streaming**: Create adaptive bitrate streaming with multiple quality levels.
- **Vertical Video Support**: Maintain aspect ratio for vertical (portrait) videos.
- **Remote URL Processing**: Automatically download and process videos from remote sources.
- **JSON Logs**: Structured logging in JSON format for easy integration with monitoring tools.
- **Progress Tracking**: Real-time progress updates during transcoding.
- **Error Handling**: Structured error reporting in JSON format.
- **Modular Architecture**: Well-organized codebase with independent, testable components.

## 📋 Requirements

- Go 1.21 or higher
- FFmpeg installed on the system
- FFprobe installed on the system (usually comes with FFmpeg)

## 🚀 Installation

### Using Go

```bash
# Clone the repository
git clone https://github.com/heyjunin/HLSpresso.git
cd HLSpresso

# Install dependencies
go mod tidy

# Build the binary
go build -o HLSpresso cmd/transcoder/main.go
```

### Using Makefile

```bash
# Clone the repository
git clone https://github.com/heyjunin/HLSpresso.git
cd HLSpresso

# Build the binary
make build

# Install the binary (may require sudo)
sudo make install
```

## 🚀 Quick Start

### Basic HLS Transcoding (Local File)

```bash
./HLSpresso -i input_video.mp4 -o output_directory
```

### Basic MP4 Transcoding

```bash
./HLSpresso -i input_video.mp4 -o output_video.mp4 -t mp4
```

### Remote URL to HLS

```bash
./HLSpresso -i https://example.com/video.mp4 -o output_directory --remote
```

## 📚 Use Cases and Examples

### 1. Standard HLS Adaptive Streaming

Create HLS with default quality levels (1080p, 720p, 480p):

```bash
./HLSpresso -i input_video.mp4 -o output_directory
```

### 2. HLS with Default Resolutions

This example creates HLS streams using the default built-in quality levels.

```bash
./HLSpresso -i input_video.mp4 -o output_directory
```

*Note: Currently, specifying custom HLS resolutions via the command line is not supported. The tool uses default resolutions. For custom resolutions, please use HLSpresso as a library.*

### 3. Custom HLS Segment Duration

Adjust the HLS segment duration (in seconds):

```bash
./HLSpresso -i input_video.mp4 -o output_directory --hls-segment-duration 6
```

### 4. Set HLS Playlist Type

Set the HLS playlist type to VOD (Video on Demand) or EVENT:

```bash
./HLSpresso -i input_video.mp4 -o output_directory --hls-playlist-type vod
```

### 5. Remote Video Processing

Download and transcode from a URL:

```bash
./HLSpresso -i https://example.com/video.mp4 -o output_directory --remote
```

### 6. MP4 Transcoding with Custom Settings

Create a simple MP4 file with custom FFmpeg parameters:

```bash
./HLSpresso -i input_video.mp4 -o output_video.mp4 -t mp4 \
  --ffmpeg-param "-crf 18" --ffmpeg-param "-preset slower"
```

### 7. Vertical Video Support

Process vertical videos (portrait mode) while maintaining aspect ratio using default HLS resolutions:

```bash
./HLSpresso -i vertical_video.mp4 -o output_directory
```

### 8. Social Media Optimized Vertical Video

Create vertical video with optimized settings for social media (using default HLS resolutions):

```bash
./HLSpresso -i vertical_video.mp4 -o output_directory \
  --hls-segment-duration 2
```

### 9. HLS with Custom FFmpeg Path

Specify a custom FFmpeg binary location:

```bash
./HLSpresso -i input_video.mp4 -o output_directory --ffmpeg /path/to/ffmpeg
```

### 10. Allow Overwriting Existing Files

Force overwrite of existing files without prompting:

```bash
./HLSpresso -i input_video.mp4 -o output_directory --overwrite
```

### 11. Specify Download Directory

Set a custom directory for downloaded remote videos:

```bash
./HLSpresso -i https://example.com/video.mp4 -o output_directory \
  --remote --download-dir /path/to/downloads
```

### 12. Combine Multiple Options

Combine various options for advanced use cases (using default HLS resolutions):

```bash
./HLSpresso -i https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ForBiggerBlazes.mp4 \
  -o output_directory \
  --remote \
  --hls-segment-duration 4 \
  --hls-playlist-type vod
```

## 🧰 Command Line Reference

```
HLSpresso - Tool for generating HLS adaptive streams

Usage:
  HLSpresso [flags]

Flags:
  -h, --help                       Display help information
  -i, --input string               Input file path or URL (required)
      --remote                     Treat input as a remote URL (downloads first)
      --stream                     Attempt to stream directly from input URL (implies remote)
      --download-dir string        Directory to save downloaded files (if not streaming) (default "downloads")
      --overwrite                  Allow overwriting existing files
  -o, --output string              Output directory or file path (required)
  -t, --type string                Output type: 'hls' or 'mp4' (default "hls")
      --hls-segment-duration int   HLS segment duration in seconds (default 10)
      --hls-playlist-type string   HLS playlist type: 'vod' or 'event' (default "vod")
      --ffmpeg string              Path to ffmpeg binary (default "ffmpeg")
      --ffmpeg-param stringArray   Extra parameters to pass to ffmpeg
      --progress-file string       Path to file for writing progress percentage (e.g., progress.txt)
      --progress-file-format string Format for progress file: 'text' (percentage only) or 'json' (full event) (default "text")
```

## 📜 Shell Script Helper

For simpler usage, you can also use the provided shell script:

```bash
# Basic usage
./scripts/HLSpresso.sh -i input_video.mp4 -o output_directory

# Custom segment duration and playlist type
./scripts/HLSpresso.sh -i input_video.mp4 -o output_directory -d 6 -t vod

# Custom resolutions (format: widthxheight:v:bitrate:a:bitrate)
./scripts/HLSpresso.sh -i input_video.mp4 -o output_directory \
  -r "1920x1080:v:5000k:a:192k,1280x720:v:2800k:a:128k,854x480:v:1400k:a:96k"
```

## 🏗️ Code Architecture

The project is organized into several packages:

- **cmd/transcoder**: Command line interface
- **pkg/transcoder**: Core transcoding logic
- **pkg/downloader**: URL download functionality
- **pkg/hls**: HLS generation with adaptive bitrates
- **pkg/logger**: JSON structured logging
- **pkg/progress**: Progress reporting
- **pkg/errors**: Structured error handling

## 🧪 Testing

### Unit Tests

Run the unit tests for all packages:

```bash
go test -v ./pkg/...

# Or using make
make test-unit
```

### End-to-end Tests

The end-to-end tests verify the complete functionality from input to output:

```bash
# Using make
make test-e2e

# Or directly
./scripts/run_e2e_tests.sh
```

### Examples Tests

Run the library usage examples located in the `examples/` directory to ensure they execute correctly:

```bash
# Using make
make test-examples

# Or directly
./scripts/run_examples_tests.sh
```

**Note:** Running example tests requires `ffmpeg` and `ffprobe` to be installed and may take some time due to transcoding operations (even with dummy input files).

### All Tests

Run all tests (unit and e2e):

```bash
make test
```

## 🌐 Cross-platform Building

Build binaries for multiple platforms:

```bash
make build-all
```

This creates binaries for:
- Linux (amd64)
- macOS (amd64, arm64)
- Windows (amd64)

The binaries will be available in the `build` directory.

## 📊 Example JSON Output

### Progress Output
```json
{
  "status": "processing",
  "percentage": 45.2,
  "step": "transcoding",
  "stage": "Creating HLS stream",
  "timestamp": "2023-08-15T14:23:45Z"
}
```

### Error Output
```json
{
  "type": "transcoding_error",
  "message": "FFmpeg command failed",
  "details": "Exit status 1",
  "timestamp": "2023-08-15T14:25:12Z",
  "code": 13
}
```

## 🛠️ Error Handling System

HLSpresso includes a robust error handling system designed to provide clear, actionable information when issues occur. All errors are structured with detailed information to help you quickly diagnose and resolve problems.

### Error Types

The library implements several specialized error types to address different failure scenarios:

| Error Type | Description | Code Range |
|------------|-------------|------------|
| NetworkError | Connection and network-related issues | 1000-1099 |
| DiskSpaceError | Storage and disk space issues | 1100-1199 |
| FileNotFoundError | Missing input files or directories | 1200-1299 |
| InvalidFileFormatError | Unsupported or corrupted file formats | 1300-1399 |
| PermissionError | Access permission issues | 1400-1499 |
| MemoryError | Memory allocation problems | 1500-1599 |
| CodecNotFoundError | Missing or incompatible codecs | 1600-1699 |
| InvalidOutputPathError | Output path issues | 1700-1799 |
| UnsupportedResolutionError | Video resolution problems | 1800-1899 |

### Error Structure

Each error contains the following information:

- **Type**: The category of error (e.g., `network_error`, `disk_space_error`)
- **Message**: User-friendly description of what went wrong
- **Details**: Technical details or underlying error message
- **Timestamp**: When the error occurred (RFC3339 format)
- **Code**: Specific error code for precise identification

### Handling Errors in Your Code

When using HLSpresso as a library, you can catch and process structured errors:

```go
outputFilePath, err := transcoder.Transcode(ctx)
if err != nil {
    // Check if it's a structured error
    if sErr, ok := err.(*errors.StructuredError); ok {
        // Access structured error fields
        fmt.Printf("Error %d: %s\n", sErr.Code, sErr.Message)
        
        // Handle specific error types
        switch sErr.Type {
        case errors.NetworkError:
            fmt.Println("Network issue detected. Check your connection.")
        case errors.DiskSpaceError:
            fmt.Println("Not enough disk space. Free up space and try again.")
        case errors.FileNotFoundError:
            fmt.Println("Input file couldn't be found. Check the path.")
        // Handle other error types...
        }
        
        // Log the full error details
        jsonErr, _ := sErr.JSON()
        logger.Error(jsonErr)
    } else {
        // Handle non-structured errors
        fmt.Printf("Generic error: %v\n", err)
    }
    return
}
```

### Common Error Codes and Solutions

#### Network Errors (1000-1099)
- **1000 (ErrNetworkConnectionFailed)**: Connection to remote server failed
  - *Solution*: Check your internet connection and the server URL
- **1001 (ErrNetworkTimeout)**: Network operation timed out
  - *Solution*: Check your connection speed or try again later
- **1002 (ErrNetworkDNSFailure)**: DNS resolution failed
  - *Solution*: Verify the server address or check your DNS settings
- **1003 (ErrNetworkServerUnavailable)**: Remote server unavailable
  - *Solution*: Verify the server is running or try again later

#### Disk Space Errors (1100-1199)
- **1100 (ErrDiskSpaceInsufficient)**: Not enough disk space
  - *Solution*: Free up disk space or use a different output location
- **1101 (ErrDiskQuotaExceeded)**: User or system disk quota exceeded
  - *Solution*: Clear disk space or request a quota increase

#### File Errors (1200-1399)
- **1200 (ErrFileNotFound)**: Input file not found
  - *Solution*: Verify the file path and existence
- **1300 (ErrInvalidFileFormat)**: File format not supported
  - *Solution*: Use a supported format (MP4, MOV, AVI, MKV, WEBM)
- **1302 (ErrCorruptedFile)**: Input file is corrupted
  - *Solution*: Check file integrity or obtain a clean copy

#### Permission Errors (1400-1499)
- **1400 (ErrPermissionDenied)**: Access permission denied
  - *Solution*: Check file/directory permissions

#### Codec Errors (1600-1699)
- **1600 (ErrCodecNotFound)**: Required codec not found
  - *Solution*: Install missing codec or FFmpeg component
- **1602 (ErrMissingDependency)**: Missing dependency (usually FFmpeg)
  - *Solution*: Install FFmpeg and required dependencies

### Error Prevention Best Practices

1. **Verify input files** before starting transcoding operations
2. **Check available disk space** for output, especially for HLS which creates multiple files
3. **Validate file formats** to ensure they're supported
4. **Handle network operations** with appropriate timeouts and retries
5. **Check permissions** for both input files and output directories
6. **Monitor memory usage** when processing large files

## 🔌 Advanced Integration

### Using as a Library

HLSpresso is designed to be easily integrated into your own Go applications. The core functionality resides in the `pkg/` directory, primarily within the `transcoder` package.

**1. Installation:**

Add HLSpresso as a dependency to your project:

```bash
go get github.com/heyjunin/HLSpresso@latest # Or specify a version tag like @v1.0.0
```

**2. Basic Usage:**

Here's a basic example demonstrating how to transcode a local file to HLS:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/heyjunin/HLSpresso/pkg/hls"
	"github.com/heyjunin/HLSpresso/pkg/progress"
	"github.com/heyjunin/HLSpresso/pkg/transcoder"
)

func main() {
	// Create a progress reporter to receive updates
	// (You can implement your own progress.Reporter interface for custom handling)
	progressReporter := progress.NewReporter(
		progress.WithThrottle(1 * time.Second), // Throttle updates
		// Example: Writing progress to a JSON file
		progress.WithProgressFile("transcode_progress.json"),
		progress.WithProgressFileFormat("json"),
	)

	// Configure the transcoder options
	options := transcoder.Options{
		// --- Input ---
		InputPath:     "input.mp4", // Path to the local video file
		IsRemoteInput: false,       // Set to true if InputPath is a URL

		// --- Output ---
		OutputPath: "output_hls_directory", // Directory for HLS output (or .mp4 file path for MP4)
		OutputType: transcoder.HLSOutput,   // Use transcoder.MP4Output for MP4

		// --- HLS Specific (if OutputType is HLSOutput) ---
		HLSSegmentDuration: 10,            // Segment duration in seconds
		HLSPlaylistType:    "vod",        // "vod" or "event"
		// Use default resolutions by leaving HLSResolutions nil or empty.
		// HLSResolutions: hls.DefaultResolutions,

		// --- Advanced ---
		// FFmpegBinary: "/path/to/custom/ffmpeg", // Optional: Specify FFmpeg path
		// FFmpegExtraParams: []string{"-preset", "slow"}, // Optional: Extra FFmpeg flags
		AllowOverwrite: true, // Optional: Allow overwriting output files
	}

	// Create a new transcoder instance
	trans, err := transcoder.New(options, progressReporter)
	if err != nil {
		log.Fatalf("Error creating transcoder: %v\n", err)
	}

	// Start a goroutine to listen for progress updates
	go func() {
		for p := range progressReporter.Updates() {
			fmt.Printf("Progress: %.1f%% - Step: %s (%s)\n", p.Percentage, p.Step, p.Stage)
		}
	}()

	// Start transcoding (use context for cancellation)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute) // Example timeout
	defer cancel()

	fmt.Println("Starting transcoding...")
	outputFilePath, err := trans.Transcode(ctx)
	if err != nil {
		// Handle potential structured errors
		if e, ok := err.(*errors.StructuredError); ok {
			log.Fatalf("Transcoding failed (Code: %d): %s - Details: %s\n", e.Code, e.Message, e.Details)
		} else {
			log.Fatalf("Transcoding failed: %v\n", err)
		}
		return
	}

	// Close the progress reporter when done
	// Note: DefaultReporter doesn't require explicit Close() anymore
	fmt.Printf("Transcoding completed successfully. Output at: %s\n", outputFilePath)
}

```

**3. Custom HLS Resolutions:**

To specify custom HLS resolutions when using HLSpresso as a library, populate the `HLSResolutions` field in the `transcoder.Options` struct. This field takes a slice of `hls.VideoResolution`.

```go
// Example: Define custom resolutions
customResolutions := []hls.VideoResolution{
    {Width: 1920, Height: 1080, VideoBitrate: "5000k", MaxRate: "5350k", BufSize: "7500k", AudioBitrate: "192k"},
    {Width: 1280, Height: 720, VideoBitrate: "2800k", MaxRate: "2996k", BufSize: "4200k", AudioBitrate: "128k"},
    {Width: 854, Height: 480, VideoBitrate: "1400k", MaxRate: "1498k", BufSize: "2100k", AudioBitrate: "96k"},
}

options := transcoder.Options{
    // ... other options
    OutputType:     transcoder.HLSOutput,
    HLSResolutions: customResolutions,
}

// ... create transcoder and run Transcode() ...
```

**4. Dependency Injection:**

For more advanced control, you can use `transcoder.NewWithDeps` to inject your own implementations of the logger (`logger.Logger`) or downloader (`downloader.Downloader`).

**5. Error Handling:**

The `Transcode` function can return structured errors defined in the `pkg/errors` package (`errors.StructuredError`). You can check the error type and access fields like `Code`, `Message`, and `Details` for more specific error handling.

## 📚 Using HLSpresso as a Library

You can integrate HLSpresso into your own Go applications.

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/heyjunin/HLSpresso/pkg/logger"
	"github.com/heyjunin/HLSpresso/pkg/progress"
	"github.com/heyjunin/HLSpresso/pkg/transcoder"
	"github.com/heyjunin/HLSpresso/pkg/downloader" // Import downloader for non-streaming remote
)

func main() {
	ctx := context.Background()

	// Create a logger (e.g., a simple one)
	appLogger := logger.NewLogger()

	// Create a progress reporter (e.g., a basic console reporter)
	reporter := progress.NewConsoleReporter()
	defer reporter.Close()

	// --- Example 1: Local file to HLS ---
	optsLocal := transcoder.Options{
		InputPath:  "path/to/your/local_video.mp4",
		OutputPath: "output/hls_local",
		OutputType: transcoder.HLSOutput,
		// Use default HLS settings or specify custom ones
	}
	transLocal, err := transcoder.New(optsLocal, reporter)
	if err != nil {
		log.Fatalf("Failed to create transcoder for local file: %v", err)
	}

	log.Println("Starting local HLS transcoding...")
	masterPlaylist, err := transLocal.Transcode(ctx)
	if err != nil {
		appLogger.Error("Local HLS transcoding failed", "main", map[string]interface{}{"error": err.Error()})
		// Handle structured error if needed:
		// if sErr, ok := err.(*errors.StructuredError); ok { ... }
		return
	}
	log.Printf("Local HLS transcoding finished. Master playlist: %s\n", masterPlaylist)


	// --- Example 2: Remote file to HLS (Download First) ---
	optsRemoteDownload := transcoder.Options{
		InputPath:   "https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/BigBuckBunny.mp4",
		OutputPath:  "output/hls_remote_download",
		OutputType:  transcoder.HLSOutput,
		DownloadDir: "temp_downloads", // Optional: specify download location
	}
	// For remote inputs *without* StreamFromURL, New() provides a default downloader.
	// Or, provide a custom one using NewWithDeps.
	transRemoteDownload, err := transcoder.New(optsRemoteDownload, reporter)
	if err != nil {
		log.Fatalf("Failed to create transcoder for remote download: %v", err)
	}

	log.Println("Starting remote HLS transcoding (download first)...")
	masterPlaylistRemote, err := transRemoteDownload.Transcode(ctx)
	if err != nil {
		appLogger.Error("Remote HLS (download) transcoding failed", "main", map[string]interface{}{"error": err.Error()})
		return
	}
	log.Printf("Remote HLS (download) transcoding finished. Master playlist: %s\n", masterPlaylistRemote)


	// --- Example 3: Remote file to HLS (Streaming Input) ---
	optsRemoteStream := transcoder.Options{
		InputPath:     "https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ForBiggerBlazes.mp4", // Use a different video for variety
		OutputPath:    "output/hls_remote_stream",
		OutputType:    transcoder.HLSOutput,
		StreamFromURL: true, // Enable streaming input
	}
	// When StreamFromURL is true, a downloader is NOT required or used.
	// We can use New() or NewWithDeps(..., nil).
	transRemoteStream, err := transcoder.New(optsRemoteStream, reporter)
	if err != nil {
		log.Fatalf("Failed to create transcoder for remote stream: %v", err)
	}

	log.Println("Starting remote HLS transcoding (streaming input)...")
	masterPlaylistStream, err := transRemoteStream.Transcode(ctx)
	if err != nil {
		appLogger.Error("Remote HLS (stream) transcoding failed", "main", map[string]interface{}{"error": err.Error()})
		return
	}
	log.Printf("Remote HLS (stream) transcoding finished. Master playlist: %s\n", masterPlaylistStream)
}

```

### Streaming Input (`StreamFromURL`)

By default, if the `InputPath` is a URL, HLSpresso will download the entire video file to a temporary directory (`DownloadDir`) before starting the transcoding process. This ensures the process is less susceptible to network interruptions during the potentially long transcoding phase.

However, you can enable direct streaming input by setting `StreamFromURL: true` in the `transcoder.Options`.

```go
opts := transcoder.Options{
	InputPath:     "https://your-video-source.com/video.mp4",
	OutputPath:    "output/hls_stream",
	StreamFromURL: true, // Process directly from the URL
}
```

**Considerations:**

*   **Network Dependency:** When `StreamFromURL` is true, the entire transcoding process relies on a stable network connection to the input URL. Network errors will cause the transcoding to fail.
*   **Server Support:** The server hosting the video must support HTTP range requests (seeking) for optimal performance and compatibility with FFmpeg.
*   **Downloader Skipped:** Features provided by the `pkg/downloader` (like custom retry logic, specific timeouts during download) are bypassed when streaming directly. FFmpeg handles the network connection.
*   **No Downloader Needed:** When `StreamFromURL` is true, you do not need to provide a `downloader` instance when using `transcoder.NewWithDeps`.

Choose the method (download first or stream directly) based on your reliability requirements and the nature of your video source.

## ❓ Troubleshooting

### Common Errors

1. **FFmpeg Not Found**: Ensure FFmpeg is installed and in your PATH or use the `--ffmpeg` flag.
2. **Permission Denied**: Ensure you have write permissions to the output directory.
3. **Input File Not Found**: Verify the input file path is correct.
4. **Remote URL Errors**: Check internet connectivity and URL validity.

### FFmpeg Version

This tool has been tested with FFmpeg 4.x and above. If you encounter issues, check your FFmpeg version:

```bash
ffmpeg -version
```

## 📄 License

MIT

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request to the [GitHub repository](https://github.com/heyjunin/HLSpresso). 