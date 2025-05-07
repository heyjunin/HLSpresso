package transcoder

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/heyjunin/HLSpresso/pkg/downloader"
	"github.com/heyjunin/HLSpresso/pkg/errors"
	"github.com/heyjunin/HLSpresso/pkg/hls"
	"github.com/heyjunin/HLSpresso/pkg/logger"
	"github.com/heyjunin/HLSpresso/pkg/progress"
	"golang.org/x/sys/unix"
	stderrors "errors" // Renomeado para evitar conflito
)

// OutputType represents the type of output to generate (HLS or MP4).
type OutputType string

const (
	// MP4Output specifies that the output should be a single MP4 file.
	MP4Output OutputType = "mp4"
	// HLSOutput specifies that the output should be HLS adaptive streaming files
	// (manifests and segments).
	HLSOutput OutputType = "hls"
)

// Options contains settings for configuring the Transcoder.
type Options struct {
	// InputPath is the path to the local input video file or a URL if IsRemoteInput is true.
	InputPath string
	// IsRemoteInput indicates whether the InputPath should be treated as a remote URL
	// to be downloaded first.
	IsRemoteInput bool
	// DownloadDir specifies the directory where remote files should be downloaded.
	// Defaults to "downloads" if not set. Only used if IsRemoteInput is true.
	DownloadDir string
	// AllowOverwrite allows the transcoder to overwrite existing output files or
	// downloaded files without error.
	AllowOverwrite bool

	// OutputPath specifies the destination for the transcoded output.
	// For HLSOutput, this should be a directory where manifests and segments will be stored.
	// For MP4Output, this should be the full path to the output MP4 file.
	OutputPath string
	// OutputType determines the format of the output (HLS or MP4).
	// Defaults to HLSOutput if not set.
	OutputType OutputType

	// HLSSegmentDuration sets the target duration for HLS segments in seconds.
	// Only used if OutputType is HLSOutput. Defaults to 10.
	HLSSegmentDuration int
	// HLSResolutions defines the specific quality levels (resolution, bitrates)
	// for HLS adaptive streaming. If nil or empty and UseAutoResolutions is false,
	// default resolutions will be used.
	// Only used if OutputType is HLSOutput.
	HLSResolutions []hls.VideoResolution
	// HLSPlaylistType specifies the HLS playlist type ("vod" or "event").
	// Only used if OutputType is HLSOutput. Defaults to "vod".
	HLSPlaylistType string

	// FFmpegBinary allows specifying a custom path to the ffmpeg executable.
	// Defaults to "ffmpeg" if not set (assuming it's in the system PATH).
	FFmpegBinary string
	// FFmpegExtraParams provides a way to pass additional command-line arguments
	// directly to the underlying ffmpeg process. Use with caution.
	FFmpegExtraParams []string

	// UseAutoResolutions, if true, attempts to detect the input video's resolution
	// and automatically generates a set of suitable HLS resolutions, overriding
	// the HLSResolutions field.
	// Only used if OutputType is HLSOutput.
	UseAutoResolutions bool

	// StreamFromURL, if true and InputPath is a URL, instructs the transcoder to
	// attempt streaming directly from the URL via ffmpeg instead of downloading
	// the file first. This requires ffmpeg to have network access and support
	// for the URL's protocol. The Downloader is not used in this mode.
	// Defaults to false.
	StreamFromURL bool
}

// Transcoder handles the video transcoding process.
// It should be created using New() or NewWithDeps().
type Transcoder struct {
	options    Options
	progRep    progress.Reporter
	logger     logger.Logger
	downloader *downloader.Downloader
}

