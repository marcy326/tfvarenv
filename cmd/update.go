package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"tfvarenv/utils/command"
)

func NewUpdateCmd() *cobra.Command {
	utils, err := command.NewUtils()
	if err != nil {
		fmt.Printf("Error initializing command utils: %v\n", err)
		os.Exit(1)
	}

	updateCmd := &cobra.Command{
		Use:   "update [environment]",
		Short: "Update environment configuration",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := runUpdate(cmd.Context(), utils, args[0]); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	return updateCmd
}

func runUpdate(ctx context.Context, utils command.Utils, envName string) error {
	// Get current environment configuration
	env, err := utils.GetEnvironment(envName)
	if err != nil {
		return fmt.Errorf("environment not found: %w", err)
	}

	reader := bufio.NewReader(os.Stdin)

	// Show current configuration
	fmt.Printf("\nCurrent configuration for environment '%s':\n", envName)
	if env.Description != "" {
		fmt.Printf("Description: %s\n", env.Description)
	}
	fmt.Printf("AWS Account: %s\n", env.AWS.AccountID)
	fmt.Printf("Region: %s\n", env.AWS.Region)
	fmt.Printf("S3 Bucket: %s\n", env.S3.Bucket)
	fmt.Printf("S3 Prefix: %s\n", env.S3.Prefix)
	fmt.Printf("Local Path: %s\n", env.Local.TFVarsPath)
	fmt.Printf("Auto Backup: %v\n", env.Deployment.AutoBackup)
	fmt.Printf("Require Approval: %v\n", env.Deployment.RequireApproval)

	// Update fields
	fmt.Print("\nEnter new values (press Enter to keep current value)\n")

	// Description
	fmt.Printf("Description [%s]: ", env.Description)
	if desc, _ := reader.ReadString('\n'); strings.TrimSpace(desc) != "" {
		env.Description = strings.TrimSpace(desc)
	}

	// AWS Region
	fmt.Printf("AWS Region [%s]: ", env.AWS.Region)
	if region, _ := reader.ReadString('\n'); strings.TrimSpace(region) != "" {
		region = strings.TrimSpace(region)
		// Update AWS client and verify account
		awsClient, err := utils.GetAWSClientWithRegion(region)
		if err != nil {
			return fmt.Errorf("failed to initialize AWS client: %w", err)
		}
		accountID, err := awsClient.GetAccountID(ctx)
		if err != nil {
			return fmt.Errorf("failed to get AWS account ID: %w", err)
		}
		env.AWS.Region = region
		env.AWS.AccountID = accountID
	}

	// S3 Configuration
	fmt.Printf("S3 Bucket [%s]: ", env.S3.Bucket)
	if bucket, _ := reader.ReadString('\n'); strings.TrimSpace(bucket) != "" {
		bucket = strings.TrimSpace(bucket)
		if err := utils.GetAWSClient().CheckBucketVersioning(ctx, bucket); err != nil {
			return fmt.Errorf("S3 bucket verification failed: %w", err)
		}
		env.S3.Bucket = bucket
	}

	fmt.Printf("S3 Prefix [%s]: ", env.S3.Prefix)
	if prefix, _ := reader.ReadString('\n'); strings.TrimSpace(prefix) != "" {
		env.S3.Prefix = strings.TrimSpace(prefix)
	}

	// Local Configuration
	fmt.Printf("Local tfvars path [%s]: ", env.Local.TFVarsPath)
	if path, _ := reader.ReadString('\n'); strings.TrimSpace(path) != "" {
		env.Local.TFVarsPath = strings.TrimSpace(path)
	}

	// Deployment Configuration
	env.Deployment.AutoBackup = promptYesNo("Enable auto backup?", env.Deployment.AutoBackup)
	env.Deployment.RequireApproval = promptYesNo("Require deployment approval?", env.Deployment.RequireApproval)

	// Update configuration
	if err := utils.AddEnvironment(env); err != nil {
		return fmt.Errorf("failed to update environment: %w", err)
	}

	fmt.Printf("\nEnvironment '%s' updated successfully.\n", envName)
	return nil
}
