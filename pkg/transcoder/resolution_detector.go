package transcoder

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
)

// VideoInfo holds detected resolution and duration information for a video file.
type VideoInfo struct {
	// Width of the video in pixels.
	Width int
	// Height of the video in pixels.
	Height int
	// Duration of the video in seconds.
	Duration float64
}

// FFprobeOutput represents the structure of the JSON output from the ffprobe command
// when using the -show_format and -show_streams flags.
// It's used internally for parsing the ffprobe results.
type FFprobeOutput struct {
	Streams []struct {
		CodecType string `json:"codec_type"`
		Width     int    `json:"width,omitempty"`
		Height    int    `json:"height,omitempty"`
	} `json:"streams"`
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
}

// DetectVideoResolution uses the ffprobe command-line tool (which must be in the system PATH)
// to detect the resolution (width, height) and duration of the specified video file.
// It parses the JSON output from ffprobe.
// The context can be used to cancel the ffprobe execution.
// Returns a VideoInfo struct containing the detected information or an error if ffprobe fails,
// parsing fails, or video stream information cannot be found.
func DetectVideoResolution(ctx context.Context, inputPath string) (*VideoInfo, error) {
	// Preparar comando FFprobe para obter informações do vídeo em formato JSON
	cmd := exec.CommandContext(
		ctx,
		"ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		inputPath,
	)

	// Executar comando e obter saída
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("erro ao executar FFprobe: %w", err)
	}

	// Parsear a saída JSON
	var probeOutput FFprobeOutput
	if err := json.Unmarshal(output, &probeOutput); err != nil {
		return nil, fmt.Errorf("erro ao parsear saída do FFprobe: %w", err)
	}

	// Encontrar o stream de vídeo
	var videoInfo VideoInfo
	foundVideo := false

	for _, stream := range probeOutput.Streams {
		if stream.CodecType == "video" {
			videoInfo.Width = stream.Width
			videoInfo.Height = stream.Height
			foundVideo = true
			break
		}
	}

	if !foundVideo {
		return nil, fmt.Errorf("nenhum stream de vídeo encontrado no arquivo")
	}

	// Verificar se a resolução foi detectada
	if videoInfo.Width == 0 || videoInfo.Height == 0 {
		return nil, fmt.Errorf("não foi possível detectar a resolução do vídeo")
	}

	// Parsear a duração
	if probeOutput.Format.Duration != "" {
		duration, err := strconv.ParseFloat(probeOutput.Format.Duration, 64)
		if err == nil {
			videoInfo.Duration = duration
		}
	}

	return &videoInfo, nil
}
