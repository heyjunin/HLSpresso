package transcoder

import (
	"bufio"
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/eusoujuninho/HLSpresso/pkg/downloader"
	"github.com/eusoujuninho/HLSpresso/pkg/errors"
	"github.com/eusoujuninho/HLSpresso/pkg/hls"
	"github.com/eusoujuninho/HLSpresso/pkg/logger"
	"github.com/eusoujuninho/HLSpresso/pkg/progress"
)

// OutputType represents the type of output to generate
type OutputType string

const (
	// MP4Output generates a single MP4 file
	MP4Output OutputType = "mp4"
	// HLSOutput generates HLS adaptive streaming files
	HLSOutput OutputType = "hls"
)

// Options contains settings for the transcoder
type Options struct {
	// Input options
	InputPath      string
	IsRemoteInput  bool
	DownloadDir    string
	AllowOverwrite bool

	// Output options
	OutputPath string
	OutputType OutputType

	// HLS-specific options
	HLSSegmentDuration int
	HLSResolutions     []hls.VideoResolution
	HLSPlaylistType    string

	// Advanced options
	FFmpegBinary      string
	FFmpegExtraParams []string

	// Auto-resolution options
	UseAutoResolutions bool
}

// Transcoder handles video transcoding
type Transcoder struct {
	options    Options
	progRep    progress.Reporter
	logger     logger.Logger
	downloader *downloader.Downloader
}

// New creates a new Transcoder with default dependencies
func New(options Options, progressReporter progress.Reporter) (*Transcoder, error) {
	return NewWithDeps(options, progressReporter, logger.NewLogger(), nil)
}

// NewWithDeps creates a new Transcoder with custom dependencies
func NewWithDeps(options Options, progressReporter progress.Reporter, logger logger.Logger, dl *downloader.Downloader) (*Transcoder, error) {
	// Set defaults if not specified
	if options.OutputType == "" {
		options.OutputType = HLSOutput
	}
	if options.FFmpegBinary == "" {
		options.FFmpegBinary = "ffmpeg"
	}
	if options.DownloadDir == "" {
		options.DownloadDir = "downloads"
	}

	// Validate options
	if options.InputPath == "" {
		return nil, errors.New(errors.ValidationError, "Input path is required", "", 1)
	}
	if options.OutputPath == "" {
		return nil, errors.New(errors.ValidationError, "Output path is required", "", 2)
	}

	// Se não foi fornecido um downloader e estamos processando entrada remota, criar um novo
	var downloaderInstance *downloader.Downloader
	if dl != nil {
		downloaderInstance = dl
	} else if options.IsRemoteInput {
		downloaderInstance = &downloader.Downloader{}
	}

	return &Transcoder{
		options:    options,
		progRep:    progressReporter,
		logger:     logger,
		downloader: downloaderInstance,
	}, nil
}

// Transcode performs the video transcoding process
func (t *Transcoder) Transcode(ctx context.Context) (string, error) {
	// Primeiro, verificar se o FFmpeg está disponível
	if err := t.checkFFmpeg(); err != nil {
		return "", err
	}

	// Processar o input (download se for remoto)
	inputPath, err := t.handleInput(ctx)
	if err != nil {
		return "", err
	}

	outputPath := t.options.OutputPath

	// Se estiver usando resolução automática e o tipo de saída for HLS,
	// detectar a resolução do vídeo de entrada e configurar as resoluções HLS
	if t.options.UseAutoResolutions && t.options.OutputType == HLSOutput {
		t.logger.Info("Detectando resolução do vídeo para configuração automática", "transcoder", nil)

		// Detectar a resolução do vídeo
		videoInfo, err := DetectVideoResolution(ctx, inputPath)
		if err != nil {
			return "", fmt.Errorf("erro ao detectar resolução do vídeo: %w", err)
		}

		t.logger.Info("Resolução do vídeo detectada", "transcoder", map[string]interface{}{
			"width":    videoInfo.Width,
			"height":   videoInfo.Height,
			"duration": videoInfo.Duration,
		})

		// Gerar resoluções automáticas com base na resolução detectada
		autoResolutions := hls.GenerateAutoResolutions(videoInfo.Width, videoInfo.Height)

		// Registrar as resoluções que serão usadas
		t.logger.Info("Usando resoluções automáticas", "transcoder", map[string]interface{}{
			"resolutions": hls.FormatAutoResolutions(autoResolutions),
		})

		// Atualizar as opções com as resoluções automáticas
		t.options.HLSResolutions = autoResolutions
	}

	// Transcodificar de acordo com o tipo de saída
	switch t.options.OutputType {
	case MP4Output:
		t.logger.Info("Transcoding to MP4", "transcoder", map[string]interface{}{
			"input":  inputPath,
			"output": outputPath,
		})
		return t.transcodeToMP4(ctx, inputPath, outputPath)
	case HLSOutput:
		t.logger.Info("Creating HLS adaptive streams", "transcoder", map[string]interface{}{
			"input":  inputPath,
			"output": outputPath,
		})
		return t.createHLSStreams(ctx, inputPath, outputPath)
	default:
		return "", fmt.Errorf("tipo de saída desconhecido: %s", t.options.OutputType)
	}
}

