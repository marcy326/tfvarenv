package version

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"tfvarenv/config"
	"tfvarenv/utils/aws"
	"tfvarenv/utils/file"
)

const ManagementFormatVersion = "1.0"

// Manager provides version management functionality
type Manager interface {
	AddVersion(ctx context.Context, version *Version) error
	GetVersions(ctx context.Context, opts *QueryOptions) ([]Version, error)
	GetLatestVersion(ctx context.Context) (*Version, error)
	GetVersion(ctx context.Context, versionID string) (*Version, error)
	GetStats(ctx context.Context) (*VersionStats, error)
	CompareVersions(ctx context.Context, v1, v2 string) ([]string, error)
}

type manager struct {
	awsClient aws.Client
	fileUtils file.Utils
	env       *config.Environment
}

// NewManager creates a new version manager
func NewManager(awsClient aws.Client, fileUtils file.Utils, env *config.Environment) Manager {
	return &manager{
		awsClient: awsClient,
		fileUtils: fileUtils,
		env:       env,
	}
}

func (m *manager) AddVersion(ctx context.Context, version *Version) error {
	management, err := m.getVersionManagement(ctx)
	if err != nil {
		// Initialize a new version management structure
		// Create an empty version management information for the environment
		management = &VersionManagement{
			FormatVersion: ManagementFormatVersion,
			LastUpdated:   time.Now(),
			Environment: struct {
				Name   string `json:"name"`
				S3Path string `json:"s3_path"`
			}{
				Name:   m.env.Name,
				S3Path: m.env.GetS3Path(),
			},
			Versions: make([]Version, 0),
		}
	}

	// Add new version
	management.Versions = append([]Version{*version}, management.Versions...)
	management.LatestVersionID = version.VersionID
	management.LastUpdated = time.Now()

	// Limit the number of stored versions to 100 to prevent excessive storage
	if len(management.Versions) > 100 {
		management.Versions = management.Versions[:100]
	}

	// Marshal the version management data into a JSON format with indentation for readability
	data, err := json.MarshalIndent(management, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal version management: %w", err)
	}

	uploadInput := &aws.UploadInput{
		Bucket:      m.env.S3.Bucket,
		Key:         m.env.GetVersionMetadataKey(),
		Content:     data,
		ContentType: "application/json",
	}

	if _, err := m.awsClient.UploadFile(ctx, uploadInput); err != nil {
		return fmt.Errorf("failed to save version management: %w", err)
	}

	return nil
}

func (m *manager) GetVersions(ctx context.Context, opts *QueryOptions) ([]Version, error) {
	management, err := m.getVersionManagement(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get version management: %w", err)
	}

	versions := m.filterVersions(management.Versions, opts)

	if opts != nil && opts.LatestOnly && len(versions) > 1 {
		versions = versions[:1]
	}

	return versions, nil
}

func (m *manager) GetLatestVersion(ctx context.Context) (*Version, error) {
	versions, err := m.GetVersions(ctx, &QueryOptions{
		LatestOnly: true,
	})
	if err != nil {
		return nil, err
	}

	if len(versions) == 0 {
		return nil, fmt.Errorf("no versions found")
	}

	return &versions[0], nil
}

func (m *manager) GetVersion(ctx context.Context, versionID string) (*Version, error) {
	management, err := m.getVersionManagement(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get version management: %w", err)
	}

	for _, v := range management.Versions {
		if v.VersionID == versionID {
			return &v, nil
		}
	}

	return nil, fmt.Errorf("version not found: %s", versionID)
}

func (m *manager) GetStats(ctx context.Context) (*VersionStats, error) {
	management, err := m.getVersionManagement(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get version management: %w", err)
	}

	stats := &VersionStats{
		TotalVersions:    len(management.Versions),
		VersionsByUser:   make(map[string]int),
		SizeDistribution: make(map[string]int),
	}

	var totalSize int64
	userCounts := make(map[string]int)

	for _, v := range management.Versions {
		totalSize += v.Size
		userCounts[v.UploadedBy]++

		// Calculate size distribution
		sizeRange := getSizeRange(v.Size)
		stats.SizeDistribution[sizeRange]++
	}

	if stats.TotalVersions > 0 {
		stats.AverageSize = totalSize / int64(stats.TotalVersions)
	}

	// Find most active user
	maxCount := 0
	for user, count := range userCounts {
		if count > maxCount {
			maxCount = count
			stats.MostActiveUser = user
		}
	}

	stats.LastUpdated = management.LastUpdated
	stats.VersionsByUser = userCounts

	return stats, nil
}