// New creates a new Transcoder with the given options and progress reporter.
// It uses default implementations for logging and downloading.
// If the input is remote (URL) and StreamFromURL is false (default),
// it automatically provides a basic downloader instance.
// Returns an error if the provided options are invalid.
func New(options Options, progressReporter progress.Reporter) (*Transcoder, error) {
	// Determine if input is remote early to decide on default downloader
	isRemote, _ := url.ParseRequestURI(options.InputPath)
	isRemoteInput := (isRemote != nil && (isRemote.Scheme == "http" || isRemote.Scheme == "https"))

	// Create a default downloader only if needed (remote input and not streaming)
	var defaultDownloader *downloader.Downloader
	if isRemoteInput && !options.StreamFromURL {
		defaultDownloader = &downloader.Downloader{}
	}

	return NewWithDeps(options, progressReporter, logger.NewLogger(), defaultDownloader)
}

// NewWithDeps creates a new Transcoder with custom dependencies.
// This allows injecting specific logger or downloader implementations, useful for testing
// or advanced integration.
//
// Note on downloader:
// - If options.InputPath is a URL and options.StreamFromURL is false, a non-nil downloader (dl) *must* be provided.
// - If options.StreamFromURL is true, the downloader (dl) is ignored and can be nil.
// - If options.InputPath is a local path, the downloader (dl) is ignored and can be nil.
//
// Returns an error if the provided options are invalid (e.g., missing paths, missing downloader when required).
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

	// Check if input is remote
	isRemote, _ := url.ParseRequestURI(options.InputPath)
	options.IsRemoteInput = (isRemote != nil && (isRemote.Scheme == "http" || isRemote.Scheme == "https"))

	// Validate downloader requirement
	if options.IsRemoteInput && !options.StreamFromURL && dl == nil {
		// Se não foi fornecido um downloader e estamos processando entrada remota
		// E não estamos no modo StreamFromURL, retornar erro.
		// (No modo StreamFromURL, o downloader não é necessário)
		return nil, errors.New(errors.ValidationError, "Downloader dependency is required for remote inputs when StreamFromURL is false", "", 3)
	}

	return &Transcoder{
		options:    options,
		progRep:    progressReporter,
		logger:     logger,
		downloader: dl, // Assign the provided downloader (can be nil if not needed)
	}, nil
}

