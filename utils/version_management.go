package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"tfvarenv/config"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Format version for version management file
const VersionManagementFormatVersion = "1.0"

// Version represents a single version of a tfvars file
type Version struct {
	VersionID   string    `json:"version_id"`
	Timestamp   time.Time `json:"timestamp"`
	Hash        string    `json:"hash"`
	Description string    `json:"description"`
	UploadedBy  string    `json:"uploaded_by"`
	Size        int64     `json:"size"`
}

// VersionManagement represents the version management file structure
type VersionManagement struct {
	FormatVersion string `json:"format_version"`
	Environment   struct {
		Name   string `json:"name"`
		S3Path string `json:"s3_path"`
	} `json:"environment"`
	Versions        []Version `json:"versions"`
	LatestVersionID string    `json:"latest_version_id"`
}

// VersionQueryOptions represents options for querying versions
type VersionQueryOptions struct {
	Since        time.Time
	Before       time.Time
	Limit        int
	SortByDate   bool
	LatestOnly   bool
	DeployedOnly bool
	SearchText   string
}

// getVersionManagementPath returns the S3 path for version management file
func getVersionManagementPath(env *config.Environment) string {
	return fmt.Sprintf("%s/.versions/%s.versions.json", env.S3.Prefix, env.S3.TFVarsKey)
}

// getVersionManagement retrieves the version management file from S3
func getVersionManagement(env *config.Environment) (*VersionManagement, error) {
	cfg, err := getAWSConfig(env.AWS.Region)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS config: %w", err)
	}

	s3Client := s3.NewFromConfig(cfg)
	key := getVersionManagementPath(env)

	result, err := s3Client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(env.S3.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			// Return new empty version management if file doesn't exist
			return &VersionManagement{
				FormatVersion: VersionManagementFormatVersion,
				Environment: struct {
					Name   string `json:"name"`
					S3Path string `json:"s3_path"`
				}{
					Name:   env.Name,
					S3Path: env.GetS3Path(),
				},
				Versions: make([]Version, 0),
			}, nil
		}
		return nil, fmt.Errorf("failed to get version management file: %w", err)
	}
	defer result.Body.Close()

	var versionMgmt VersionManagement
	if err := json.NewDecoder(result.Body).Decode(&versionMgmt); err != nil {
		return nil, fmt.Errorf("failed to decode version management file: %w", err)
	}

	return &versionMgmt, nil
}

// saveVersionManagement saves the version management file to S3
func saveVersionManagement(env *config.Environment, versionMgmt *VersionManagement) error {
	cfg, err := getAWSConfig(env.AWS.Region)
	if err != nil {
		return fmt.Errorf("unable to load AWS config: %w", err)
	}

	s3Client := s3.NewFromConfig(cfg)
	key := getVersionManagementPath(env)

	data, err := json.MarshalIndent(versionMgmt, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal version management: %w", err)
	}

	_, err = s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(env.S3.Bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		return fmt.Errorf("failed to save version management file: %w", err)
	}

	return nil
}

// AddVersion adds a new version to the version management file
func AddVersion(env *config.Environment, version *Version) error {
	versionMgmt, err := getVersionManagement(env)
	if err != nil {
		return fmt.Errorf("failed to get version management: %w", err)
	}

	// Add new version
	versionMgmt.Versions = append(versionMgmt.Versions, *version)
	versionMgmt.LatestVersionID = version.VersionID

	// Save updated version management
	if err := saveVersionManagement(env, versionMgmt); err != nil {
		return fmt.Errorf("failed to save version management: %w", err)
	}

	return nil
}

// GetVersions retrieves versions based on query options
func GetVersions(env *config.Environment, options *VersionQueryOptions) ([]Version, error) {
	versionMgmt, err := getVersionManagement(env)
	if err != nil {
		return nil, fmt.Errorf("failed to get version management: %w", err)
	}

	if len(versionMgmt.Versions) == 0 {
		return []Version{}, nil
	}

	// Filter and sort versions
	versions := filterVersions(versionMgmt.Versions, options)

	// Apply limit if specified
	if options != nil && options.Limit > 0 && len(versions) > options.Limit {
		versions = versions[:options.Limit]
	}

	return versions, nil
}

// filterVersions applies filtering and sorting based on options
func filterVersions(versions []Version, options *VersionQueryOptions) []Version {
	if options == nil {
		return versions
	}

	var filtered []Version
	for _, v := range versions {
		if !filterVersion(v, options) {
			continue
		}
		filtered = append(filtered, v)
	}

	// Sort versions by date if requested
	if options.SortByDate {
		sortVersionsByDate(filtered)
	}

	return filtered
}

// filterVersion checks if a version matches the query options
func filterVersion(version Version, options *VersionQueryOptions) bool {
	// Check date range
	if !options.Since.IsZero() && version.Timestamp.Before(options.Since) {
		return false
	}
	if !options.Before.IsZero() && version.Timestamp.After(options.Before) {
		return false
	}

	// Check description text
	if options.SearchText != "" && !strings.Contains(strings.ToLower(version.Description), strings.ToLower(options.SearchText)) {
		return false
	}

	return true
}

// sortVersionsByDate sorts versions by timestamp in descending order
func sortVersionsByDate(versions []Version) {
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Timestamp.After(versions[j].Timestamp)
	})
}
