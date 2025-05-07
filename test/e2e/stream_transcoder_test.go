package e2e

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/heyjunin/HLSpresso/pkg/hls"
	"github.com/heyjunin/HLSpresso/pkg/progress"
	"github.com/heyjunin/HLSpresso/pkg/transcoder"
)

const (
	// URL para o vídeo de teste (BigBuckBunny)
	testStreamVideoURL = "https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/BigBuckBunny.mp4"
	// Diretório temporário para saída de testes de stream
	testStreamOutputDir = "../../testdata/temp_stream"
)

// ValidateStreamProgressEvents verifica se os eventos capturados seguem o padrão esperado para streaming
// Não deve haver eventos de download, mas deve haver eventos de transcodificação
func (m *MockProgressReporter) ValidateStreamProgressEvents(t *testing.T) {
	// Deve ter pelo menos 3 eventos: início, algum progresso e conclusão
	if len(m.events) < 3 {
		t.Errorf("Número insuficiente de eventos de progresso: %d (esperado >= 3)", len(m.events))
		return
	}

	// Verificar evento inicial
	if m.events[0].Status != "started" || m.events[0].Percentage != 0 {
		t.Errorf("Evento inicial inválido: %+v", m.events[0])
	}

	// Verificar evento final
	lastIdx := len(m.events) - 1
	if m.events[lastIdx].Status != "completed" || m.events[lastIdx].Percentage != 100 {
		t.Errorf("Evento final inválido: %+v", m.events[lastIdx])
	}

	// Verificar a ausência de eventos de download e a presença de eventos de transcodificação
	found := map[string]bool{
		"downloading": false,
		"transcoding": false,
	}

	events := m.GetEvents()
	for _, event := range events {
		// Verificar eventos de download (não devem existir)
		if event.Step == "downloading" && event.Stage == "Downloading file" {
			found["downloading"] = true
		}

		// Verificar eventos de transcodificação (devem existir)
		if event.Step == "transcoding" || strings.Contains(event.Stage, "ffmpeg") {
			found["transcoding"] = true
		}
	}

	// Não deve ter eventos de download
	if found["downloading"] {
		t.Errorf("Eventos de download encontrados, mas não deveriam existir no modo stream")
	}

	// Deve ter eventos de transcodificação
	if !found["transcoding"] {
		t.Errorf("Eventos de transcodificação não encontrados, mas são necessários")
	}
}

// Função init para garantir que os diretórios necessários existam
func init() {
	// Criar diretório de saída para streams se não existir
	os.MkdirAll(testStreamOutputDir, 0755)
}

// setupStreamTestDirectories cria os diretórios necessários para os testes de stream
func setupStreamTestDirectories(t *testing.T) {
	// Criar diretório de saída se não existir
	if err := os.MkdirAll(testStreamOutputDir, 0755); err != nil {
		t.Fatalf("Erro ao criar diretório de saída para testes de stream: %v", err)
	}
}

// TestTranscodeStreamToHLS testa a transcodificação de vídeo remoto para HLS usando modo stream
func TestTranscodeStreamToHLS(t *testing.T) {
	// Verifique se existem ferramentas FFmpeg instaladas
	if !checkFFmpegInstalled() {
		t.Skip("FFmpeg não encontrado, pulando teste")
		return
	}

	// Verifique se o diretório de teste existe, senão crie-o
	setupStreamTestDirectories(t)

	// Caminho de saída
	outputPath := filepath.Join(testStreamOutputDir, "stream_hls")

	// Criação do reporter de progresso mock para capturar logs
	mockReporter := NewMockProgressReporter()

	// Configuração do transcoder com StreamFromURL definido como true
	options := transcoder.Options{
		InputPath:          testStreamVideoURL,
		IsRemoteInput:      true,
		StreamFromURL:      true, // <--- Modo stream ativado
		AllowOverwrite:     true,
		OutputPath:         outputPath,
		OutputType:         transcoder.HLSOutput,
		HLSPlaylistType:    "vod",
		HLSSegmentDuration: 3, // Use um valor pequeno para testes mais rápidos
		HLSResolutions: []hls.VideoResolution{
			// Use apenas uma resolução para testes mais rápidos
			{Width: 640, Height: 360, VideoBitrate: "800k", MaxRate: "856k", BufSize: "1200k", AudioBitrate: "64k"},
		},
	}

	// Criar contexto com timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Criar transcoder com nosso reporter mock
	trans, err := transcoder.New(options, mockReporter)
	if err != nil {
		t.Fatalf("Erro ao criar transcoder: %v", err)
	}

	// Executar transcodificação
	masterPlaylistPath, err := trans.Transcode(ctx)
	if err != nil {
		t.Fatalf("Erro durante transcodificação para HLS: %v", err)
	}

	// Verificar se a playlist master foi gerada
	if _, err := os.Stat(masterPlaylistPath); os.IsNotExist(err) {
		t.Errorf("Playlist master não foi gerada em: %s", masterPlaylistPath)
	}

	// Verificar se o diretório de stream foi criado
	streamDir := filepath.Join(outputPath, "stream_0")
	if _, err := os.Stat(streamDir); os.IsNotExist(err) {
		t.Errorf("Diretório de stream não foi criado em: %s", streamDir)
	}

	// Verificar se a playlist de variante foi gerada
	variantPlaylistPath := filepath.Join(streamDir, "playlist.m3u8")
	if _, err := os.Stat(variantPlaylistPath); os.IsNotExist(err) {
		t.Errorf("Playlist de variante não foi gerada em: %s", variantPlaylistPath)
	}

	// Verificar se pelo menos um segmento TS foi gerado
	segmentCount := countTSFiles(streamDir)
	if segmentCount == 0 {
		t.Errorf("Nenhum segmento TS foi gerado em: %s", streamDir)
	}

	// Validar os eventos de progresso (expectDownloadEvents=false para modo stream)
	mockReporter.ValidateStreamProgressEvents(t)
}

