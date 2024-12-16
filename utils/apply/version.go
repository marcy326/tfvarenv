package apply

import (
	"context"
	"fmt"
	"os"
	"time"

	"tfvarenv/utils/aws"
	"tfvarenv/utils/version"
)

// VersionInfo contains version-related information
type VersionInfo struct {
	Version    *version.Version
	IsNew      bool
	SourceFile string
}

func (m *Manager) getVersionInfo(ctx context.Context, opts *Options) (*VersionInfo, error) {
	versionManager := version.NewManager(m.awsClient, m.fileUtils, opts.Environment)

	if opts.Remote {
		return m.getRemoteVersion(ctx, versionManager, opts)
	}
	return m.getLocalVersion(ctx, versionManager, opts)
}

func (m *Manager) getRemoteVersion(ctx context.Context, versionManager version.Manager, opts *Options) (*VersionInfo, error) {
	var ver *version.Version
	var err error

	if opts.VersionID != "" {
		ver, err = versionManager.GetVersion(ctx, opts.VersionID)
	} else {
		ver, err = versionManager.GetLatestVersion(ctx)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get version information: %w", err)
	}

	return &VersionInfo{
		Version: ver,
		IsNew:   false,
	}, nil
}

func (m *Manager) getLocalVersion(ctx context.Context, versionManager version.Manager, opts *Options) (*VersionInfo, error) {
	varFile := opts.VarFile
	if varFile == "" {
		varFile = opts.Environment.Local.TFVarsPath
	}

	// Get file information
	fileInfo, err := m.fileUtils.GetFileInfo(varFile)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Check for existing version
	latestVer, _ := versionManager.GetLatestVersion(ctx)
	if latestVer != nil && latestVer.Hash == fileInfo.Hash {
		return &VersionInfo{
			Version:    latestVer,
			IsNew:      false,
			SourceFile: varFile,
		}, nil
	}

	// Create new version
	content, err := m.fileUtils.ReadFile(varFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read tfvars file: %w", err)
	}

	// Upload to S3
	uploadInput := &aws.UploadInput{
		Bucket:      opts.Environment.S3.Bucket,
		Key:         opts.Environment.GetS3Path(),
		Content:     content,
		Description: fmt.Sprintf("Uploaded during local apply from %s", varFile),
		Metadata: map[string]string{
			"Hash":       fileInfo.Hash,
			"UploadedBy": os.Getenv("USER"),
		},
	}

	uploadOutput, err := m.awsClient.UploadFile(ctx, uploadInput)
	if err != nil {
		return nil, fmt.Errorf("failed to upload tfvars: %w", err)
	}

	// Create version record
	newVersion := &version.Version{
		VersionID:   uploadOutput.VersionID,
		Hash:        fileInfo.Hash,
		Timestamp:   time.Now(),
		Description: uploadInput.Description,
		UploadedBy:  os.Getenv("USER"),
		Size:        fileInfo.Size,
	}

	if err := versionManager.AddVersion(ctx, newVersion); err != nil {
		return nil, fmt.Errorf("failed to record version: %w", err)
	}

	return &VersionInfo{
		Version:    newVersion,
		IsNew:      true,
		SourceFile: varFile,
	}, nil
}
