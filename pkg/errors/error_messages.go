package errors

// Mensagens de erro padronizadas para cada código
var ErrorMessages = map[int]string{
	// NetworkError
	ErrNetworkConnectionFailed: "Erro de rede ao tentar acessar o arquivo. Verifique sua conexão e tente novamente.",
	ErrNetworkTimeout:          "Tempo limite de rede excedido. Verifique sua conexão e tente novamente.",
	ErrNetworkDNSFailure:       "Falha na resolução DNS. Verifique o endereço do servidor e tente novamente.",
	ErrNetworkServerUnavailable: "Servidor indisponível. Tente novamente mais tarde.",
	
	// DiskSpaceError
	ErrDiskSpaceInsufficient:   "Espaço insuficiente no disco para processar o arquivo. Libere espaço e tente novamente.",
	ErrDiskQuotaExceeded:       "Cota de disco excedida. Libere espaço ou ajuste sua cota.",
	ErrDiskWriteFailed:         "Falha ao escrever no disco. Verifique as permissões e o espaço disponível.",
	
	// FileNotFoundError
	ErrFileNotFound:            "Arquivo não encontrado. Verifique o caminho e se o arquivo está acessível.",
	ErrFileNotAccessible:       "Arquivo inacessível. Verifique as permissões e se o arquivo existe.",
	ErrDirectoryNotFound:       "Diretório não encontrado. Verifique o caminho e se o diretório existe.",
	
	// InvalidFileFormatError
	ErrInvalidFileFormat:       "Formato de arquivo inválido. Somente MP4, MOV, AVI, MKV, WEBM são suportados.",
	ErrUnsupportedFileFormat:   "Formato de arquivo não suportado. Utilize um dos formatos compatíveis.",
	ErrCorruptedFile:           "O arquivo parece estar corrompido. Verifique a integridade do arquivo.",
	
	// PermissionError
	ErrPermissionDenied:        "Permissão negada. Verifique as permissões de leitura/gravação no arquivo ou diretório.",
	ErrReadPermissionDenied:    "Permissão de leitura negada. Verifique as permissões do arquivo.",
	ErrWritePermissionDenied:   "Permissão de escrita negada. Verifique as permissões do diretório de destino.",
	
	// MemoryError
	ErrOutOfMemory:             "Memória insuficiente para processar o arquivo. Tente reduzir a resolução ou o tamanho do arquivo.",
	ErrMemoryAllocationFailed:  "Falha na alocação de memória. Tente fechar outros aplicativos ou processos.",
	
	// CodecNotFoundError
	ErrCodecNotFound:           "Codec necessário não encontrado. Certifique-se de que o codec necessário está instalado.",
	ErrCodecNotSupported:       "Codec não suportado nesta plataforma.",
	ErrMissingDependency:       "Dependência necessária não encontrada. Verifique a instalação do FFmpeg.",
	
	// InvalidOutputPathError
	ErrInvalidOutputPath:       "Caminho de saída inválido ou inacessível. Verifique as permissões e tente novamente.",
	ErrOutputPathNotAccessible: "Caminho de saída inacessível. Verifique as permissões e se o diretório existe.",
	ErrOutputDirectoryCreationFailed: "Falha ao criar diretório de saída. Verifique as permissões.",
	
	// UnsupportedResolutionError
	ErrUnsupportedResolution:   "Resolução de vídeo não suportada. Tente uma resolução compatível.",
	ErrInvalidResolution:       "Resolução de vídeo inválida. Use uma resolução válida.",
	ErrResolutionTooHigh:       "Resolução de vídeo muito alta. Use uma resolução menor.",
	ErrResolutionTooLow:        "Resolução de vídeo muito baixa. Use uma resolução maior.",
}

// GetErrorMessage retorna a mensagem de erro padronizada para um código de erro
func GetErrorMessage(code int) string {
	if msg, ok := ErrorMessages[code]; ok {
		return msg
	}
	return "Erro desconhecido."
}

