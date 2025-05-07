package errors

// NetworkError indica problemas de rede durante a transcodificação
const NetworkError ErrorType = "network_error"

// DiskSpaceError indica falta de espaço em disco
const DiskSpaceError ErrorType = "disk_space_error"

// FileNotFoundError indica que um arquivo não foi encontrado
const FileNotFoundError ErrorType = "file_not_found_error"

// InvalidFileFormatError indica formato de arquivo inválido ou não suportado
const InvalidFileFormatError ErrorType = "invalid_file_format_error"

// PermissionError indica problemas de permissão de acesso
const PermissionError ErrorType = "permission_error"

// MemoryError indica problemas de memória durante o processamento
const MemoryError ErrorType = "memory_error"

// CodecNotFoundError indica falta de codecs necessários
const CodecNotFoundError ErrorType = "codec_not_found_error"

// InvalidOutputPathError indica caminho de saída inválido
const InvalidOutputPathError ErrorType = "invalid_output_path_error"

// UnsupportedResolutionError indica resolução de vídeo não suportada
const UnsupportedResolutionError ErrorType = "unsupported_resolution_error"
