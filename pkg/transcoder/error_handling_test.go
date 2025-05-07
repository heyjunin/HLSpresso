package transcoder

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/heyjunin/HLSpresso/pkg/errors"
	"github.com/heyjunin/HLSpresso/pkg/hls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNetworkError verifica se erros de rede são tratados corretamente
func TestNetworkError(t *testing.T) {
	mockReporter := &mockProgressReporter{}
	tempDir := t.TempDir()
	
	// Servidor mock que vai responder com erro
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	
	// URL de um servidor que não existe
	nonExistentURL := "http://non.existent.server.example.com/video.mp4"
	
	tests := []struct {
		name          string
		url           string
		streamFromURL bool
		expectedError errors.ErrorType
		expectedCode  int
	}{
		{
			name:          "Server error response streaming",
			url:           server.URL,
			streamFromURL: true,
			expectedError: errors.NetworkError,
			expectedCode:  errors.ErrNetworkServerUnavailable,
		},
		{
			name:          "Non-existent server streaming",
			url:           nonExistentURL,
			streamFromURL: true,
			expectedError: errors.NetworkError,
			expectedCode:  errors.ErrNetworkDNSFailure,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := Options{
				InputPath:     tt.url,
				OutputPath:    filepath.Join(tempDir, "output"),
				IsRemoteInput: true,
				StreamFromURL: tt.streamFromURL,
			}
			
			trans, err := New(opts, mockReporter)
			require.NoError(t, err)
			
			// Tentativa de transcodificação que deve falhar
			_, err = trans.Transcode(context.Background())
			
			// Verificar tipo do erro
			assert.Error(t, err)
			structErr, ok := err.(*errors.StructuredError)
			assert.True(t, ok, "O erro deveria ser um StructuredError")
			
			// Verificar especificamente o tipo de erro e código
			if ok {
				assert.Equal(t, tt.expectedError, structErr.Type)
				assert.Equal(t, tt.expectedCode, structErr.Code)
			}
		})
	}
}

// TestFileNotFoundError verifica se erros de arquivo não encontrado são tratados corretamente
func TestFileNotFoundError(t *testing.T) {
	mockReporter := &mockProgressReporter{}
	tempDir := t.TempDir()
	nonExistentFile := filepath.Join(tempDir, "nonexistent.mp4")
	
	opts := Options{
		InputPath:     nonExistentFile,
		OutputPath:    filepath.Join(tempDir, "output"),
		IsRemoteInput: false,
	}
	
	trans, err := New(opts, mockReporter)
	require.NoError(t, err)
	
	// Tentativa de transcodificação que deve falhar
	_, err = trans.Transcode(context.Background())
	
	// Verificar tipo do erro
	assert.Error(t, err)
	structErr, ok := err.(*errors.StructuredError)
	assert.True(t, ok, "O erro deveria ser um StructuredError")
	
	// Verificar especificamente o tipo de erro e código
	if ok {
		assert.Equal(t, errors.FileNotFoundError, structErr.Type)
		assert.Equal(t, errors.ErrFileNotFound, structErr.Code)
	}
}

// TestInvalidFileFormatError verifica se erros de formato de arquivo são tratados corretamente
func TestInvalidFileFormatError(t *testing.T) {
	mockReporter := &mockProgressReporter{}
	tempDir := t.TempDir()
	
	// Criar um arquivo de texto simples (não um vídeo)
	textFile := filepath.Join(tempDir, "text_file.txt")
	err := os.WriteFile(textFile, []byte("This is not a video file"), 0644)
	require.NoError(t, err)
	
	// Criar um diretório (não um arquivo)
	dirPath := filepath.Join(tempDir, "directory")
	err = os.Mkdir(dirPath, 0755)
	require.NoError(t, err)
	
	tests := []struct {
		name          string
		inputPath     string
		expectedError errors.ErrorType
		expectedCode  int
	}{
		{
			name:          "Arquivo de texto (formato inválido)",
			inputPath:     textFile,
			expectedError: errors.InvalidFileFormatError,
			expectedCode:  errors.ErrUnsupportedFileFormat,
		},
		{
			name:          "Diretório como entrada",
			inputPath:     dirPath,
			expectedError: errors.InvalidFileFormatError,
			expectedCode:  errors.ErrInvalidFileFormat,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := Options{
				InputPath:     tt.inputPath,
				OutputPath:    filepath.Join(tempDir, "output"),
				IsRemoteInput: false,
			}
			
			trans, err := New(opts, mockReporter)
			require.NoError(t, err)
			
			// Tentativa de transcodificação que deve falhar
			_, err = trans.Transcode(context.Background())
			
			// Verificar tipo do erro
			assert.Error(t, err)
			structErr, ok := err.(*errors.StructuredError)
			assert.True(t, ok, "O erro deveria ser um StructuredError")
			
			// Verificar especificamente o tipo de erro e código
			if ok {
				assert.Equal(t, tt.expectedError, structErr.Type)
				assert.Equal(t, tt.expectedCode, structErr.Code)
			}
		})
	}
}

