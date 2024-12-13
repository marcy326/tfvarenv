package config

import (
	"fmt"
)

const (
	configFileName = ".tfvarenv.json"
	defaultVersion = "1.0"
)

// Config represents the root configuration structure
type Config struct {
	Version       string                 `json:"version"`
	DefaultRegion string                 `json:"default_region"`
	Environments  map[string]Environment `json:"environments"`
}

// Environment represents a single environment configuration
type Environment struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	S3          EnvironmentS3Config `json:"s3"`
	AWS         AWSConfig           `json:"aws"`
	Local       LocalConfig         `json:"local"`
	Deployment  DeploymentConfig    `json:"deployment"`
	Backend     BackendConfig       `json:"backend"`
}

// EnvironmentS3Config represents environment-specific S3 settings
type EnvironmentS3Config struct {
	Bucket    string `json:"bucket"`
	Prefix    string `json:"prefix"`
	TFVarsKey string `json:"tfvars_key"`
}

// AWSConfig represents AWS-specific settings
type AWSConfig struct {
	AccountID string `json:"account_id"`
	Region    string `json:"region"`
}

// LocalConfig represents local file settings
type LocalConfig struct {
	TFVarsPath string `json:"tfvars_path"`
}

// DeploymentConfig represents deployment settings
type DeploymentConfig struct {
	AutoBackup      bool `json:"auto_backup"`
	RequireApproval bool `json:"require_approval"`
}

// BackendConfig represents backend configuration settings
type BackendConfig struct {
	ConfigPath string `json:"config_path"` // backend configファイルのパス
}

// GetS3Path returns the full S3 path for the tfvars file
func (e *Environment) GetS3Path() string {
	return fmt.Sprintf("s3://%s/%s/%s", e.S3.Bucket, e.S3.Prefix, e.S3.TFVarsKey)
}

// GetVersionMetadataKey returns the S3 key for version metadata
func (e *Environment) GetVersionMetadataKey() string {
	return fmt.Sprintf("%s/.%s.versions.json", e.S3.Prefix, e.S3.TFVarsKey)
}

// GetDeploymentHistoryKey returns the S3 key for deployment history
func (e *Environment) GetDeploymentHistoryKey() string {
	return fmt.Sprintf("%s/.deployments.json", e.S3.Prefix)
}
