package hls

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/heyjunin/HLSpresso/pkg/errors"
	"github.com/heyjunin/HLSpresso/pkg/logger"
	"github.com/heyjunin/HLSpresso/pkg/progress"
)

// VideoResolution defines a video resolution and bitrate settings
type VideoResolution struct {
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	VideoBitrate string `json:"video_bitrate"`
	MaxRate      string `json:"max_rate"`
	BufSize      string `json:"buf_size"`
	AudioBitrate string `json:"audio_bitrate"`
}

// DefaultResolutions provides common video resolutions for adaptive streaming
var DefaultResolutions = []VideoResolution{
	{Width: 1920, Height: 1080, VideoBitrate: "5000k", MaxRate: "5350k", BufSize: "7500k", AudioBitrate: "192k"}, // 1080p
	{Width: 1280, Height: 720, VideoBitrate: "2800k", MaxRate: "2996k", BufSize: "4200k", AudioBitrate: "128k"},  // 720p
	{Width: 854, Height: 480, VideoBitrate: "1400k", MaxRate: "1498k", BufSize: "2100k", AudioBitrate: "96k"},    // 480p
}

// Options contains settings for HLS generation
type Options struct {
	InputFile         string
	OutputDir         string
	SegmentDuration   int
	PlaylistType      string
	Resolutions       []VideoResolution
	MasterPlaylist    string
	SegmentFormat     string
	VariantStreamMap  string
	FFmpegBinary      string
	Progress          progress.Reporter
	FFmpegExtraParams []string
}

// Generator handles HLS playlist generation
type Generator struct {
	options Options
}

// New creates a new HLS generator
func New(options Options) *Generator {
	// Set defaults if not specified
	if options.SegmentDuration == 0 {
		options.SegmentDuration = 10
	}
	if options.PlaylistType == "" {
		options.PlaylistType = "vod"
	}
	if options.MasterPlaylist == "" {
		options.MasterPlaylist = "master.m3u8"
	}
	if options.SegmentFormat == "" {
		options.SegmentFormat = "mpegts"
	}
	if options.FFmpegBinary == "" {
		options.FFmpegBinary = "ffmpeg"
	}
	if len(options.Resolutions) == 0 {
		options.Resolutions = DefaultResolutions
	}

	return &Generator{
		options: options,
	}
}

// CreateHLS generates an adaptive HLS stream from the input video
func (g *Generator) CreateHLS(ctx context.Context) (string, error) {
	// Create output directory
	if err := os.MkdirAll(g.options.OutputDir, 0755); err != nil {
		return "", errors.Wrap(err, errors.SystemError, "Failed to create output directory", 1)
	}

	// Create directories for each stream variant
	for i := range g.options.Resolutions {
		streamDir := filepath.Join(g.options.OutputDir, fmt.Sprintf("stream_%d", i))
		if err := os.MkdirAll(streamDir, 0755); err != nil {
			return "", errors.Wrap(err, errors.HLSError, "Failed to create stream directory", 2)
		}
	}

	// Build ffmpeg command arguments
	args := g.buildFFmpegArgs()

	// Log command
	cmd := g.options.FFmpegBinary + " " + strings.Join(args, " ")
	logger.Debug("Executing FFmpeg command", "hls", map[string]interface{}{
		"command": cmd,
	})

	// Prepare command
	ffmpegCmd := exec.CommandContext(ctx, g.options.FFmpegBinary, args...)

	// Capture stderr for progress tracking
	stderr, err := ffmpegCmd.StderrPipe()
	if err != nil {
		return "", errors.Wrap(err, errors.HLSError, "Failed to create stderr pipe", 3)
	}

	// Start the command
	if err := ffmpegCmd.Start(); err != nil {
		return "", errors.Wrap(err, errors.HLSError, "Failed to start ffmpeg", 4)
	}

	// Initialize progress tracking
	totalFrames := int64(0)
	if g.options.Progress != nil {
		totalFrames = estimateTotalFrames(g.options.InputFile)
		if totalFrames > 0 {
			g.options.Progress.Start(totalFrames)
		}
	}

	// Track progress by parsing ffmpeg output
	go func() {
		progressRegex := regexp.MustCompile(`frame=\s*(\d+)`)
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()

			// Parse frame count for progress
			if g.options.Progress != nil && totalFrames > 0 {
				if matches := progressRegex.FindStringSubmatch(line); len(matches) > 1 {
					if frame, err := strconv.ParseInt(matches[1], 10, 64); err == nil {
						g.options.Progress.Update(frame, "transcoding", "Creating HLS stream")
					}
				}
			}

			// Log FFmpeg output
			logger.Debug(line, "ffmpeg", nil)
		}
	}()

	// Wait for command to complete
	err = ffmpegCmd.Wait()
	if err != nil {
		return "", errors.Wrap(err, errors.HLSError, "FFmpeg command failed", 5)
	}

	// Complete progress
	if g.options.Progress != nil {
		g.options.Progress.Complete()
	}

	masterPath := filepath.Join(g.options.OutputDir, g.options.MasterPlaylist)
	logger.Info("HLS generation completed", "hls", map[string]interface{}{
		"master_playlist": masterPath,
	})

	return masterPath, nil
}

