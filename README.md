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

### 2. Custom HLS with Specific Resolutions

```bash
./HLSpresso -i input_video.mp4 -o output_directory \
  --hls-resolutions "1920x1080:5000k:5350k:7500k:192k,1280x720:2800k:2996k:4200k:128k,854x480:1400k:1498k:2100k:96k"
```

Each resolution is specified in the format:
`width x height : video bitrate : max bitrate : buffer size : audio bitrate`

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

Process vertical videos (portrait mode) while maintaining aspect ratio:

```bash
./HLSpresso -i vertical_video.mp4 -o output_directory \
  --hls-resolutions "720x1280:2800k:2996k:4200k:128k,540x960:1400k:1498k:2100k:96k,360x640:800k:856k:1200k:64k"
```

### 8. Social Media Optimized Vertical Video

Create vertical video with optimized settings for social media:

```bash
./HLSpresso -i vertical_video.mp4 -o output_directory \
  --hls-segment-duration 2 \
  --hls-resolutions "720x1280:3000k:3200k:4500k:128k,540x960:1800k:1900k:2700k:96k,360x640:900k:950k:1400k:64k"
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

Combine various options for advanced use cases:

```bash
./HLSpresso -i https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ForBiggerBlazes.mp4 \
  -o output_directory \
  --remote \
  --hls-segment-duration 4 \
  --hls-playlist-type vod \
  --hls-resolutions "1280x720:2800k:2996k:4200k:128k,854x480:1400k:1498k:2100k:96k,640x360:800k:856k:1200k:64k"
```

## 🧰 Command Line Reference

```
HLSpresso - Tool for generating HLS adaptive streams

Usage:
  HLSpresso [flags]

Flags:
  -h, --help                       Display help information
  -i, --input string               Input file path or URL (required)
      --remote                     Treat input as a remote URL
      --download-dir string        Directory to save downloaded files (default "downloads")
      --overwrite                  Allow overwriting existing files
  -o, --output string              Output directory or file path (required)
  -t, --type string                Output type: 'hls' or 'mp4' (default "hls")
      --hls-segment-duration int   HLS segment duration in seconds (default 10)
      --hls-playlist-type string   HLS playlist type: 'vod' or 'event' (default "vod")
      --hls-resolutions string     Custom resolutions for HLS (format: widthxheight:vbr:maxrate:bufsize:abr,...)
      --ffmpeg string              Path to ffmpeg binary (default "ffmpeg")
      --ffmpeg-param stringArray   Extra parameters to pass to ffmpeg
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

The e2e tests include:
- Testing local file to HLS transcoding
- Testing remote URL to MP4 transcoding
- Testing vertical video transcoding
- Testing command line interface

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

## 🔌 Advanced Integration

### Using as a Library

You can import and use the transcoder programmatically in your own Go code:

```go
package main

import (
    "context"
    "fmt"

    "github.com/heyjunin/HLSpresso/pkg/progress"
    "github.com/heyjunin/HLSpresso/pkg/transcoder"
)

func main() {
    // Create a progress reporter
    progressReporter := progress.NewReporter()

    // Create transcoder options
    options := transcoder.Options{
        InputPath:          "input.mp4",
        OutputPath:         "output_directory",
        OutputType:         transcoder.HLSOutput,
        HLSSegmentDuration: 6,
        HLSPlaylistType:    "vod",
    }

    // Create transcoder
    trans, err := transcoder.New(options, progressReporter)
    if err != nil {
        fmt.Printf("Error creating transcoder: %v\n", err)
        return
    }

    // Start transcoding
    ctx := context.Background()
    outputPath, err := trans.Transcode(ctx)
    if err != nil {
        fmt.Printf("Transcoding failed: %v\n", err)
        return
    }

    fmt.Printf("Transcoding completed successfully. Output at: %s\n", outputPath)
}
```

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