// handleInput processes the input path (downloading if needed)
func (t *Transcoder) handleInput(ctx context.Context) (string, error) {
	// If input is not remote or explicit flag set to false, just return the path
	if !t.options.IsRemoteInput {
		// Check if file exists locally
		if _, err := os.Stat(t.options.InputPath); os.IsNotExist(err) {
			return "", errors.New(errors.ValidationError, "Input file does not exist", t.options.InputPath, 4)
		}
		return t.options.InputPath, nil
	}

	// Input is a URL, download it
	t.logger.Info("Downloading remote input", "transcoder", map[string]interface{}{
		"url": t.options.InputPath,
	})

	// Parse URL to validate and extract filename
	parsedURL, err := url.Parse(t.options.InputPath)
	if err != nil {
		return "", errors.Wrap(err, errors.ValidationError, "Invalid input URL", 5)
	}

	// Extract filename from URL path
	urlPath := parsedURL.Path
	fileName := filepath.Base(urlPath)
	if fileName == "" || fileName == "." || fileName == "/" {
		fileName = fmt.Sprintf("download_%d.mp4", time.Now().Unix())
	}

	// Create download directory
	if err := os.MkdirAll(t.options.DownloadDir, 0755); err != nil {
		return "", errors.Wrap(err, errors.SystemError, "Failed to create download directory", 6)
	}

	// Set output path for download
	downloadPath := filepath.Join(t.options.DownloadDir, fileName)

	// Se não tivermos um downloader, criar um especificamente para este download
	var downloadedPath string
	if t.downloader == nil {
		// Initialize downloader com opções para este download específico
		dl := downloader.New(downloader.Options{
			URL:           t.options.InputPath,
			OutputPath:    downloadPath,
			Timeout:       30 * time.Minute,
			Progress:      t.progRep,
			AllowOverride: t.options.AllowOverwrite,
		})

		// Download file
		downloadedPath, err = dl.Download(ctx)
		if err != nil {
			return "", errors.Wrap(err, errors.DownloadError, "Failed to download input file", 7)
		}
	} else {
		// Usar o downloader existente, primeiro configurando-o para esta tarefa
		t.downloader = downloader.New(downloader.Options{
			URL:           t.options.InputPath,
			OutputPath:    downloadPath,
			Timeout:       30 * time.Minute,
			Progress:      t.progRep,
			AllowOverride: t.options.AllowOverwrite,
		})

		// Download file
		downloadedPath, err = t.downloader.Download(ctx)
		if err != nil {
			return "", errors.Wrap(err, errors.DownloadError, "Failed to download input file", 7)
		}
	}

	return downloadedPath, nil
}

// createHLS generates HLS adaptive streaming files
func (t *Transcoder) createHLS(ctx context.Context, inputPath string) (string, error) {
	t.logger.Info("Creating HLS adaptive streams", "transcoder", map[string]interface{}{
		"input":  inputPath,
		"output": t.options.OutputPath,
	})

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(t.options.OutputPath, 0755); err != nil {
		return "", errors.Wrap(err, errors.SystemError, "Failed to create output directory", 8)
	}

	// Set HLS options
	hlsOptions := hls.Options{
		InputFile:         inputPath,
		OutputDir:         t.options.OutputPath,
		SegmentDuration:   t.options.HLSSegmentDuration,
		PlaylistType:      t.options.HLSPlaylistType,
		Resolutions:       t.options.HLSResolutions,
		FFmpegBinary:      t.options.FFmpegBinary,
		FFmpegExtraParams: t.options.FFmpegExtraParams,
		Progress:          t.progRep,
	}

	// Create HLS generator
	hlsGen := hls.New(hlsOptions)

	// Generate HLS streams
	masterPlaylistPath, err := hlsGen.CreateHLS(ctx)
	if err != nil {
		return "", errors.Wrap(err, errors.HLSError, "Failed to create HLS streams", 9)
	}

	return masterPlaylistPath, nil
}

