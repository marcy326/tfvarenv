package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"tfvarenv/config"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

const DeploymentHistoryFormatVersion = "1.0"

// DeploymentHistory represents the deployment history file structure
type DeploymentHistory struct {
	FormatVersion    string             `json:"format_version"`
	Environment      string             `json:"environment"`
	LatestDeployment *DeploymentRecord  `json:"latest_deployment,omitempty"`
	Deployments      []DeploymentRecord `json:"deployments"`
}

// DeploymentRecord represents a single deployment record
type DeploymentRecord struct {
	Timestamp  time.Time `json:"timestamp"`
	VersionID  string    `json:"version_id"`
	DeployedBy string    `json:"deployed_by"`
	Command    string    `json:"command"`
	Status     string    `json:"status"`
}

// GetDeploymentHistory retrieves the deployment history for an environment
func GetDeploymentHistory(env *config.Environment) (*DeploymentHistory, error) {
	cfg, err := getAWSConfig(env.AWS.Region)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS config: %w", err)
	}

	s3Client := s3.NewFromConfig(cfg)
	key := env.GetDeploymentHistoryKey()

	result, err := s3Client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(env.S3.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			// Return new empty history if file doesn't exist
			return &DeploymentHistory{
				FormatVersion: DeploymentHistoryFormatVersion,
				Environment:   env.Name,
				Deployments:   make([]DeploymentRecord, 0),
			}, nil
		}
		return nil, fmt.Errorf("failed to get deployment history file: %w", err)
	}
	defer result.Body.Close()

	var history DeploymentHistory
	if err := json.NewDecoder(result.Body).Decode(&history); err != nil {
		return nil, fmt.Errorf("failed to decode deployment history: %w", err)
	}

	return &history, nil
}

// AddDeploymentRecord adds a new deployment record to the history
func AddDeploymentRecord(env *config.Environment, record *DeploymentRecord) error {
	history, err := GetDeploymentHistory(env)
	if err != nil {
		return fmt.Errorf("failed to get deployment history: %w", err)
	}

	// Add new record
	history.Deployments = append(history.Deployments, *record)
	history.LatestDeployment = record

	// Sort deployments by timestamp (newest first)
	sort.Slice(history.Deployments, func(i, j int) bool {
		return history.Deployments[i].Timestamp.After(history.Deployments[j].Timestamp)
	})

	// Save updated history
	if err := saveDeploymentHistory(env, history); err != nil {
		return fmt.Errorf("failed to save deployment history: %w", err)
	}

	return nil
}

// GetLatestDeployment returns the most recent deployment record
func GetLatestDeployment(env *config.Environment) (*DeploymentRecord, error) {
	history, err := GetDeploymentHistory(env)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment history: %w", err)
	}

	if history.LatestDeployment != nil {
		return history.LatestDeployment, nil
	}

	if len(history.Deployments) > 0 {
		return &history.Deployments[0], nil
	}

	return nil, nil
}

// saveDeploymentHistory saves the deployment history file to S3
func saveDeploymentHistory(env *config.Environment, history *DeploymentHistory) error {
	cfg, err := getAWSConfig(env.AWS.Region)
	if err != nil {
		return fmt.Errorf("unable to load AWS config: %w", err)
	}

	s3Client := s3.NewFromConfig(cfg)
	key := env.GetDeploymentHistoryKey()

	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal deployment history: %w", err)
	}

	_, err = s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(env.S3.Bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		return fmt.Errorf("failed to save deployment history file: %w", err)
	}

	return nil
}
