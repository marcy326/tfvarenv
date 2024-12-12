package utils

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type VersionInfo struct {
	VersionID   string    `json:"version_id"`
	Hash        string    `json:"hash"`
	Timestamp   time.Time `json:"timestamp"`
	Description string    `json:"description"`
	Size        int64     `json:"size"`
}

// getAWSConfig loads AWS configuration for the specified region
func getAWSConfig(region string) (aws.Config, error) {
	return config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(aws.NewCredentialsCache(aws.CredentialsProviderFunc(
			func(ctx context.Context) (aws.Credentials, error) {
				return aws.Credentials{
					AccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
					SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
					SessionToken:    os.Getenv("AWS_SESSION_TOKEN"),
				}, nil
			},
		))),
	)
}

// GetAWSAccountID retrieves the current AWS account ID
func GetAWSAccountID(region string) (string, error) {
	cfg, err := getAWSConfig(region)
	if err != nil {
		return "", fmt.Errorf("unable to load AWS config: %w", err)
	}

	stsClient := sts.NewFromConfig(cfg)
	output, err := stsClient.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", fmt.Errorf("failed to get caller identity: %w", err)
	}

	return *output.Account, nil
}

// CheckS3BucketVersioning verifies if versioning is enabled for the specified bucket
func CheckS3BucketVersioning(bucket string) error {
	cfg, err := getAWSConfig("") // Region doesn't matter for this check
	if err != nil {
		return fmt.Errorf("unable to load AWS config: %w", err)
	}

	s3Client := s3.NewFromConfig(cfg)
	output, err := s3Client.GetBucketVersioning(context.TODO(), &s3.GetBucketVersioningInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		return fmt.Errorf("failed to check bucket versioning: %w", err)
	}

	if output.Status != types.BucketVersioningStatusEnabled {
		return fmt.Errorf("versioning is not enabled on bucket %s", bucket)
	}

	return nil
}

// parseS3URI parses an S3 URI into bucket and key
func parseS3URI(s3URI string) (string, string, error) {
	if !strings.HasPrefix(s3URI, "s3://") {
		return "", "", fmt.Errorf("invalid S3 URI format: %s", s3URI)
	}

	parts := strings.SplitN(s3URI[5:], "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid S3 URI format: %s", s3URI)
	}

	return parts[0], parts[1], nil
}

