package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/heyjunin/HLSpresso/pkg/hls"
	"github.com/heyjunin/HLSpresso/pkg/progress"
	"github.com/heyjunin/HLSpresso/pkg/transcoder"
)

const (
	// URL para um vídeo de teste curto e público para download
	testVideoURL = "https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ForBiggerBlazes.mp4"
	// Caminho para um vídeo de teste local (caso o download falhe)
	localTestVideoPath = "../../testdata/test_video.mp4"
	// Diretório temporário para saída
	testOutputDir = "../../testdata/temp"
	// Diretório de download temporário
	testDownloadDir = "../../testdata/downloads"
)

// MockProgressReporter é um reporter de progresso que captura logs para testes
type MockProgressReporter struct {
	events      []progress.ProgressEvent
	total       int64
	current     int64
	lastStep    string
	lastStage   string
	lastStatus  string
	initialized bool
	completed   bool
}

// NewMockProgressReporter cria um novo reporter mock para testes
func NewMockProgressReporter() *MockProgressReporter {
	return &MockProgressReporter{
		events: []progress.ProgressEvent{},
	}
}

// Start inicia o progresso com o número total de passos
func (m *MockProgressReporter) Start(totalSteps int64) {
	m.total = totalSteps
	m.current = 0
	m.initialized = true
	m.lastStatus = "started"

	// Registrar evento inicial
	event := progress.ProgressEvent{
		Status:     "started",
		Percentage: 0,
		Step:       "",
		Stage:      "",
		Timestamp:  time.Now().Format(time.RFC3339),
	}
	m.events = append(m.events, event)
}

// Update atualiza o progresso atual
func (m *MockProgressReporter) Update(current int64, step, stage string) {
	m.current = current
	m.lastStep = step
	m.lastStage = stage
	m.lastStatus = "processing"

	percentage := 0.0
	if m.total > 0 {
		percentage = float64(current) / float64(m.total) * 100
	}

	// Registrar evento de atualização
	event := progress.ProgressEvent{
		Status:     "processing",
		Percentage: percentage,
		Step:       step,
		Stage:      stage,
		Timestamp:  time.Now().Format(time.RFC3339),
	}
	m.events = append(m.events, event)
}

// Increment incrementa o progresso em 1
func (m *MockProgressReporter) Increment(step, stage string) {
	m.current++
	m.lastStep = step
	m.lastStage = stage
	m.lastStatus = "processing"

	percentage := 0.0
	if m.total > 0 {
		percentage = float64(m.current) / float64(m.total) * 100
	}

	// Registrar evento de incremento
	event := progress.ProgressEvent{
		Status:     "processing",
		Percentage: percentage,
		Step:       step,
		Stage:      stage,
		Timestamp:  time.Now().Format(time.RFC3339),
	}
	m.events = append(m.events, event)
}

// Complete marca o progresso como concluído
func (m *MockProgressReporter) Complete() {
	m.current = m.total
	m.lastStatus = "completed"
	m.completed = true

	// Registrar evento final
	event := progress.ProgressEvent{
		Status:     "completed",
		Percentage: 100,
		Step:       m.lastStep,
		Stage:      m.lastStage,
		Timestamp:  time.Now().Format(time.RFC3339),
	}
	m.events = append(m.events, event)
}

// Close is a no-op implementation for the mock reporter to satisfy the interface.
func (m *MockProgressReporter) Close() {
	// No-op for mock
}

// Updates returns a closed channel, as the mock reporter primarily uses GetEvents().
func (m *MockProgressReporter) Updates() <-chan progress.ProgressEvent {
	ch := make(chan progress.ProgressEvent)
	close(ch)
	return ch
}

