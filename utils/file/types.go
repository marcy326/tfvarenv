package file

// Utils defines file operation interfaces
type Utils interface {
	CalculateHash(path string, opts *HashOptions) (string, error)
	CopyFile(src, dst string, opts *Options) error
	CreateBackup(sourcePath string, opts *BackupOptions) (string, error)
	EnsureDirectory(path string) error
	GetFileInfo(path string) (*FileInfo, error)
	WriteFile(path string, content []byte, opts *Options) error
	ReadFile(path string) ([]byte, error)
	FileExists(path string) (bool, error)
}

// Options represents common file operation options
type Options struct {
	CreateDirs   bool
	Overwrite    bool
	PreserveMode bool
	BufferSize   int
}

// BackupOptions represents options for backup operations
type BackupOptions struct {
	BasePath   string
	TimeFormat string
	MaxBackups int
	Compress   bool
}

// FileInfo represents metadata about a file
type FileInfo struct {
	Path       string
	Hash       string
	Size       int64
	CreatedAt  int64
	ModifiedAt int64
	Mode       uint32
}

// HashOptions represents options for hash calculation
type HashOptions struct {
	Algorithm  string
	BufferSize int
}

// CopyProgress represents progress of a copy operation
type CopyProgress struct {
	TotalBytes       int64
	TransferredBytes int64
	Percentage       float64
}