// CalculateFileHash generates SHA-256 hash for a file
func CalculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// UploadToS3WithVersioning uploads a file to S3 with versioning
func UploadToS3WithVersioning(localPath, s3URI, region, description string) (*VersionInfo, error) {
	bucket, key, err := parseS3URI(s3URI)
	if err != nil {
		return nil, err
	}

	cfg, err := getAWSConfig(region)
	if err != nil {
		return nil, err
	}

	s3Client := s3.NewFromConfig(cfg)

	// Calculate file hash
	hash, err := CalculateFileHash(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate file hash: %w", err)
	}

	// Open and get file size
	file, err := os.Open(localPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}

	// Upload file
	result, err := s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String("application/x-tfvars"),
		Metadata: map[string]string{
			"Hash":        hash,
			"Description": description,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	return &VersionInfo{
		VersionID:   *result.VersionId,
		Hash:        hash,
		Timestamp:   time.Now(),
		Description: description,
		Size:        fileInfo.Size(),
	}, nil
}

// DownloadFromS3WithVersion downloads a specific version of a file from S3
func DownloadFromS3WithVersion(s3URI, localPath, region, versionID string) (string, error) {
	bucket, key, err := parseS3URI(s3URI)
	if err != nil {
		return "", err
	}

	cfg, err := getAWSConfig(region)
	if err != nil {
		return "", err
	}

	s3Client := s3.NewFromConfig(cfg)

	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	if versionID != "" {
		input.VersionId = aws.String(versionID)
	}

	result, err := s3Client.GetObject(context.TODO(), input)
	if err != nil {
		return "", fmt.Errorf("failed to download file: %w", err)
	}
	defer result.Body.Close()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return "", err
	}

	file, err := os.Create(localPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := io.Copy(file, result.Body); err != nil {
		return "", err
	}

	return localPath, nil
}

// DownloadFromS3 downloads the latest version of a file from S3
func DownloadFromS3(s3URI, localPath, region string) (string, error) {
	return DownloadFromS3WithVersion(s3URI, localPath, region, "")
}

// GetLatestS3Version gets information about the latest version of a file in S3
func GetLatestS3Version(bucket, s3URI, region string) (*VersionInfo, error) {
	_, key, err := parseS3URI(s3URI)
	if err != nil {
		return nil, err
	}

	cfg, err := getAWSConfig(region)
	if err != nil {
		return nil, err
	}

	s3Client := s3.NewFromConfig(cfg)

	result, err := s3Client.ListObjectVersions(context.TODO(), &s3.ListObjectVersionsInput{
		Bucket:  aws.String(bucket),
		Prefix:  aws.String(key),
		MaxKeys: aws.Int32(1),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list versions: %w", err)
	}

	if len(result.Versions) == 0 {
		return nil, fmt.Errorf("no versions found for %s", s3URI)
	}

	latestVersion := result.Versions[0]

	// Get object metadata
	obj, err := s3Client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket:    aws.String(bucket),
		Key:       aws.String(key),
		VersionId: latestVersion.VersionId,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object metadata: %w", err)
	}

	return &VersionInfo{
		VersionID:   *latestVersion.VersionId,
		Hash:        obj.Metadata["Hash"],
		Timestamp:   *latestVersion.LastModified,
		Description: obj.Metadata["Description"],
		Size:        *latestVersion.Size,
	}, nil
}

// ListS3Versions lists all versions of a file in S3
func ListS3Versions(s3URI, region string) ([]VersionInfo, error) {
	bucket, key, err := parseS3URI(s3URI)
	if err != nil {
		return nil, err
	}

	cfg, err := getAWSConfig(region)
	if err != nil {
		return nil, err
	}

	s3Client := s3.NewFromConfig(cfg)

	var versions []VersionInfo
	input := &s3.ListObjectVersionsInput{
		Bucket: aws.String(bucket),
		Prefix: aws.String(key),
	}

	result, err := s3Client.ListObjectVersions(context.TODO(), input)
	if err != nil {
		return nil, fmt.Errorf("failed to list versions: %w", err)
	}

	for _, version := range result.Versions {
		// Get object metadata for each version
		obj, err := s3Client.HeadObject(context.TODO(), &s3.HeadObjectInput{
			Bucket:    aws.String(bucket),
			Key:       aws.String(key),
			VersionId: version.VersionId,
		})
		if err != nil {
			continue // Skip versions we can't get metadata for
		}

		versions = append(versions, VersionInfo{
			VersionID:   *version.VersionId,
			Hash:        obj.Metadata["Hash"],
			Timestamp:   *version.LastModified,
			Description: obj.Metadata["Description"],
			Size:        *version.Size,
		})
	}

	return versions, nil
}

// DeploymentInfo represents a deployment record
type DeploymentInfo struct {
	VersionID    string    `json:"version_id"`
	DeployedAt   time.Time `json:"deployed_at"`
	DeployedBy   string    `json:"deployed_by"`
	TerraformCmd string    `json:"terraform_cmd"`
}

// RecordDeployment records deployment information to S3
func RecordDeployment(bucket, key string, region string, info DeploymentInfo) error {
	cfg, err := getAWSConfig(region)
	if err != nil {
		return fmt.Errorf("unable to load AWS config: %w", err)
	}

	s3Client := s3.NewFromConfig(cfg)

	// Get existing deployments
	deployments := []DeploymentInfo{}
	result, err := s3Client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		// If file doesn't exist, start with empty list
		var notFound *types.NoSuchKey
		if !errors.As(err, &notFound) {
			return fmt.Errorf("failed to get deployment history: %w", err)
		}
	} else {
		defer result.Body.Close()
		if err := json.NewDecoder(result.Body).Decode(&deployments); err != nil {
			return fmt.Errorf("failed to decode deployment history: %w", err)
		}
	}

	// Add new deployment record
	deployments = append(deployments, info)

	// Marshal updated deployments
	data, err := json.Marshal(deployments)
	if err != nil {
		return fmt.Errorf("failed to marshal deployment history: %w", err)
	}

	// Upload updated deployment history
	_, err = s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return fmt.Errorf("failed to save deployment history: %w", err)
	}

	return nil
}

func ListDeployments(bucket, key, region string) ([]DeploymentInfo, error) {
	cfg, err := getAWSConfig(region)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS config: %w", err)
	}

	s3Client := s3.NewFromConfig(cfg)

	// Get deployment history file
	result, err := s3Client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var notFound *types.NoSuchKey
		if errors.As(err, &notFound) {
			return []DeploymentInfo{}, nil
		}
		return nil, fmt.Errorf("failed to get deployment history: %w", err)
	}
	defer result.Body.Close()

	// Decode deployment history
	var deployments []DeploymentInfo
	if err := json.NewDecoder(result.Body).Decode(&deployments); err != nil {
		return nil, fmt.Errorf("failed to decode deployment history: %w", err)
	}

	return deployments, nil
}