// Transcode executes the video transcoding process based on the options the Transcoder
// was initialized with.
// The context can be used to cancel the transcoding operation (e.g., on timeout or user request).
// It returns the path to the primary output file (e.g., the main HLS manifest or the MP4 file)
// upon successful completion, or an error if the process fails. The error may be a
// *errors.StructuredError containing more details.
func (t *Transcoder) Transcode(ctx context.Context) (string, error) {
	// Primeiro, verificar se o FFmpeg está disponível
	if err := t.checkFFmpeg(); err != nil {
		return "", err
	}

	// Processar o input (baixar se for remoto e não estiver no modo stream)
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

// handleInput processes the input path. If the input is a remote URL and
// StreamFromURL is false, it downloads the file first. Otherwise, it returns
// the original InputPath (either local file or URL for streaming).
// Returns the path/URL to be used as input for ffmpeg, or an error.
func (t *Transcoder) handleInput(ctx context.Context) (string, error) {
	// If input is a URL and StreamFromURL is true, use the URL directly.
	if t.options.IsRemoteInput && t.options.StreamFromURL {
		t.logger.Info("Streaming directly from URL", "transcoder", map[string]interface{}{
			"url": t.options.InputPath,
		})
		// Basic validation of the URL format itself
		_, err := url.ParseRequestURI(t.options.InputPath)
		if err != nil {
			return "", errors.Wrap(err, errors.ValidationError, "Invalid input URL for streaming", 5)
		}
		
		// Verificar se a URL é acessível antes de prosseguir
		timeout := time.Second * 10
		client := http.Client{
			Timeout: timeout,
		}
		
		req, err := http.NewRequestWithContext(ctx, http.MethodHead, t.options.InputPath, nil)
		if err != nil {
			return "", errors.Wrap(err, errors.NetworkError, errors.GetErrorMessage(errors.ErrNetworkConnectionFailed), errors.ErrNetworkConnectionFailed)
		}
		
		resp, err := client.Do(req)
		if err != nil {
			if os.IsTimeout(err) {
				return "", errors.Wrap(err, errors.NetworkError, errors.GetErrorMessage(errors.ErrNetworkTimeout), errors.ErrNetworkTimeout)
			}
			
			// Verificar o erro de DNS - precisamos garantir que não seja caso-sensível e inclua variações comuns
			errMsg := strings.ToLower(err.Error())
			if strings.Contains(errMsg, "no such host") || 
			   strings.Contains(errMsg, "lookup") || 
			   strings.Contains(errMsg, "dns") || 
			   strings.Contains(errMsg, "could not resolve") || 
			   strings.Contains(errMsg, "unknown host") {
				return "", errors.Wrap(err, errors.NetworkError, errors.GetErrorMessage(errors.ErrNetworkDNSFailure), errors.ErrNetworkDNSFailure)
			}
			
			// Outros erros de conexão
			if strings.Contains(err.Error(), "dial") || strings.Contains(err.Error(), "connection") {
				return "", errors.Wrap(err, errors.NetworkError, errors.GetErrorMessage(errors.ErrNetworkConnectionFailed), errors.ErrNetworkConnectionFailed)
			}
			
			return "", errors.Wrap(err, errors.NetworkError, errors.GetErrorMessage(errors.ErrNetworkConnectionFailed), errors.ErrNetworkConnectionFailed)
		}
		
		defer resp.Body.Close()
		
		if resp.StatusCode >= 400 {
			return "", errors.New(errors.NetworkError, errors.GetErrorMessage(errors.ErrNetworkServerUnavailable), 
				fmt.Sprintf("Server returned status code %d", resp.StatusCode), errors.ErrNetworkServerUnavailable)
		}
		
		// Verificar se é um formato de vídeo suportado
		contentType := resp.Header.Get("Content-Type")
		if !strings.HasPrefix(contentType, "video/") && !strings.Contains(contentType, "application/octet-stream") {
			return "", errors.New(errors.InvalidFileFormatError, errors.GetErrorMessage(errors.ErrInvalidFileFormat), 
				fmt.Sprintf("Content-Type: %s", contentType), errors.ErrInvalidFileFormat)
		}
		
		return t.options.InputPath, nil // Return the URL
	}

	// If input is not remote, check if the local file exists.
	if !t.options.IsRemoteInput {
		// Verificar existência do arquivo
		info, err := os.Stat(t.options.InputPath)
		if os.IsNotExist(err) {
			return "", errors.New(errors.FileNotFoundError, errors.GetErrorMessage(errors.ErrFileNotFound), 
				t.options.InputPath, errors.ErrFileNotFound)
		}
		
		// Verificar se é realmente um arquivo e não um diretório
		if info.IsDir() {
			return "", errors.New(errors.InvalidFileFormatError, "O caminho fornecido é um diretório, não um arquivo", 
				t.options.InputPath, errors.ErrInvalidFileFormat)
		}
		
		// Verificar permissões de leitura
		file, err := os.Open(t.options.InputPath)
		if err != nil {
			if os.IsPermission(err) {
				return "", errors.New(errors.PermissionError, errors.GetErrorMessage(errors.ErrReadPermissionDenied), 
					t.options.InputPath, errors.ErrReadPermissionDenied)
			}
			return "", errors.Wrap(err, errors.SystemError, "Falha ao abrir o arquivo de entrada", 4)
		}
		file.Close()
		
		// Verificar se o arquivo tem tamanho não-zero
		if info.Size() == 0 {
			return "", errors.New(errors.InvalidFileFormatError, "O arquivo está vazio", 
				t.options.InputPath, errors.ErrCorruptedFile)
		}
		
		// Verificar extensão do arquivo para formatos comuns de vídeo
		ext := strings.ToLower(filepath.Ext(t.options.InputPath))
		supportedFormats := map[string]bool{
			".mp4": true, ".mov": true, ".avi": true, ".mkv": true, ".webm": true,
			".flv": true, ".wmv": true, ".mpeg": true, ".mpg": true, ".m4v": true,
			".3gp": true, ".ts": true, ".mts": true, ".m2ts": true,
		}
		
		if !supportedFormats[ext] {
			return "", errors.New(errors.InvalidFileFormatError, errors.GetErrorMessage(errors.ErrUnsupportedFileFormat), 
				fmt.Sprintf("Extensão: %s", ext), errors.ErrUnsupportedFileFormat)
		}
		
		return t.options.InputPath, nil // Return the local file path
	}

	// --- Download Logic (only runs if IsRemoteInput is true and StreamFromURL is false) ---

	// Ensure downloader is available (validated in constructor, but double-check)
	if t.downloader == nil {
		return "", errors.New(errors.SystemError, "Downloader is required but not available", "", 10) // Should not happen if constructor validation is correct
	}

	t.logger.Info("Downloading remote input before transcoding", "transcoder", map[string]interface{}{
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

	// Verificar espaço em disco antes de iniciar o download
	downloadDir := t.options.DownloadDir
	var stat unix.Statfs_t
	
	// Verificar espaço disponível no diretório de download
	if err := unix.Statfs(downloadDir, &stat); err == nil {
		// Calcular espaço livre em bytes
		freeSpace := stat.Bavail * uint64(stat.Bsize)
		
		// Verificar se há pelo menos 500MB disponíveis (valor arbitrário, ajustar conforme necessário)
		minRequiredSpace := uint64(500 * 1024 * 1024) // 500 MB
		if freeSpace < minRequiredSpace {
			return "", errors.New(errors.DiskSpaceError, errors.GetErrorMessage(errors.ErrDiskSpaceInsufficient), 
				fmt.Sprintf("Espaço disponível: %d bytes", freeSpace), errors.ErrDiskSpaceInsufficient)
		}
	}

	// Create download directory
	if err := os.MkdirAll(t.options.DownloadDir, 0755); err != nil {
		if os.IsPermission(err) {
			return "", errors.Wrap(err, errors.PermissionError, errors.GetErrorMessage(errors.ErrWritePermissionDenied), errors.ErrWritePermissionDenied)
		}
		return "", errors.Wrap(err, errors.SystemError, "Failed to create download directory", 6)
	}

	// Set output path for download
	downloadPath := filepath.Join(t.options.DownloadDir, fileName)

	// Initialize variable for downloaded path
	var downloadedPath string

	// Configurar o downloader existente para esta tarefa
	downloadOptions := downloader.Options{
		URL:           t.options.InputPath,
		OutputPath:    downloadPath,
		Timeout:       30 * time.Minute, // TODO: Make timeout configurable?
		Progress:      t.progRep,
		AllowOverride: t.options.AllowOverwrite,
	}

	// Se um downloader foi injetado, reconfigure-o
	*t.downloader = *downloader.New(downloadOptions)

	// Download file
	downloadedPath, err = t.downloader.Download(ctx)
	if err != nil {
		// Melhorar a tipagem de erros do downloader
		if os.IsPermission(err) {
			return "", errors.Wrap(err, errors.PermissionError, errors.GetErrorMessage(errors.ErrWritePermissionDenied), errors.ErrWritePermissionDenied)
		}
		
		if strings.Contains(err.Error(), "no space") {
			return "", errors.Wrap(err, errors.DiskSpaceError, errors.GetErrorMessage(errors.ErrDiskSpaceInsufficient), errors.ErrDiskSpaceInsufficient)
		}
		
		if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline exceeded") {
			return "", errors.Wrap(err, errors.NetworkError, errors.GetErrorMessage(errors.ErrNetworkTimeout), errors.ErrNetworkTimeout)
		}
		
		if strings.Contains(err.Error(), "connection") || strings.Contains(err.Error(), "dial") {
			return "", errors.Wrap(err, errors.NetworkError, errors.GetErrorMessage(errors.ErrNetworkConnectionFailed), errors.ErrNetworkConnectionFailed)
		}
		
		return "", errors.Wrap(err, errors.DownloadError, "Failed to download input file", 7)
	}

	return downloadedPath, nil
}

// createHLS generates HLS adaptive streaming files based on the transcoder's options.
// Deprecated: This function's logic is now part of the internal createHLSStreams.
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

// createMP4 generates a single MP4 file based on the transcoder's options.
// Deprecated: This function's logic is now part of the internal transcodeToMP4.
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

// checkFFmpeg verifies that FFmpeg is installed and working correctly,
// including checking for essential codecs and dependencies.
func (t *Transcoder) checkFFmpeg() error {
	cmd := exec.Command(t.options.FFmpegBinary, "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, errors.CodecNotFoundError, 
			errors.GetErrorMessage(errors.ErrMissingDependency), 
			errors.ErrMissingDependency)
	}

	outputStr := string(output)
	t.logger.Debug("FFmpeg version info", "ffmpeg", map[string]interface{}{
		"version_output": outputStr,
	})

	// Verificar codecs importantes
	cmd = exec.Command(t.options.FFmpegBinary, "-codecs")
	codecsOutput, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, errors.SystemError, "Falha ao verificar codecs do FFmpeg", 20)
	}

	codecsStr := string(codecsOutput)
	
	// Verificar se os codecs principais estão presentes
	requiredCodecs := []struct {
		name        string
		searchTerm  string
		errorCode   int
	}{
		{"libx264", "libx264", errors.ErrCodecNotFound},
		{"AAC audio", "aac", errors.ErrCodecNotSupported},
	}

	for _, codec := range requiredCodecs {
		if !strings.Contains(codecsStr, codec.searchTerm) {
			return errors.New(errors.CodecNotFoundError, 
				errors.GetErrorMessage(codec.errorCode), 
				fmt.Sprintf("Codec %s não encontrado", codec.name), 
				codec.errorCode)
		}
	}

	return nil
}

