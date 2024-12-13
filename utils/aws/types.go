package aws

import (
	"context"
	"time"
)

// Client defines the interface for AWS operations
type Client interface {
	GetAccountID(ctx context.Context) (string, error)
	CheckBucketVersioning(ctx context.Context, bucket string) error
	UploadFile(ctx context.Context, input *UploadInput) (*UploadOutput, error)
	DownloadFile(ctx context.Context, input *DownloadInput) (*DownloadOutput, error)
	ListVersions(ctx context.Context, input *ListVersionsInput) (*ListVersionsOutput, error)
}

// UploadInput represents input parameters for file upload
type UploadInput struct {
	Bucket      string
	Key         string
	Content     []byte
	ContentType string
	Description string
	Metadata    map[string]string
}

// UploadOutput represents the result of a file upload
type UploadOutput struct {
	VersionID string
	ETag      string
}

// DownloadInput represents input parameters for file download
type DownloadInput struct {
	Bucket    string
	Key       string
	VersionID string
}

// DownloadOutput represents the result of a file download
type DownloadOutput struct {
	Content     []byte
	VersionID   string
	Metadata    map[string]string
	ContentType string
}

// ListVersionsInput represents input parameters for listing versions
type ListVersionsInput struct {
	Bucket     string
	Key        string
	MaxKeys    int32
	StartAfter string
}

// ListVersionsOutput represents the result of listing versions
type ListVersionsOutput struct {
	Versions    []VersionInfo
	IsTruncated bool
	NextMarker  string
}

// VersionInfo represents metadata about a specific version
type VersionInfo struct {
	VersionID   string
	Hash        string
	Timestamp   time.Time
	Description string
	Size        int64
	IsLatest    bool
	Metadata    map[string]string
}