func (m *manager) CompareVersions(ctx context.Context, v1, v2 string) ([]string, error) {
	ver1, err := m.GetVersion(ctx, v1)
	if err != nil {
		return nil, err
	}

	ver2, err := m.GetVersion(ctx, v2)
	if err != nil {
		return nil, err
	}

	if ver1.Hash == ver2.Hash {
		return nil, nil
	}

	// Download and compare contents
	content1, err := m.downloadVersion(ctx, v1)
	if err != nil {
		return nil, err
	}

	content2, err := m.downloadVersion(ctx, v2)
	if err != nil {
		return nil, err
	}

	return compareContents(content1, content2), nil
}

func (m *manager) getVersionManagement(ctx context.Context) (*VersionManagement, error) {
	input := &aws.DownloadInput{
		Bucket: m.env.S3.Bucket,
		Key:    m.env.GetVersionMetadataKey(),
	}

	output, err := m.awsClient.DownloadFile(ctx, input)
	if err != nil {
		// Return new empty management if file doesn't exist
		return &VersionManagement{
			FormatVersion: ManagementFormatVersion,
			LastUpdated:   time.Now(),
			Environment: struct {
				Name   string `json:"name"`
				S3Path string `json:"s3_path"`
			}{
				Name:   m.env.Name,
				S3Path: m.env.GetS3Path(),
			},
			Versions: make([]Version, 0),
		}, nil
	}

	var management VersionManagement
	if err := json.Unmarshal(output.Content, &management); err != nil {
		return nil, fmt.Errorf("failed to decode version management: %w", err)
	}

	return &management, nil
}

func (m *manager) saveVersionManagement(ctx context.Context, management *VersionManagement) error {
	data, err := json.MarshalIndent(management, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal version management: %w", err)
	}

	input := &aws.UploadInput{
		Bucket:      m.env.S3.Bucket,
		Key:         m.env.GetVersionMetadataKey(),
		Content:     data,
		ContentType: "application/json",
	}

	_, err = m.awsClient.UploadFile(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to save version management: %w", err)
	}

	return nil
}

func (m *manager) filterVersions(versions []Version, opts *QueryOptions) []Version {
	if opts == nil {
		return versions
	}

	var filtered []Version
	for _, v := range versions {
		if !m.matchesFilter(v, opts) {
			continue
		}
		filtered = append(filtered, v)
	}

	if opts.SortByDate {
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].Timestamp.After(filtered[j].Timestamp)
		})
	}

	return filtered
}

func (m *manager) matchesFilter(v Version, opts *QueryOptions) bool {
	if !opts.Since.IsZero() && v.Timestamp.Before(opts.Since) {
		return false
	}
	if !opts.Before.IsZero() && v.Timestamp.After(opts.Before) {
		return false
	}
	if opts.SearchText != "" && !strings.Contains(
		strings.ToLower(v.Description),
		strings.ToLower(opts.SearchText)) {
		return false
	}
	return true
}

func (m *manager) downloadVersion(ctx context.Context, versionID string) ([]byte, error) {
	input := &aws.DownloadInput{
		Bucket:    m.env.S3.Bucket,
		Key:       m.env.GetS3Path(),
		VersionID: versionID,
	}

	output, err := m.awsClient.DownloadFile(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to download version: %w", err)
	}

	return output.Content, nil
}

func compareContents(content1, content2 []byte) []string {
	// Simple line-by-line comparison
	lines1 := strings.Split(string(content1), "\n")
	lines2 := strings.Split(string(content2), "\n")

	var diffs []string
	for i := 0; i < len(lines1) || i < len(lines2); i++ {
		var l1, l2 string
		if i < len(lines1) {
			l1 = lines1[i]
		}
		if i < len(lines2) {
			l2 = lines2[i]
		}
		if l1 != l2 {
			diffs = append(diffs, fmt.Sprintf("Line %d:\n  - %s\n  + %s", i+1, l1, l2))
		}
	}

	return diffs
}

func getSizeRange(size int64) string {
	switch {
	case size < 1024:
		return "< 1KB"
	case size < 10*1024:
		return "1KB-10KB"
	case size < 100*1024:
		return "10KB-100KB"
	case size < 1024*1024:
		return "100KB-1MB"
	default:
		return "> 1MB"
	}
}