// TestTranscodeStreamToMP4 testa a transcodificação de vídeo remoto para MP4 usando modo stream
func TestTranscodeStreamToMP4(t *testing.T) {
	// Verifique se existem ferramentas FFmpeg instaladas
	if !checkFFmpegInstalled() {
		t.Skip("FFmpeg não encontrado, pulando teste")
		return
	}

	// Verifique se o diretório de teste existe, senão crie-o
	setupStreamTestDirectories(t)

	// Caminho de saída
	outputPath := filepath.Join(testStreamOutputDir, "stream_output.mp4")

	// Criação do reporter de progresso
	reporter := progress.NewReporter()
	defer reporter.Complete()

	// Configuração do transcoder com StreamFromURL definido como true
	options := transcoder.Options{
		InputPath:      testStreamVideoURL,
		IsRemoteInput:  true,
		StreamFromURL:  true, // <--- Modo stream ativado
		AllowOverwrite: true,
		OutputPath:     outputPath,
		OutputType:     transcoder.MP4Output,
	}

	// Criar contexto com timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Criar transcoder
	trans, err := transcoder.New(options, reporter)
	if err != nil {
		t.Fatalf("Erro ao criar transcoder: %v", err)
	}

	// Executar transcodificação
	outputVideoPath, err := trans.Transcode(ctx)
	if err != nil {
		t.Fatalf("Erro durante transcodificação para MP4: %v", err)
	}

	// Verificar se o arquivo MP4 foi gerado
	if _, err := os.Stat(outputVideoPath); os.IsNotExist(err) {
		t.Errorf("Arquivo MP4 não foi gerado em: %s", outputVideoPath)
	}

	// Verificar tamanho do arquivo (deve ser maior que 0 bytes)
	fileInfo, err := os.Stat(outputVideoPath)
	if err != nil {
		t.Errorf("Não foi possível obter informações do arquivo: %v", err)
	} else if fileInfo.Size() == 0 {
		t.Errorf("Arquivo MP4 gerado está vazio: %s", outputVideoPath)
	}

	// Não podemos validar eventos diretamente com o reporter padrão,
	// mas podemos verificar se o arquivo foi gerado corretamente
	fmt.Printf("Arquivo MP4 gerado com sucesso: %s (Tamanho: %d bytes)\n", 
		outputVideoPath, fileInfo.Size())
}

// TestTranscodeStreamWithAdaptiveResolutions testa a transcodificação HLS com múltiplas resoluções usando modo stream
func TestTranscodeStreamWithAdaptiveResolutions(t *testing.T) {
	// Verifique se existem ferramentas FFmpeg instaladas
	if !checkFFmpegInstalled() {
		t.Skip("FFmpeg não encontrado, pulando teste")
		return
	}

	// Verifique se o diretório de teste existe, senão crie-o
	setupStreamTestDirectories(t)

	// Caminho de saída
	outputPath := filepath.Join(testStreamOutputDir, "stream_adaptive_hls")

	// Criação do reporter de progresso mock para capturar logs
	mockReporter := NewMockProgressReporter()

	// Configuração do transcoder com StreamFromURL definido como true e múltiplas resoluções
	options := transcoder.Options{
		InputPath:          testStreamVideoURL,
		IsRemoteInput:      true,
		StreamFromURL:      true, // <--- Modo stream ativado
		AllowOverwrite:     true,
		OutputPath:         outputPath,
		OutputType:         transcoder.HLSOutput,
		HLSPlaylistType:    "vod",
		HLSSegmentDuration: 4,
		HLSResolutions: []hls.VideoResolution{
			{Width: 1280, Height: 720, VideoBitrate: "2800k", MaxRate: "2996k", BufSize: "4200k", AudioBitrate: "128k"},
			{Width: 854, Height: 480, VideoBitrate: "1400k", MaxRate: "1498k", BufSize: "2100k", AudioBitrate: "96k"},
			{Width: 640, Height: 360, VideoBitrate: "800k", MaxRate: "856k", BufSize: "1200k", AudioBitrate: "64k"},
		},
	}

	// Criar contexto com timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Criar transcoder com nosso reporter mock
	trans, err := transcoder.New(options, mockReporter)
	if err != nil {
		t.Fatalf("Erro ao criar transcoder: %v", err)
	}

	// Executar transcodificação
	masterPlaylistPath, err := trans.Transcode(ctx)
	if err != nil {
		t.Fatalf("Erro durante transcodificação para HLS adaptativo: %v", err)
	}

	// Verificar se a playlist master foi gerada
	if _, err := os.Stat(masterPlaylistPath); os.IsNotExist(err) {
		t.Errorf("Playlist master não foi gerada em: %s", masterPlaylistPath)
	}

	// Verificar o conteúdo da playlist master para confirmar as resoluções
	checkMasterPlaylist(t, masterPlaylistPath, options.HLSResolutions)

	// Verificar as playlists de variantes
	checkVariantPlaylists(t, outputPath, len(options.HLSResolutions))

	// Validar os eventos de progresso (expectDownloadEvents=false para modo stream)
	mockReporter.ValidateStreamProgressEvents(t)
} 