// transcodeToMP4 converts the input to MP4 format
func (t *Transcoder) transcodeToMP4(ctx context.Context, inputPath, outputPath string) (string, error) {
	t.logger.Info("Transcoding to MP4", "transcoder", map[string]interface{}{
		"input":  inputPath,
		"output": outputPath,
	})

	// Create output directory if needed
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		if os.IsPermission(err) {
			return "", errors.Wrap(err, errors.PermissionError, 
				errors.GetErrorMessage(errors.ErrWritePermissionDenied), 
				errors.ErrWritePermissionDenied)
		}
		
		// Verificar se há espaço no disco
		var stat unix.Statfs_t
		if unix.Statfs(outputDir, &stat) == nil {
			freeSpace := stat.Bavail * uint64(stat.Bsize)
			if freeSpace < uint64(500*1024*1024) { // 500MB mínimo
				return "", errors.New(errors.DiskSpaceError, 
					errors.GetErrorMessage(errors.ErrDiskSpaceInsufficient), 
					fmt.Sprintf("Espaço livre: %d bytes", freeSpace), 
					errors.ErrDiskSpaceInsufficient)
			}
		}
		
		return "", errors.Wrap(err, errors.SystemError, "Failed to create output directory", 10)
	}

	// Verificar se já existe um arquivo de saída e se podemos sobrescrevê-lo
	if _, err := os.Stat(outputPath); err == nil && !t.options.AllowOverwrite {
		return "", errors.New(errors.InvalidOutputPathError, 
			"O arquivo de saída já existe e a sobrescrita não está permitida", 
			outputPath, errors.ErrInvalidOutputPath)
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
	args = append(args, "-y", outputPath)

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
		if strings.Contains(err.Error(), "no such file") {
			return "", errors.Wrap(err, errors.CodecNotFoundError, 
				errors.GetErrorMessage(errors.ErrMissingDependency), 
				errors.ErrMissingDependency)
		}
		return "", errors.Wrap(err, errors.TranscodingError, "Failed to start FFmpeg", 12)
	}

	// Start progress reader in a goroutine
	go func() {
		t.trackProgress(stderr)
	}()

	// Wait for completion
	if err := cmd.Wait(); err != nil {
		var exitErr *exec.ExitError
		if stderrors.As(err, &exitErr) {
			// Analisar o código de saída para determinar o tipo de erro
			if exitErr.ExitCode() == 1 {
				// Verificar se há problemas comuns de codec
				errOutput := string(exitErr.Stderr)
				if strings.Contains(errOutput, "Unknown encoder") {
					return "", errors.Wrap(err, errors.CodecNotFoundError, 
						errors.GetErrorMessage(errors.ErrCodecNotFound), 
						errors.ErrCodecNotFound)
				}
				
				// Verificar se há problemas de memória
				if strings.Contains(errOutput, "Cannot allocate memory") || 
				   strings.Contains(errOutput, "out of memory") {
					return "", errors.Wrap(err, errors.MemoryError, 
						errors.GetErrorMessage(errors.ErrOutOfMemory), 
						errors.ErrOutOfMemory)
				}
				
				// Verificar problemas com o formato do arquivo
				if strings.Contains(errOutput, "Invalid data") || 
				   strings.Contains(errOutput, "could not find codec parameters") {
					return "", errors.Wrap(err, errors.InvalidFileFormatError, 
						errors.GetErrorMessage(errors.ErrCorruptedFile), 
						errors.ErrCorruptedFile)
				}
			}
		}
		
		return "", errors.Wrap(err, errors.TranscodingError, "FFmpeg process failed", 13)
	}

	// Check if output file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return "", errors.New(errors.TranscodingError, "Output file was not created", outputPath, 14)
	}

	t.logger.Info("Transcoding completed successfully", "transcoder", map[string]interface{}{
		"output_path": outputPath,
	})

	return outputPath, nil
}

