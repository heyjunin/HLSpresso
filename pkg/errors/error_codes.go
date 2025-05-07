package errors

// Códigos de erro para diferentes componentes
const (
	// Códigos de erro para NetworkError (1000-1099)
	ErrNetworkConnectionFailed = 1000
	ErrNetworkTimeout         = 1001
	ErrNetworkDNSFailure      = 1002
	ErrNetworkServerUnavailable = 1003
	
	// Códigos de erro para DiskSpaceError (1100-1199)
	ErrDiskSpaceInsufficient  = 1100
	ErrDiskQuotaExceeded     = 1101
	ErrDiskWriteFailed       = 1102

	// Códigos de erro para FileNotFoundError (1200-1299)
	ErrFileNotFound          = 1200
	ErrFileNotAccessible     = 1201
	ErrDirectoryNotFound     = 1202

	// Códigos de erro para InvalidFileFormatError (1300-1399)
	ErrInvalidFileFormat     = 1300
	ErrUnsupportedFileFormat = 1301
	ErrCorruptedFile         = 1302

	// Códigos de erro para PermissionError (1400-1499)
	ErrPermissionDenied      = 1400
	ErrReadPermissionDenied  = 1401
	ErrWritePermissionDenied = 1402

	// Códigos de erro para MemoryError (1500-1599)
	ErrOutOfMemory           = 1500
	ErrMemoryAllocationFailed = 1501

	// Códigos de erro para CodecNotFoundError (1600-1699)
	ErrCodecNotFound         = 1600
	ErrCodecNotSupported     = 1601
	ErrMissingDependency     = 1602

	// Códigos de erro para InvalidOutputPathError (1700-1799)
	ErrInvalidOutputPath     = 1700
	ErrOutputPathNotAccessible = 1701
	ErrOutputDirectoryCreationFailed = 1702

	// Códigos de erro para UnsupportedResolutionError (1800-1899)
	ErrUnsupportedResolution = 1800
	ErrInvalidResolution     = 1801
	ErrResolutionTooHigh     = 1802
	ErrResolutionTooLow      = 1803
)
