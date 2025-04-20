package transcoder

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
)

// VideoInfo representa as informações de resolução e duração obtidas de um vídeo
type VideoInfo struct {
	Width    int
	Height   int
	Duration float64
}

// FFprobeOutput representa a estrutura de saída JSON do FFprobe
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

// DetectVideoResolution utiliza FFprobe para detectar a resolução e duração do vídeo
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