// trackProgress lê a saída do FFmpeg em stderr e atualiza o progresso da transcodificação
func (t *Transcoder) trackProgress(stderr io.ReadCloser) {
	scanner := bufio.NewScanner(stderr)
	timeRegex := regexp.MustCompile(`time=(\d+):(\d+):(\d+\.\d+)`)
	
	for scanner.Scan() {
		line := scanner.Text()
		
		// Log FFmpeg output
		t.logger.Debug(line, "ffmpeg", nil)
		
		// Se não temos reporter de progresso, apenas continue registrando a saída
		if t.progRep == nil {
			continue
		}
		
		// Procurar informações de tempo no formato HH:MM:SS.MS
		if matches := timeRegex.FindStringSubmatch(line); len(matches) > 3 {
			hours, _ := strconv.Atoi(matches[1])
			minutes, _ := strconv.Atoi(matches[2])
			seconds, _ := strconv.ParseFloat(matches[3], 64)
			
			// Converter para segundos
			currentTime := float64(hours*3600) + float64(minutes*60) + seconds
			
			// Atualizar o progresso (valor entre 0-100 ou valor absoluto em segundos)
			t.progRep.Update(int64(currentTime), "transcoding", fmt.Sprintf("Processando: %02d:%02d:%05.2f", hours, minutes, seconds))
		}
	}
}

