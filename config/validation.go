package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// validateEnvironment performs validation of environment configuration
func validateEnvironment(env *Environment) error {
	if env.Name == "" {
		return errors.New("environment name is required")
	}

	if err := validateS3Config(&env.S3); err != nil {
		return err
	}

	if err := validateAWSConfig(&env.AWS); err != nil {
		return err
	}

	if err := validateLocalConfig(&env.Local); err != nil {
		return err
	}

	return validateDeploymentConfig(&env.Deployment)
}

func validateS3Config(s3 *EnvironmentS3Config) error {
	if s3.Bucket == "" {
		return errors.New("S3 bucket is required")
	}

	if s3.Prefix == "" {
		return errors.New("S3 prefix is required")
	}

	if s3.TFVarsKey == "" {
		s3.TFVarsKey = "terraform.tfvars"
	}

	return nil
}

func validateAWSConfig(aws *AWSConfig) error {
	if aws.AccountID == "" {
		return errors.New("AWS account ID is required")
	}

	if aws.Region == "" {
		aws.Region = "ap-northeast-1"
	}

	return nil
}

func validateLocalConfig(local *LocalConfig) error {
	if local.TFVarsPath == "" {
		return errors.New("local tfvars path is required")
	}

	dir := filepath.Dir(local.TFVarsPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory for tfvars: %w", err)
	}

	return nil
}

func validateDeploymentConfig(deploy *DeploymentConfig) error {
	// Currently no validation rules for deployment config
	return nil
}
