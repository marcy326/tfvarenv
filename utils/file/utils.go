package file

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type utils struct{}

func NewUtils() Utils {
	return &utils{}
}

func (u *utils) CalculateHash(path string, opts *HashOptions) (string, error) {
	if opts == nil {
		opts = &HashOptions{
			Algorithm:  "sha256",
			BufferSize: 32 * 1024, // 32KB buffer
		}
	}

	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	buf := make([]byte, opts.BufferSize)

	for {
		n, err := file.Read(buf)
		if n > 0 {
			hash.Write(buf[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("failed to read file: %w", err)
		}
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (u *utils) CopyFile(src, dst string, opts *Options) error {
	if opts == nil {
		opts = &Options{
			CreateDirs:   true,
			Overwrite:    false,
			PreserveMode: true,
		}
	}

	// Check if destination exists
	if !opts.Overwrite {
		if _, err := os.Stat(dst); err == nil {
			return fmt.Errorf("destination file already exists: %s", dst)
		}
	}

	// Create destination directory if needed
	if opts.CreateDirs {
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Open source file
	source, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer source.Close()

	// Create destination file
	destination, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destination.Close()

	// Copy the contents
	if _, err := io.Copy(destination, source); err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	// Preserve file mode if requested
	if opts.PreserveMode {
		sourceInfo, err := os.Stat(src)
		if err != nil {
			return fmt.Errorf("failed to get source file info: %w", err)
		}
		if err := os.Chmod(dst, sourceInfo.Mode()); err != nil {
			return fmt.Errorf("failed to set file mode: %w", err)
		}
	}

	return nil
}

func (u *utils) CreateBackup(sourcePath string, opts *BackupOptions) (string, error) {
	if opts == nil {
		opts = &BackupOptions{
			BasePath:   ".backups",
			TimeFormat: "20060102150405",
			MaxBackups: 10,
		}
	}

	// Create backup directory
	backupDir := filepath.Join(opts.BasePath, filepath.Base(filepath.Dir(sourcePath)))
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Generate backup file name
	timestamp := time.Now().Format(opts.TimeFormat)
	backupPath := filepath.Join(backupDir,
		fmt.Sprintf("%s.%s.bak", filepath.Base(sourcePath), timestamp))

	// Copy file to backup location
	if err := u.CopyFile(sourcePath, backupPath, &Options{
		CreateDirs:   true,
		Overwrite:    false,
		PreserveMode: true,
	}); err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}

	// Cleanup old backups if needed
	if opts.MaxBackups > 0 {
		if err := u.cleanupOldBackups(backupDir, opts.MaxBackups); err != nil {
			// Log warning but don't fail the backup operation
			fmt.Printf("Warning: failed to cleanup old backups: %v\n", err)
		}
	}

	return backupPath, nil
}

func (u *utils) EnsureDirectory(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	return nil
}

func (u *utils) GetFileInfo(path string) (*FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	hash, err := u.CalculateHash(path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate file hash: %w", err)
	}

	return &FileInfo{
		Path:       path,
		Hash:       hash,
		Size:       info.Size(),
		CreatedAt:  info.ModTime().Unix(),
		ModifiedAt: info.ModTime().Unix(),
		Mode:       uint32(info.Mode()),
	}, nil
}

func (u *utils) WriteFile(path string, content []byte, opts *Options) error {
	if opts == nil {
		opts = &Options{
			CreateDirs: true,
			Overwrite:  false,
		}
	}

	if !opts.Overwrite {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("file already exists: %s", path)
		}
	}

	if opts.CreateDirs {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	if err := os.WriteFile(path, content, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func (u *utils) cleanupOldBackups(backupDir string, maxBackups int) error {
	files, err := filepath.Glob(filepath.Join(backupDir, "*.bak"))
	if err != nil {
		return fmt.Errorf("failed to list backup files: %w", err)
	}

	if len(files) <= maxBackups {
		return nil
	}

	// Sort files by modification time (oldest first)
	sort.Slice(files, func(i, j int) bool {
		iInfo, _ := os.Stat(files[i])
		jInfo, _ := os.Stat(files[j])
		return iInfo.ModTime().Before(jInfo.ModTime())
	})

	// Remove oldest files
	for i := 0; i < len(files)-maxBackups; i++ {
		if err := os.Remove(files[i]); err != nil {
			return fmt.Errorf("failed to remove old backup: %w", err)
		}
	}

	return nil
}

func (u *utils) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (u *utils) FileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