// TestUnsupportedResolutionError verifica se erros de resolução não suportada são tratados corretamente
func TestUnsupportedResolutionError(t *testing.T) {
	mockReporter := &mockProgressReporter{}
	tempDir := t.TempDir()
	
	// Criar um arquivo de vídeo dummy
	dummyVideoFile := filepath.Join(tempDir, "dummy.mp4")
	err := os.WriteFile(dummyVideoFile, []byte("dummy video content"), 0644)
	require.NoError(t, err)
	
	tests := []struct {
		name          string
		resolutions   []hls.VideoResolution
		expectedError errors.ErrorType
		expectedCode  int
	}{
		{
			name: "Resolução inválida (0x0)",
			resolutions: []hls.VideoResolution{
				{Width: 0, Height: 0, VideoBitrate: "500k", AudioBitrate: "64k"},
			},
			expectedError: errors.UnsupportedResolutionError,
			expectedCode:  errors.ErrInvalidResolution,
		},
		{
			name: "Resolução muito alta (10000x10000)",
			resolutions: []hls.VideoResolution{
				{Width: 10000, Height: 10000, VideoBitrate: "50000k", AudioBitrate: "192k"},
			},
			expectedError: errors.UnsupportedResolutionError,
			expectedCode:  errors.ErrResolutionTooHigh,
		},
		{
			name: "Resolução muito baixa (10x10)",
			resolutions: []hls.VideoResolution{
				{Width: 10, Height: 10, VideoBitrate: "50k", AudioBitrate: "32k"},
			},
			expectedError: errors.UnsupportedResolutionError,
			expectedCode:  errors.ErrResolutionTooLow,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := Options{
				InputPath:       dummyVideoFile,
				OutputPath:      filepath.Join(tempDir, "output"),
				OutputType:      HLSOutput,
				HLSResolutions:  tt.resolutions,
				IsRemoteInput:   false,
				AllowOverwrite:  true,
			}
			
			trans, err := New(opts, mockReporter)
			require.NoError(t, err)
			
			// Tentativa de transcodificação que deve falhar
			_, err = trans.Transcode(context.Background())
			
			// Verificar tipo do erro
			assert.Error(t, err)
			structErr, ok := err.(*errors.StructuredError)
			assert.True(t, ok, "O erro deveria ser um StructuredError")
			
			// Verificar especificamente o tipo de erro e código
			if ok {
				assert.Equal(t, tt.expectedError, structErr.Type, fmt.Sprintf("Erro: %v", structErr))
				assert.Equal(t, tt.expectedCode, structErr.Code)
			}
		})
	}
}

// TestPermissionError verifica se erros de permissão são tratados corretamente
func TestPermissionError(t *testing.T) {
	t.Skip("Teste de permissão depende de privilégios do sistema e não pode ser facilmente simulado")
	// Nota: Este teste é marcado como skip porque testar permissões 
	// de forma confiável é difícil em ambientes de CI/CD
}

// TestInvalidOutputPathError verifica se erros de caminho de saída inválido são tratados corretamente
func TestInvalidOutputPathError(t *testing.T) {
	mockReporter := &mockProgressReporter{}
	tempDir := t.TempDir()
	
	// Criar um arquivo de vídeo dummy
	dummyVideoFile := filepath.Join(tempDir, "dummy.mp4")
	err := os.WriteFile(dummyVideoFile, []byte("dummy video content"), 0644)
	require.NoError(t, err)
	
	// Criar um arquivo para usar como diretório de saída (o que causará erro)
	invalidOutputPath := filepath.Join(tempDir, "output_file")
	err = os.WriteFile(invalidOutputPath, []byte("this is not a directory"), 0644)
	require.NoError(t, err)
	
	opts := Options{
		InputPath:     dummyVideoFile,
		OutputPath:    invalidOutputPath, // Usando arquivo como diretório de saída para HLS
		OutputType:    HLSOutput,         // HLS requer um diretório
		IsRemoteInput: false,
	}
	
	trans, err := New(opts, mockReporter)
	require.NoError(t, err)
	
	// Tentativa de transcodificação que deve falhar
	_, err = trans.Transcode(context.Background())
	
	// Verificar tipo do erro
	assert.Error(t, err)
	
	// O erro pode ser SystemError ou InvalidOutputPathError dependendo 
	// de como o sistema operacional reporta a tentativa de criar um 
	// diretório onde já existe um arquivo
	structErr, ok := err.(*errors.StructuredError)
	assert.True(t, ok, "O erro deveria ser um StructuredError")
	
	// Verificar que é um dos erros esperados
	if ok {
		possibleErrors := []errors.ErrorType{errors.SystemError, errors.InvalidOutputPathError}
		found := false
		for _, errType := range possibleErrors {
			if structErr.Type == errType {
				found = true
				break
			}
		}
		assert.True(t, found, "O erro deveria ser um dos tipos esperados")
	}
}

// Observação: Os testes para os erros restantes como DiskSpaceError, MemoryError e 
// CodecNotFoundError são mais difíceis de testar em um ambiente automatizado, pois 
// requerem manipulação do sistema operacional ou dependências externas. 