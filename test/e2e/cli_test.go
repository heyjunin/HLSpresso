package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	// Caminho para o binário compilado
	binaryPath = "../../HLSpresso"
)

func TestCLILocalVideoToHLS(t *testing.T) {
	// Verifique se o binário existe
	if !binaryExists() {
		t.Skip("Binário não encontrado em " + binaryPath + ", pulando teste")
		return
	}

	// Verifique se o FFmpeg está instalado
	if !checkFFmpegInstalled() {
		t.Skip("FFmpeg não encontrado, pulando teste")
		return
	}

	// Prepare o vídeo de teste
	setupTestDirectories(t)
	inputPath := ensureTestVideoExists(t)

	// Caminho de saída
	outputPath := filepath.Join(testOutputDir, "cli_hls")

	// Comando para executar
	args := []string{
		"-i", inputPath,
		"-o", outputPath,
		"-t", "hls",
		"--hls-segment-duration", "2",
	}

	// Executar o comando com timeout
	cmd := exec.Command(binaryPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Executar com timeout
	if err := runWithTimeout(cmd, 3*time.Minute); err != nil {
		t.Fatalf("Erro ao executar comando CLI: %v", err)
	}

	// Verificar se o master playlist foi criado
	masterPlaylistPath := filepath.Join(outputPath, "master.m3u8")
	if _, err := os.Stat(masterPlaylistPath); os.IsNotExist(err) {
		t.Errorf("Master playlist não foi gerado: %s", masterPlaylistPath)
	}

	// Verificar se pelo menos um diretório de stream foi criado
	streamDir := filepath.Join(outputPath, "stream_0")
	if _, err := os.Stat(streamDir); os.IsNotExist(err) {
		t.Errorf("Diretório de stream não foi criado: %s", streamDir)
	}
}

func TestCLIRemoteVideoToMP4(t *testing.T) {
	// Verifique se o binário existe
	if !binaryExists() {
		t.Skip("Binário não encontrado em " + binaryPath + ", pulando teste")
		return
	}

	// Verifique se o FFmpeg está instalado
	if !checkFFmpegInstalled() {
		t.Skip("FFmpeg não encontrado, pulando teste")
		return
	}

	// Prepare os diretórios de teste
	setupTestDirectories(t)

	// Caminho de saída
	outputPath := filepath.Join(testOutputDir, "cli_output.mp4")

	// Comando para executar
	args := []string{
		"-i", testVideoURL,
		"-o", outputPath,
		"-t", "mp4",
		"--remote",
		"--download-dir", testDownloadDir,
	}

	// Executar o comando com timeout
	cmd := exec.Command(binaryPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Executar com timeout
	if err := runWithTimeout(cmd, 5*time.Minute); err != nil {
		t.Fatalf("Erro ao executar comando CLI: %v", err)
	}

	// Verificar se o arquivo foi criado
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Errorf("Arquivo MP4 não foi gerado: %s", outputPath)
	}
}

func TestCLIHelp(t *testing.T) {
	// Verifique se o binário existe
	if !binaryExists() {
		t.Skip("Binário não encontrado em " + binaryPath + ", pulando teste")
		return
	}

	// Executar comando de ajuda
	cmd := exec.Command(binaryPath, "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Erro ao executar comando de ajuda: %v", err)
	}

	// Verificar se a saída contém informações básicas
	outputStr := string(output)
	for _, expected := range []string{"HLSpresso", "input", "output", "hls"} {
		if !strings.Contains(strings.ToLower(outputStr), strings.ToLower(expected)) {
			t.Errorf("Saída do comando de ajuda não contém a palavra '%s'", expected)
		}
	}
}

// Funções auxiliares

func binaryExists() bool {
	_, err := os.Stat(binaryPath)
	return err == nil
}

func runWithTimeout(cmd *exec.Cmd, timeout time.Duration) error {
	// Iniciar o comando
	if err := cmd.Start(); err != nil {
		return err
	}

	// Criar canal para sinal de conclusão
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	// Esperar pela conclusão ou timeout
	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		// Tentar finalizar gentilmente o processo primeiro
		if err := cmd.Process.Signal(os.Interrupt); err != nil {
			// Se não conseguir interromper, matar forçadamente
			cmd.Process.Kill()
		}
		return nil
	}
}
