package hls

import (
	"fmt"
	"math"
)

// DefaultBitrates provides recommended bitrate settings (video, maxrate, bufsize, audio)
// for common resolution names (e.g., "1080p", "720p").
// These are used by GenerateAutoResolutions to select appropriate bitrates.
var (
	DefaultBitrates = map[string]struct {
		Video   string
		MaxRate string
		BufSize string
		Audio   string
	}{
		"2160p": {"15000k", "16050k", "22500k", "192k"}, // 4K
		"1440p": {"9000k", "9630k", "13500k", "192k"},   // 2K/QHD
		"1080p": {"5000k", "5350k", "7500k", "192k"},    // Full HD
		"720p":  {"2800k", "2996k", "4200k", "128k"},    // HD
		"480p":  {"1400k", "1498k", "2100k", "96k"},     // SD
		"360p":  {"800k", "856k", "1200k", "64k"},       // Low
		"240p":  {"400k", "428k", "600k", "48k"},        // Very Low
	}
)

// GenerateAutoResolutions analyzes the original video dimensions (width, height)
// and generates a suitable set of VideoResolution structs for HLS adaptive streaming.
// It preserves the aspect ratio of the input video and includes the original resolution
// along with lower standard resolutions (like 1080p, 720p, 480p etc.) that are
// smaller than the original.
// It avoids upscaling (generating resolutions higher than the original).
// The bitrates for each generated resolution are based on the DefaultBitrates map.
func GenerateAutoResolutions(originalWidth, originalHeight int) []VideoResolution {
	// Determinar se é vídeo vertical ou horizontal
	isVertical := originalHeight > originalWidth

	// Calcular aspect ratio (sempre como width:height)
	var aspectRatio float64
	if isVertical {
		aspectRatio = float64(originalWidth) / float64(originalHeight)
	} else {
		aspectRatio = float64(originalHeight) / float64(originalWidth)
	}

	// Arredondar para evitar problemas de precisão
	aspectRatio = math.Round(aspectRatio*100) / 100

	// Selecionar as resoluções possíveis baseadas na resolução original
	var resolutions []VideoResolution

	// Adicionamos sempre a resolução original primeiro com a melhor qualidade
	var originalQuality string
	originalMax := math.Max(float64(originalWidth), float64(originalHeight))

	// Determinar a qualidade da resolução original
	switch {
	case originalMax >= 2160:
		originalQuality = "2160p"
	case originalMax >= 1440:
		originalQuality = "1440p"
	case originalMax >= 1080:
		originalQuality = "1080p"
	case originalMax >= 720:
		originalQuality = "720p"
	case originalMax >= 480:
		originalQuality = "480p"
	case originalMax >= 360:
		originalQuality = "360p"
	default:
		originalQuality = "240p"
	}

	// Adicionar a resolução original primeiro
	bitrates := DefaultBitrates[originalQuality]
	resolutions = append(resolutions, VideoResolution{
		Width:        originalWidth,
		Height:       originalHeight,
		VideoBitrate: bitrates.Video,
		MaxRate:      bitrates.MaxRate,
		BufSize:      bitrates.BufSize,
		AudioBitrate: bitrates.Audio,
	})

	// Lista de resoluções padrão a considerar (do maior para o menor)
	standardResolutions := []struct {
		name   string
		size   int
		minDim int
	}{
		{"1080p", 1080, 1080},
		{"720p", 720, 720},
		{"480p", 480, 480},
		{"360p", 360, 360},
		{"240p", 240, 240},
	}

	// Adicionar resoluções inferiores à original
	for _, res := range standardResolutions {
		// Pular se esta qualidade for igual ou maior que a original
		if res.size >= int(originalMax) {
			continue
		}

		// Calcular dimensões mantendo aspect ratio
		var width, height int
		if isVertical {
			// Para vídeos verticais, a altura é a dimensão principal
			height = res.size
			width = int(math.Round(float64(height) * aspectRatio))

			// Garantir que largura é pelo menos a mínima esperada para esta resolução
			if width < res.minDim/3 {
				width = res.minDim / 3
			}
		} else {
			// Para vídeos horizontais, a largura é a dimensão principal
			width = res.size
			height = int(math.Round(float64(width) * aspectRatio))

			// Garantir que altura é pelo menos a mínima esperada para esta resolução
			if height < res.minDim/3 {
				height = res.minDim / 3
			}
		}

		// Garantir que as dimensões são números pares (requerido por alguns codecs)
		width = width - (width % 2)
		height = height - (height % 2)

		// Pular resoluções muito pequenas
		if width < 160 || height < 90 {
			continue
		}

		// Adicionar esta resolução
		bitrates = DefaultBitrates[res.name]
		resolutions = append(resolutions, VideoResolution{
			Width:        width,
			Height:       height,
			VideoBitrate: bitrates.Video,
			MaxRate:      bitrates.MaxRate,
			BufSize:      bitrates.BufSize,
			AudioBitrate: bitrates.Audio,
		})
	}

	return resolutions
}

// GetAutoResolutionNames returns a slice of strings containing the names (like "1080p", "720p")
// of the resolutions defined in the DefaultBitrates map.
func GetAutoResolutionNames() []string {
	names := []string{}
	for name := range DefaultBitrates {
		names = append(names, name)
	}
	return names
}

// FormatAutoResolutions takes a slice of VideoResolution structs and returns a
// human-readable string representation, typically used for logging.
// Example output: "1920x1080@5000k, 1280x720@2800k, 854x480@1400k"
func FormatAutoResolutions(resolutions []VideoResolution) string {
	result := ""
	for i, res := range resolutions {
		if i > 0 {
			result += ", "
		}
		result += fmt.Sprintf("%dx%d@%s", res.Width, res.Height, res.VideoBitrate)
	}
	return result
}