// createHLSStreams converts the input to HLS format with multiple quality levels
func (t *Transcoder) createHLSStreams(ctx context.Context, inputPath, outputPath string) (string, error) {
	t.logger.Info("Creating HLS adaptive streams", "transcoder", map[string]interface{}{
		"input":  inputPath,
		"output": outputPath,
	})

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		if os.IsPermission(err) {
			return "", errors.Wrap(err, errors.PermissionError, 
				errors.GetErrorMessage(errors.ErrWritePermissionDenied), 
				errors.ErrWritePermissionDenied)
		}
		
		// Verificar espaço em disco
		var stat unix.Statfs_t
		if unix.Statfs(outputPath, &stat) == nil {
			freeSpace := stat.Bavail * uint64(stat.Bsize)
			// Para HLS precisamos de mais espaço (múltiplas resoluções)
			if freeSpace < uint64(1024*1024*1024) { // 1GB mínimo
				return "", errors.New(errors.DiskSpaceError, 
					errors.GetErrorMessage(errors.ErrDiskSpaceInsufficient), 
					fmt.Sprintf("Espaço livre: %d bytes", freeSpace), 
					errors.ErrDiskSpaceInsufficient)
			}
		}
		
		return "", errors.Wrap(err, errors.SystemError, "Failed to create output directory", 15)
	}

	// Verificar se o caminho é acessível para escrita
	testFile := filepath.Join(outputPath, "test_write_permission.tmp")
	tmpFile, err := os.Create(testFile)
	if err != nil {
		if os.IsPermission(err) {
			return "", errors.New(errors.PermissionError, 
				errors.GetErrorMessage(errors.ErrWritePermissionDenied), 
				outputPath, errors.ErrWritePermissionDenied)
		}
		return "", errors.Wrap(err, errors.InvalidOutputPathError, 
			errors.GetErrorMessage(errors.ErrOutputPathNotAccessible), 
			errors.ErrOutputPathNotAccessible)
	}
	tmpFile.Close()
	os.Remove(testFile)

	// Verificar resoluções
	for _, res := range t.options.HLSResolutions {
		if res.Width <= 0 || res.Height <= 0 {
			return "", errors.New(errors.UnsupportedResolutionError, 
				errors.GetErrorMessage(errors.ErrInvalidResolution), 
				fmt.Sprintf("Resolução inválida: %dx%d", res.Width, res.Height), 
				errors.ErrInvalidResolution)
		}
		
		// Verificar se a resolução é muito alta (limites arbitrários, ajustar conforme necessário)
		if res.Width > 7680 || res.Height > 4320 { // Limite 8K
			return "", errors.New(errors.UnsupportedResolutionError, 
				errors.GetErrorMessage(errors.ErrResolutionTooHigh), 
				fmt.Sprintf("Resolução muito alta: %dx%d", res.Width, res.Height), 
				errors.ErrResolutionTooHigh)
		}
		
		// Verificar se a resolução é muito baixa
		if res.Width < 128 || res.Height < 96 {
			return "", errors.New(errors.UnsupportedResolutionError, 
				errors.GetErrorMessage(errors.ErrResolutionTooLow), 
				fmt.Sprintf("Resolução muito baixa: %dx%d", res.Width, res.Height), 
				errors.ErrResolutionTooLow)
		}
	}

	// Set HLS options
	hlsOptions := hls.Options{
		InputFile:         inputPath,
		OutputDir:         outputPath,
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
		// Analisar a mensagem de erro para fornecer mais detalhes
		errMsg := err.Error()
		
		if strings.Contains(errMsg, "codec") || strings.Contains(errMsg, "encoder") {
			return "", errors.Wrap(err, errors.CodecNotFoundError, 
				errors.GetErrorMessage(errors.ErrCodecNotFound), 
				errors.ErrCodecNotFound)
		}
		
		if strings.Contains(errMsg, "memory") || strings.Contains(errMsg, "allocate") {
			return "", errors.Wrap(err, errors.MemoryError, 
				errors.GetErrorMessage(errors.ErrOutOfMemory), 
				errors.ErrOutOfMemory)
		}
		
		if strings.Contains(errMsg, "permission") {
			return "", errors.Wrap(err, errors.PermissionError, 
				errors.GetErrorMessage(errors.ErrWritePermissionDenied), 
				errors.ErrWritePermissionDenied)
		}
		
		if strings.Contains(errMsg, "no space") {
			return "", errors.Wrap(err, errors.DiskSpaceError, 
				errors.GetErrorMessage(errors.ErrDiskSpaceInsufficient), 
				errors.ErrDiskSpaceInsufficient)
		}
		
		return "", errors.Wrap(err, errors.HLSError, "Failed to create HLS streams", 16)
	}

	// Verificar se o arquivo master playlist foi criado
	if _, err := os.Stat(masterPlaylistPath); os.IsNotExist(err) {
		return "", errors.New(errors.HLSError, "Master playlist não foi criado", outputPath, 17)
	}

	t.logger.Info("HLS creation completed successfully", "transcoder", map[string]interface{}{
		"master_playlist": masterPlaylistPath,
	})

	return masterPlaylistPath, nil
}
