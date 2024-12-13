package version

import (
	"time"
)

// Version represents a single version of a tfvars file
type Version struct {
	VersionID   string            `json:"version_id"`
	Hash        string            `json:"hash"`
	Timestamp   time.Time         `json:"timestamp"`
	Description string            `json:"description"`
	UploadedBy  string            `json:"uploaded_by"`
	Size        int64             `json:"size"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// VersionManagement represents the version management file structure
type VersionManagement struct {
	FormatVersion string    `json:"format_version"`
	LastUpdated   time.Time `json:"last_updated"`
	Environment   struct {
		Name   string `json:"name"`
		S3Path string `json:"s3_path"`
	} `json:"environment"`
	Versions        []Version `json:"versions"`
	LatestVersionID string    `json:"latest_version_id"`
}

// QueryOptions represents options for querying versions
type QueryOptions struct {
	Since      time.Time
	Before     time.Time
	Limit      int
	SortByDate bool
	LatestOnly bool
	SearchText string
}

// VersionStats represents statistics about versions
type VersionStats struct {
	TotalVersions    int
	AverageSize      int64
	MostActiveUser   string
	LastUpdated      time.Time
	VersionsByUser   map[string]int
	SizeDistribution map[string]int // size ranges
}
