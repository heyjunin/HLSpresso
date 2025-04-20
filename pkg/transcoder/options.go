package transcoder

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/heyjunin/HLSpresso/pkg/hls"
)

// ValidateOptions verifica se as opções estão configuradas corretamente
func ValidateOptions(opts Options) error {
	// Verificar se o caminho de entrada foi especificado
	if opts.InputPath == "" {
		return fmt.Errorf("caminho de entrada não especificado")
	}

	// Verificar se o caminho de saída foi especificado
	if opts.OutputPath == "" {
		return fmt.Errorf("caminho de saída não especificado")
	}

	// Se for remoto, verificar o diretório de download
	if opts.IsRemoteInput {
		if opts.DownloadDir == "" {
			return fmt.Errorf("diretório de download não especificado para entrada remota")
		}

		// Verificar se o diretório de download existe
		if _, err := os.Stat(opts.DownloadDir); os.IsNotExist(err) {
			// Tentar criar o diretório
			if err := os.MkdirAll(opts.DownloadDir, 0755); err != nil {
				return fmt.Errorf("não foi possível criar o diretório de download: %w", err)
			}
		}
	}

	// Configurar valores padrão para o tipo de saída
	if opts.OutputType == "" {
		// Inferir tipo de saída pelo caminho
		ext := filepath.Ext(opts.OutputPath)
		if ext == ".mp4" {
			opts.OutputType = MP4Output
		} else {
			opts.OutputType = HLSOutput
		}
	}

	// Se o tipo de saída for HLS e não estiver usando resolução automática
	if opts.OutputType == HLSOutput && !opts.UseAutoResolutions {
		// Verificar se as resoluções HLS foram especificadas
		if len(opts.HLSResolutions) == 0 {
			// Configurar um conjunto padrão de resoluções
			opts.HLSResolutions = []hls.VideoResolution{
				{Width: 1280, Height: 720, VideoBitrate: "2800k", MaxRate: "2996k", BufSize: "4200k", AudioBitrate: "128k"},
				{Width: 854, Height: 480, VideoBitrate: "1400k", MaxRate: "1498k", BufSize: "2100k", AudioBitrate: "96k"},
				{Width: 640, Height: 360, VideoBitrate: "800k", MaxRate: "856k", BufSize: "1200k", AudioBitrate: "64k"},
			}
		}
	}

	return nil
}
