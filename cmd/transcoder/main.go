package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/heyjunin/HLSpresso/pkg/hls"
	"github.com/heyjunin/HLSpresso/pkg/logger"
	"github.com/heyjunin/HLSpresso/pkg/progress"
	"github.com/heyjunin/HLSpresso/pkg/transcoder"
	"github.com/spf13/cobra"
)

var (
	// Input options
	inputPath      string
	isRemoteInput  bool
	streamFromURL  bool
	downloadDir    string
	allowOverwrite bool

	// Output options
	outputPath string
	outputType string

	// HLS options
	hlsSegmentDuration int
	hlsPlaylistType    string

	// Advanced options
	ffmpegBinary       string
	ffmpegExtraParams  []string
	progressFilePath   string
	progressFileFormat string
)

func main() {
	// Initialize logger
	logger.Init()

	// Create root command
	rootCmd := &cobra.Command{
		Use:   "HLSpresso",
		Short: "☕ HLSpresso - Tool for generating HLS adaptive streams",
		Long: `☕ HLSpresso - A powerful video transcoding tool that converts video files to HLS adaptive streaming format.
It can download videos from remote URLs and generate multiple quality levels.`,
		Run: runTranscoder,
	}

	// Input flags
	rootCmd.Flags().StringVarP(&inputPath, "input", "i", "", "Input file path or URL (required)")
	rootCmd.Flags().BoolVar(&isRemoteInput, "remote", false, "Treat input as a remote URL (downloads first)")
	rootCmd.Flags().BoolVar(&streamFromURL, "stream", false, "Attempt to stream directly from input URL (implies remote)")
	rootCmd.Flags().StringVar(&downloadDir, "download-dir", "downloads", "Directory to save downloaded files (if not streaming)")
	rootCmd.Flags().BoolVar(&allowOverwrite, "overwrite", false, "Allow overwriting existing files")

	// Output flags
	rootCmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output directory or file path (required)")
	rootCmd.Flags().StringVarP(&outputType, "type", "t", "hls", "Output type: 'hls' or 'mp4'")

	// HLS options
	rootCmd.Flags().IntVar(&hlsSegmentDuration, "hls-segment-duration", 10, "HLS segment duration in seconds")
	rootCmd.Flags().StringVar(&hlsPlaylistType, "hls-playlist-type", "vod", "HLS playlist type (vod or event)")

	// Advanced options
	rootCmd.Flags().StringVar(&ffmpegBinary, "ffmpeg", "ffmpeg", "Path to ffmpeg binary")
	rootCmd.Flags().StringArrayVar(&ffmpegExtraParams, "ffmpeg-param", []string{}, "Extra parameters to pass to ffmpeg")
	rootCmd.Flags().StringVar(&progressFilePath, "progress-file", "", "Path to file for writing progress percentage (e.g., progress.txt)")
	rootCmd.Flags().StringVar(&progressFileFormat, "progress-file-format", "text", "Format for progress file: 'text' (percentage only) or 'json' (full event)")

	// Mark required flags
	rootCmd.MarkFlagRequired("input")
	rootCmd.MarkFlagRequired("output")

	// Execute command
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runTranscoder(cmd *cobra.Command, args []string) {
	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-signalChan
		logger.Info("Received signal, shutting down", "main", map[string]interface{}{
			"signal": sig.String(),
		})
		cancel()
	}()

	// Validate progress file format
	progressFileFormatLower := strings.ToLower(progressFileFormat)
	if progressFileFormatLower != "text" && progressFileFormatLower != "json" {
		logger.Fatal("Invalid --progress-file-format value. Must be 'text' or 'json'", "main", map[string]interface{}{
			"value": progressFileFormat,
		})
		return
	}

	// Create progress reporter with options
	reporterOpts := []progress.ReporterOption{}
	if progressFilePath != "" {
		reporterOpts = append(reporterOpts, progress.WithProgressFile(progressFilePath))
		reporterOpts = append(reporterOpts, progress.WithProgressFileFormat(progressFileFormatLower))
	}
	progressReporter := progress.NewReporter(reporterOpts...)

	// Determine output type
	var outType transcoder.OutputType
	switch strings.ToLower(outputType) {
	case "hls":
		outType = transcoder.HLSOutput
	case "mp4":
		outType = transcoder.MP4Output
	default:
		logger.Fatal("Invalid output type", "main", map[string]interface{}{
			"type": outputType,
		})
		return
	}

	// Determine input type and streaming
	isActuallyRemote := isRemoteInput || streamFromURL || (strings.HasPrefix(inputPath, "http://") || strings.HasPrefix(inputPath, "https://"))
	if streamFromURL && !isActuallyRemote {
		logger.Fatal("--stream flag can only be used with URL inputs", "main", nil)
		return
	}
	if !streamFromURL && isActuallyRemote && !isRemoteInput {
		// Auto-detected URL without --remote or --stream, default to download
		logger.Info("Detected URL input, using download mode (use --stream to stream directly)", "main", nil)
	}

	// Create transcoder options
	options := transcoder.Options{
		// Input options
		InputPath:      inputPath,
		IsRemoteInput:  isActuallyRemote, // Set based on --remote, --stream, or URL detection
		StreamFromURL:  streamFromURL,    // Set by the --stream flag
		DownloadDir:    downloadDir,
		AllowOverwrite: allowOverwrite,

		// Output options
		OutputPath: outputPath,
		OutputType: outType,

		// HLS options
		HLSSegmentDuration: hlsSegmentDuration,
		HLSPlaylistType:    hlsPlaylistType,
		HLSResolutions:     hls.DefaultResolutions,

		// Advanced options
		FFmpegBinary:      ffmpegBinary,
		FFmpegExtraParams: ffmpegExtraParams,
	}

	// Create transcoder
	trans, err := transcoder.New(options, progressReporter)
	if err != nil {
		logger.Fatal("Failed to create transcoder", "main", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// Start transcoding
	logger.Info("Starting transcoder", "main", map[string]interface{}{
		"input":  inputPath,
		"output": outputPath,
		"type":   outputType,
	})

	// Perform transcoding
	outputFilePath, err := trans.Transcode(ctx)
	if err != nil {
		logger.Fatal("Transcoding failed", "main", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// Log success
	absPath, _ := filepath.Abs(outputFilePath)
	logger.Info("Transcoding completed successfully", "main", map[string]interface{}{
		"output_path": absPath,
	})
}