// JSON returns the last captured event as JSON, or an error if none exists.
func (m *MockProgressReporter) JSON() (string, error) {
	currentEvent := progress.ProgressEvent{
		Status:     m.lastStatus,
		Percentage: float64(m.current) / float64(m.total) * 100,
		Step:       m.lastStep,
		Stage:      m.lastStage,
		Timestamp:  time.Now().Format(time.RFC3339),
	}

	data, err := json.Marshal(currentEvent)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetEvents retorna todos os eventos capturados
func (m *MockProgressReporter) GetEvents() []progress.ProgressEvent {
	return m.events
}

// ValidateProgressEvents verifica se os eventos capturados seguem o padrão esperado
func (m *MockProgressReporter) ValidateProgressEvents(t *testing.T, expectDownloadEvents bool) {
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

	// Verificar formato e estrutura de todos os eventos
	for i, event := range m.events {
		// Verificar campos obrigatórios
		if event.Status == "" {
			t.Errorf("Evento %d: campo 'status' vazio", i)
		}

		// Verificar timestamp no formato RFC3339
		_, err := time.Parse(time.RFC3339, event.Timestamp)
		if err != nil {
			t.Errorf("Evento %d: timestamp '%s' não está no formato RFC3339: %v",
				i, event.Timestamp, err)
		}

		// Verificar valores lógicos
		if event.Status == "processing" && (event.Step == "" || event.Stage == "") {
			t.Errorf("Evento %d: evento de processamento sem step ou stage: %+v", i, event)
		}

		// Validar JSON
		jsonData, err := json.Marshal(event)
		if err != nil {
			t.Errorf("Evento %d: não foi possível serializar para JSON: %v", i, err)
		}

		// Verificar se o JSON está no formato esperado
		var parsedEvent map[string]interface{}
		if err := json.Unmarshal(jsonData, &parsedEvent); err != nil {
			t.Errorf("Evento %d: JSON inválido: %v", i, err)
		}

		// Verificar campos obrigatórios no JSON
		requiredFields := []string{"status", "percentage", "timestamp"}
		for _, field := range requiredFields {
			if _, exists := parsedEvent[field]; !exists {
				t.Errorf("Evento %d: campo obrigatório '%s' ausente no JSON", i, field)
			}
		}
	}

	// Verificar eventos específicos
	hasDownloadEvent := false
	hasTranscodingEvent := false

	for _, event := range m.events {
		if event.Step == "downloading" {
			hasDownloadEvent = true
		}
		if event.Step == "transcoding" {
			hasTranscodingEvent = true
		}
	}

	if expectDownloadEvents && !hasDownloadEvent {
		t.Errorf("Nenhum evento de download encontrado, mas era esperado")
	}

	if !hasTranscodingEvent {
		t.Errorf("Nenhum evento de transcodificação encontrado")
	}
}

func TestTranscodeLocalVideoToHLS(t *testing.T) {
	// Verifique se existem ferramentas FFmpeg instaladas
	if !checkFFmpegInstalled() {
		t.Skip("FFmpeg não encontrado, pulando teste")
		return
	}

	// Verifique se o diretório de teste existe, senão crie-o
	setupTestDirectories(t)

	// Prepare o arquivo de vídeo local para teste
	inputPath := ensureTestVideoExists(t)

	// Caminho de saída
	outputPath := filepath.Join(testOutputDir, "local_hls")

	// Criação do reporter de progresso
	reporter := progress.NewReporter()

	// Configuração do transcoder
	options := transcoder.Options{
		InputPath:          inputPath,
		IsRemoteInput:      false,
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
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// Criar transcoder
	trans, err := transcoder.New(options, reporter)
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
}

func TestTranscodeRemoteVideoToMP4(t *testing.T) {
	// Verifique se existem ferramentas FFmpeg instaladas
	if !checkFFmpegInstalled() {
		t.Skip("FFmpeg não encontrado, pulando teste")
		return
	}

	// Verifique se o diretório de teste existe, senão crie-o
	setupTestDirectories(t)

	// Caminho de saída
	outputPath := filepath.Join(testOutputDir, "remote_output.mp4")

	// Criação do reporter de progresso
	reporter := progress.NewReporter()
	// Ensure reporter is properly closed at the end of the test
	defer reporter.Complete()

	// Configuração do transcoder
	options := transcoder.Options{
		InputPath:      testVideoURL,
		IsRemoteInput:  true,
		DownloadDir:    testDownloadDir,
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
}

func TestTranscodeRemoteVideoToHLS(t *testing.T) {
	// Verifique se existem ferramentas FFmpeg instaladas
	if !checkFFmpegInstalled() {
		t.Skip("FFmpeg não encontrado, pulando teste")
		return
	}

	// Verifique se o diretório de teste existe, senão crie-o
	setupTestDirectories(t)

	// Caminho de saída
	outputPath := filepath.Join(testOutputDir, "remote_hls")

	// Criação do reporter de progresso mock para capturar logs
	mockReporter := NewMockProgressReporter()

	// Configuração do transcoder
	options := transcoder.Options{
		InputPath:          testVideoURL,
		IsRemoteInput:      true,
		DownloadDir:        testDownloadDir,
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

	// Validar os eventos de progresso e logs JSON (expectDownloadEvents=true para input remoto)
	mockReporter.ValidateProgressEvents(t, true)

	// Verificar se os eventos incluem estados específicos esperados no fluxo
	found := map[string]bool{
		"downloading": false,
		"transcoding": false,
	}

	events := mockReporter.GetEvents()
	for _, event := range events {
		// Verificar eventos de download
		if event.Step == "downloading" && event.Stage == "Downloading file" {
			found["downloading"] = true
		}

		// Verificar eventos de transcodificação HLS
		if event.Step == "transcoding" && strings.Contains(event.Stage, "HLS") {
			found["transcoding"] = true
		}
	}

	// Verificar se todos os estados esperados foram encontrados
	for state, wasFound := range found {
		if !wasFound {
			t.Errorf("Estado esperado '%s' não foi encontrado nos eventos de progresso", state)
		}
	}
}

// parseM3U8File analisa um arquivo M3U8 e retorna as linhas como []string
func parseM3U8File(t *testing.T, filePath string) []string {
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Erro ao ler arquivo M3U8 %s: %v", filePath, err)
	}

	lines := strings.Split(string(data), "\n")
	// Remover linhas vazias
	var result []string
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine != "" {
			result = append(result, trimmedLine)
		}
	}
	return result
}

// checkMasterPlaylist verifica se o arquivo master.m3u8 contém as resoluções esperadas
func checkMasterPlaylist(t *testing.T, masterPlaylistPath string, expectedResolutions []hls.VideoResolution) {
	lines := parseM3U8File(t, masterPlaylistPath)

	// Verificar se contém o cabeçalho HLS
	foundHeader := false
	for _, line := range lines {
		if line == "#EXTM3U" {
			foundHeader = true
			break
		}
	}
	if !foundHeader {
		t.Errorf("Cabeçalho #EXTM3U não encontrado na playlist master")
	}

	// Verificar se há o número correto de variantes
	streamInfoLines := 0
	streamUriLines := 0

	for _, line := range lines {
		if strings.HasPrefix(line, "#EXT-X-STREAM-INF:") {
			streamInfoLines++
		} else if strings.Contains(line, "stream_") && strings.Contains(line, "/playlist.m3u8") {
			streamUriLines++
		}
	}

	if streamInfoLines != len(expectedResolutions) {
		t.Errorf("Número incorreto de definições de stream na playlist master: encontrado %d, esperado %d",
			streamInfoLines, len(expectedResolutions))
	}

	if streamUriLines != len(expectedResolutions) {
		t.Errorf("Número incorreto de URIs de stream na playlist master: encontrado %d, esperado %d",
			streamUriLines, len(expectedResolutions))
	}

	// Verificar se cada resolução está definida
	for _, resolution := range expectedResolutions {
		resolutionString := fmt.Sprintf("%dx%d", resolution.Width, resolution.Height)
		resolutionFound := false

		for _, line := range lines {
			if strings.HasPrefix(line, "#EXT-X-STREAM-INF:") && strings.Contains(line, "RESOLUTION="+resolutionString) {
				resolutionFound = true
				break
			}
		}

		if !resolutionFound {
			t.Errorf("Resolução %s não encontrada na playlist master", resolutionString)
		}
	}
}

// checkVariantPlaylists verifica se foram criadas playlists variantes para cada resolução
func checkVariantPlaylists(t *testing.T, outputDir string, expectedStreams int) {
	for i := 0; i < expectedStreams; i++ {
		streamDir := filepath.Join(outputDir, fmt.Sprintf("stream_%d", i))

		// Verificar se o diretório do stream existe
		if _, err := os.Stat(streamDir); os.IsNotExist(err) {
			t.Errorf("Diretório do stream_%d não encontrado", i)
			continue
		}

		// Verificar se a playlist variante existe
		playlistPath := filepath.Join(streamDir, "playlist.m3u8")
		if _, err := os.Stat(playlistPath); os.IsNotExist(err) {
			t.Errorf("Playlist do stream_%d não encontrada", i)
			continue
		}

		// Verificar o conteúdo da playlist variante
		lines := parseM3U8File(t, playlistPath)

		// Verificar se contém o cabeçalho HLS
		foundHeader := false
		for _, line := range lines {
			if line == "#EXTM3U" {
				foundHeader = true
				break
			}
		}
		if !foundHeader {
			t.Errorf("Cabeçalho #EXTM3U não encontrado na playlist variante %d", i)
		}

		// Verificar se há pelo menos um segmento
		segmentsFound := false
		for _, line := range lines {
			if strings.HasPrefix(line, "#EXTINF:") {
				segmentsFound = true
				break
			}
		}
		if !segmentsFound {
			t.Errorf("Não foram encontrados segmentos na playlist variante %d", i)
		}

		// Verificar se há segmentos TS
		segmentCount := countTSFiles(streamDir)
		if segmentCount == 0 {
			t.Errorf("Nenhum segmento TS encontrado para o stream_%d", i)
		}
	}
}

func TestTranscodeHLSAdaptiveStreaming(t *testing.T) {
	// Verifique se existem ferramentas FFmpeg instaladas
	if !checkFFmpegInstalled() {
		t.Skip("FFmpeg não encontrado, pulando teste")
		return
	}

	// Verifique se o diretório de teste existe, senão crie-o
	setupTestDirectories(t)

	// Prepare o arquivo de vídeo local para teste
	inputPath := ensureTestVideoExists(t)

	// Caminho de saída
	outputPath := filepath.Join(testOutputDir, "adaptive_hls")

	// Definir múltiplas resoluções para teste
	resolutions := []hls.VideoResolution{
		{Width: 1280, Height: 720, VideoBitrate: "2800k", MaxRate: "2996k", BufSize: "4200k", AudioBitrate: "128k"}, // 720p
		{Width: 854, Height: 480, VideoBitrate: "1400k", MaxRate: "1498k", BufSize: "2100k", AudioBitrate: "96k"},   // 480p
		{Width: 640, Height: 360, VideoBitrate: "800k", MaxRate: "856k", BufSize: "1200k", AudioBitrate: "64k"},     // 360p
	}

	// Criação do reporter de progresso mock para capturar logs
	mockReporter := NewMockProgressReporter()

	// Configuração do transcoder
	options := transcoder.Options{
		InputPath:          inputPath,
		IsRemoteInput:      false, // Usamos um arquivo local para este teste
		OutputPath:         outputPath,
		OutputType:         transcoder.HLSOutput,
		HLSPlaylistType:    "vod",
		HLSSegmentDuration: 3, // Use um valor pequeno para testes mais rápidos
		HLSResolutions:     resolutions,
	}

	// Criar contexto com timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Criar transcoder
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

	// Verificar o conteúdo da playlist master
	checkMasterPlaylist(t, masterPlaylistPath, resolutions)

	// Verificar se foram criadas playlists variantes para cada resolução
	checkVariantPlaylists(t, outputPath, len(resolutions))

	// Verificar se os eventos de progresso foram gerados corretamente (expectDownloadEvents=false para input local)
	mockReporter.ValidateProgressEvents(t, false)

	// Verificar logs específicos de transcodificação HLS adaptativo
	foundTranscodingAdaptiveEvents := false
	events := mockReporter.GetEvents()
	for _, event := range events {
		if event.Step == "transcoding" && strings.Contains(event.Stage, "HLS") {
			foundTranscodingAdaptiveEvents = true
			break
		}
	}

	if !foundTranscodingAdaptiveEvents {
		t.Errorf("Não foram encontrados eventos de transcodificação HLS adaptativo")
	}
}

// Funções auxiliares

func checkFFmpegInstalled() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

func setupTestDirectories(t *testing.T) {
	for _, dir := range []string{testOutputDir, testDownloadDir, filepath.Dir(localTestVideoPath)} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Não foi possível criar diretório de teste %s: %v", dir, err)
		}
	}
}

func ensureTestVideoExists(t *testing.T) string {
	// Verificar se o arquivo de teste local existe
	if _, err := os.Stat(localTestVideoPath); os.IsNotExist(err) {
		// Se não existir, baixar de uma URL pública
		if err := downloadTestVideo(localTestVideoPath); err != nil {
			t.Fatalf("Não foi possível baixar o vídeo de teste: %v", err)
		}
	}
	return localTestVideoPath
}

func downloadTestVideo(outputPath string) error {
	// Criar diretório se não existir
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}

	// Baixar o arquivo
	resp, err := http.Get(testVideoURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Criar arquivo de saída
	out, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Copiar conteúdo
	_, err = io.Copy(out, resp.Body)
	return err
}

func countTSFiles(dir string) int {
	files, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}

	count := 0
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".ts" {
			count++
		}
	}
	return count
}
