package hls

import (
	"fmt"
	"math"
)

// Conjunto padrão de qualidades para diferentes resoluções
var (
	// Bitrates recomendados para diferentes resoluções (em kb/s)
	// Seguindo diretrizes comuns de streaming
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

// GenerateAutoResolutions analisa a resolução original e o aspect ratio do vídeo
// e gera um conjunto apropriado de resoluções para streaming adaptativo HLS,
// sem nunca exceder a resolução original (sem upscaling)
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

// GetAutoResolutionNames retorna uma lista de nomes das resoluções disponíveis
func GetAutoResolutionNames() []string {
	names := []string{}
	for name := range DefaultBitrates {
		names = append(names, name)
	}
	return names
}

// FormatAutoResolutions formata a lista de resoluções em uma string para log ou exibição
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