// buildFFmpegArgs constructs the ffmpeg command arguments
func (g *Generator) buildFFmpegArgs() []string {
	args := []string{
		"-i", g.options.InputFile,
		"-filter_complex",
	}

	// Build filter graph for video splits and scaling
	filter := buildFilterGraph(len(g.options.Resolutions), g.options.Resolutions)
	args = append(args, filter)

	// Add output options for each resolution
	for i, res := range g.options.Resolutions {
		// Video stream options
		args = append(args,
			"-map", fmt.Sprintf("[v%dout]", i),
			"-c:v:"+fmt.Sprintf("%d", i), "libx264",
			"-b:v:"+fmt.Sprintf("%d", i), res.VideoBitrate,
			"-maxrate:v:"+fmt.Sprintf("%d", i), res.MaxRate,
			"-bufsize:v:"+fmt.Sprintf("%d", i), res.BufSize,
		)

		// Audio stream options
		args = append(args,
			"-map", "a:0",
			"-c:a:"+fmt.Sprintf("%d", i), "aac",
			"-b:a:"+fmt.Sprintf("%d", i), res.AudioBitrate,
			"-ac", "2",
		)
	}

	// Add HLS options
	args = append(args,
		"-f", "hls",
		"-hls_time", fmt.Sprintf("%d", g.options.SegmentDuration),
		"-hls_playlist_type", g.options.PlaylistType,
		"-hls_flags", "independent_segments",
		"-hls_segment_type", g.options.SegmentFormat,
		"-hls_segment_filename", filepath.Join(g.options.OutputDir, "stream_%v/data%03d.ts"),
		"-master_pl_name", g.options.MasterPlaylist,
	)

	// Add variant stream map
	streamMap := g.options.VariantStreamMap
	if streamMap == "" {
		// Build default stream map if not provided
		var mapParts []string
		for i := range g.options.Resolutions {
			mapParts = append(mapParts, fmt.Sprintf("v:%d,a:%d", i, i))
		}
		streamMap = strings.Join(mapParts, " ")
	}

	args = append(args, "-var_stream_map", streamMap)

	// Add output pattern
	args = append(args, filepath.Join(g.options.OutputDir, "stream_%v/playlist.m3u8"))

	// Add any extra parameters
	args = append(args, g.options.FFmpegExtraParams...)

	return args
}

// buildFilterGraph constructs the FFmpeg filter graph for video splits and scaling
func buildFilterGraph(numStreams int, resolutions []VideoResolution) string {
	// Create video split
	filter := fmt.Sprintf("[0:v]split=%d", numStreams)

	// Add labels for each split output
	for i := 0; i < numStreams; i++ {
		filter += fmt.Sprintf("[v%d]", i)
	}
	filter += "; "

	// Add scaling for each resolution
	for i, res := range resolutions {
		filter += fmt.Sprintf("[v%d]scale=w=%d:h=%d[v%dout]; ", i, res.Width, res.Height, i)
	}

	// Remove trailing semicolon and space
	filter = strings.TrimSuffix(filter, "; ")

	return filter
}

// estimateTotalFrames tries to get the total frame count from the input video
func estimateTotalFrames(inputFile string) int64 {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-count_packets",
		"-show_entries", "stream=nb_read_packets",
		"-of", "csv=p=0",
		inputFile)

	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	frames, err := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64)
	if err != nil {
		return 0
	}

	return frames
}