// createMP4 generates a single MP4 output file
func (t *Transcoder) createMP4(ctx context.Context, inputPath string) (string, error) {
	t.logger.Info("Transcoding to MP4", "transcoder", map[string]interface{}{
		"input":  inputPath,
		"output": t.options.OutputPath,
	})

	// Create output directory if needed
	outputDir := filepath.Dir(t.options.OutputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", errors.Wrap(err, errors.SystemError, "Failed to create output directory", 10)
	}

	// Build FFmpeg command for MP4 output
	args := []string{
		"-i", inputPath,
		"-c:v", "libx264",
		"-preset", "medium",
		"-crf", "22",
		"-c:a", "aac",
		"-b:a", "128k",
	}

	// Add any extra parameters
	args = append(args, t.options.FFmpegExtraParams...)

	// Add output path
	args = append(args, "-y", t.options.OutputPath)

	// Log FFmpeg command
	cmdStr := t.options.FFmpegBinary + " " + strings.Join(args, " ")
	t.logger.Debug("Executing FFmpeg command", "ffmpeg", map[string]interface{}{
		"command": cmdStr,
	})

	// Run FFmpeg command
	cmd := exec.CommandContext(ctx, t.options.FFmpegBinary, args...)

	// Capture stderr for progress tracking
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", errors.Wrap(err, errors.TranscodingError, "Failed to create stderr pipe", 11)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return "", errors.Wrap(err, errors.TranscodingError, "Failed to start FFmpeg", 12)
	}

	// Get total duration to estimate progress
	totalDuration := getVideoDuration(inputPath)
	if totalDuration > 0 && t.progRep != nil {
		// Convert to frames at approx 25 fps for progress reporting
		totalFrames := int64(totalDuration * 25)
		t.progRep.Start(totalFrames)
	}

	// Process FFmpeg output for progress
	go func() {
		scanner := bufio.NewScanner(stderr)
		timeRegex := regexp.MustCompile(`time=(\d+):(\d+):(\d+\.\d+)`)

		for scanner.Scan() {
			line := scanner.Text()

			// Log FFmpeg output
			t.logger.Debug(line, "ffmpeg", nil)

			// Parse time for progress
			if t.progRep != nil && totalDuration > 0 {
				if matches := timeRegex.FindStringSubmatch(line); len(matches) > 3 {
					hours, _ := strconv.Atoi(matches[1])
					minutes, _ := strconv.Atoi(matches[2])
					seconds, _ := strconv.ParseFloat(matches[3], 64)

					currentTime := float64(hours*3600) + float64(minutes*60) + seconds
					progress := int64((currentTime / totalDuration) * 100)

					// Update progress
					t.progRep.Update(progress, "transcoding", "Creating MP4")
				}
			}
		}
	}()

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		return "", errors.Wrap(err, errors.TranscodingError, "FFmpeg command failed", 13)
	}

	// Complete progress
	if t.progRep != nil {
		t.progRep.Complete()
	}

	return t.options.OutputPath, nil
}

// getVideoDuration gets the duration of a video file in seconds
func getVideoDuration(filePath string) float64 {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		filePath)

	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	duration, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil {
		return 0
	}

	return duration
}

// checkFFmpeg checks if FFmpeg is available
func (t *Transcoder) checkFFmpeg() error {
	cmd := exec.Command(t.options.FFmpegBinary, "-version")
	if err := cmd.Run(); err != nil {
		return errors.New(errors.SystemError, "FFmpeg is not available", "", 14)
	}
	return nil
}

// transcodeToMP4 transcodes the video to MP4
func (t *Transcoder) transcodeToMP4(ctx context.Context, inputPath, outputPath string) (string, error) {
	return t.createMP4(ctx, inputPath)
}

// createHLSStreams creates HLS adaptive streaming files
func (t *Transcoder) createHLSStreams(ctx context.Context, inputPath, outputPath string) (string, error) {
	return t.createHLS(ctx, inputPath)
}
