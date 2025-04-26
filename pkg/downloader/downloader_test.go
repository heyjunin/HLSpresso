package downloader

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/heyjunin/HLSpresso/pkg/progress"
)

// mockProgressReporter é um mock simples para testes
type mockProgressReporter struct {
	started   bool
	completed bool
	updates   int
	total     int64
	current   int64
}

func (m *mockProgressReporter) Start(total int64)                 { m.started = true; m.total = total }
func (m *mockProgressReporter) Update(current int64, _, _ string) { m.updates++; m.current = current }
func (m *mockProgressReporter) Increment(_, _ string)             { m.updates++; m.current++ }
func (m *mockProgressReporter) Complete()                         { m.completed = true }
func (m *mockProgressReporter) Updates() <-chan progress.ProgressEvent {
	ch := make(chan progress.ProgressEvent)
	close(ch)
	return ch
}
func (m *mockProgressReporter) Close()                {}
func (m *mockProgressReporter) JSON() (string, error) { return "{}", nil } // Mock JSON

func TestNewDownloader(t *testing.T) {
	opts := Options{}
	d := New(opts)
	if d == nil {
		t.Fatal("New() returned nil")
	}
	if d.client.Timeout != 30*time.Minute { // Check default timeout
		t.Errorf("Expected default timeout 30m, got %v", d.client.Timeout)
	}

	optsWithTimeout := Options{Timeout: 5 * time.Minute}
	dWithTimeout := New(optsWithTimeout)
	if dWithTimeout.client.Timeout != 5*time.Minute {
		t.Errorf("Expected timeout 5m, got %v", dWithTimeout.client.Timeout)
	}
}

func TestDownloader_Download_Success(t *testing.T) {
	// Criar servidor HTTP de teste
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "12") // Importante para o progresso
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "test content")
	}))
	defer server.Close()

	// Criar diretório de teste
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "downloaded_file.txt")

	// Configurar downloader
	mockReporter := &mockProgressReporter{}
	opts := Options{
		URL:           server.URL,
		OutputPath:    outputPath,
		Progress:      mockReporter,
		AllowOverride: true,
	}
	d := New(opts)

	// Executar download
	downloadedPath, err := d.Download(context.Background())
	if err != nil {
		t.Fatalf("Download() failed: %v", err)
	}

	// Verificar caminho retornado
	if downloadedPath != outputPath {
		t.Errorf("Download() returned path %q, want %q", downloadedPath, outputPath)
	}

	// Verificar se o arquivo foi criado
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}
	if string(content) != "test content" {
		t.Errorf("Downloaded content = %q, want %q", string(content), "test content")
	}

	// Verificar progresso (simples)
	if !mockReporter.started {
		t.Error("Progress reporter Start() was not called")
	}
	if !mockReporter.completed {
		t.Error("Progress reporter Complete() was not called")
	}
	if mockReporter.updates == 0 {
		t.Error("Progress reporter Update() was never called")
	}
}

func TestDownloader_Download_SkipExisting(t *testing.T) {
	// Criar servidor (não deve ser chamado)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("Server should not be called when file exists and override is false")
	}))
	defer server.Close()

	// Criar arquivo existente
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "existing_file.txt")
	if err := os.WriteFile(outputPath, []byte("existing data"), 0644); err != nil {
		t.Fatalf("Failed to create dummy existing file: %v", err)
	}

	// Configurar downloader (AllowOverride = false)
	opts := Options{
		URL:           server.URL,
		OutputPath:    outputPath,
		AllowOverride: false,
	}
	d := New(opts)

	// Executar download
	downloadedPath, err := d.Download(context.Background())
	if err != nil {
		t.Fatalf("Download() failed unexpectedly: %v", err)
	}

	// Verificar caminho retornado
	if downloadedPath != outputPath {
		t.Errorf("Download() returned path %q, want %q", downloadedPath, outputPath)
	}

	// Verificar que o conteúdo não mudou
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(content) != "existing data" {
		t.Errorf("File content was modified, expected %q, got %q", "existing data", string(content))
	}
}

func TestDownloader_Download_OverwriteExisting(t *testing.T) {
	// Criar servidor
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "new content")
	}))
	defer server.Close()

	// Criar arquivo existente
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "overwrite_me.txt")
	if err := os.WriteFile(outputPath, []byte("old data"), 0644); err != nil {
		t.Fatalf("Failed to create dummy existing file: %v", err)
	}

	// Configurar downloader (AllowOverride = true)
	opts := Options{
		URL:           server.URL,
		OutputPath:    outputPath,
		AllowOverride: true,
	}
	d := New(opts)

	// Executar download
	_, err := d.Download(context.Background())
	if err != nil {
		t.Fatalf("Download() failed unexpectedly: %v", err)
	}

	// Verificar que o conteúdo FOI modificado
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(content) != "new content" {
		t.Errorf("File content was not overwritten, expected %q, got %q", "new content", string(content))
	}
}

func TestDownloader_Download_HTTPError(t *testing.T) {
	// Criar servidor que retorna erro
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound) // 404 Not Found
	}))
	defer server.Close()

	// Configurar downloader
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "not_found.txt")
	opts := Options{URL: server.URL, OutputPath: outputPath}
	d := New(opts)

	// Executar download - deve falhar
	_, err := d.Download(context.Background())
	if err == nil {
		t.Fatal("Download() should have failed for HTTP 404, but got nil error")
	}

	// Verificar tipo de erro (opcional, mas bom)
	// if structuredErr, ok := err.(*errors.StructuredError); ok {
	// 	if structuredErr.Type != errors.DownloadError {
	// 		t.Errorf("Expected error type %q, got %q", errors.DownloadError, structuredErr.Type)
	// 	}
	// } else {
	// 	t.Errorf("Expected a StructuredError, got %T", err)
	// }
}

func TestDownloader_Download_ContextCancel(t *testing.T) {
	// Criar servidor que espera antes de responder
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond) // Simula um download lento
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "content")
	}))
	defer server.Close()

	// Configurar downloader
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "cancelled.txt")
	opts := Options{URL: server.URL, OutputPath: outputPath}
	d := New(opts)

	// Criar contexto com cancelamento rápido
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond) // Timeout < sleep do servidor
	defer cancel()

	// Executar download - deve falhar devido ao cancelamento
	_, err := d.Download(ctx)
	if err == nil {
		t.Fatal("Download() should have failed due to context cancellation, but got nil error")
	}

	// Verificar se o erro é de contexto cancelado ou deadline excedido
	if err != context.Canceled && err != context.DeadlineExceeded {
		// Pode haver um erro de rede antes do cancelamento ser detectado, verificar o erro encapsulado
		if nestedErr := context.Cause(ctx); nestedErr != context.DeadlineExceeded && nestedErr != context.Canceled {
			t.Errorf("Expected context.Canceled or context.DeadlineExceeded, got: %v (context cause: %v)", err, nestedErr)
		}
	}
}
