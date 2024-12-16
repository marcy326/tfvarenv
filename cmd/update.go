package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"tfvarenv/config"
	"tfvarenv/utils/command"
	"tfvarenv/utils/prompt"
)

func NewUpdateCmd() *cobra.Command {
	utils, err := command.NewUtils()
	if err != nil {
		fmt.Printf("Error initializing command utils: %v\n", err)
		os.Exit(1)
	}

	updateCmd := &cobra.Command{
		Use:   "update [environment]",
		Short: "Update an existing environment",
		Long:  `Update an existing environment in tfvarenv.`,
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
	reader := bufio.NewReader(os.Stdin)

	env, err := utils.GetEnvironment(envName)
	if err != nil {
		return fmt.Errorf("failed to get environment info: %w", err)
	}

	// Show current configuration
	fmt.Printf("\nCurrent configuration for environment '%s':\n", envName)
	if env.Description != "" {
		fmt.Printf("  Description: %s\n", env.Description)
	}
	fmt.Printf("  AWS Account: %s\n", env.AWS.AccountID)
	fmt.Printf("  Region: %s\n", env.AWS.Region)
	fmt.Printf("  S3 Bucket: %s\n", env.S3.Bucket)
	fmt.Printf("  S3 Prefix: %s\n", env.S3.Prefix)
	fmt.Printf("  Local Path: %s\n", env.Local.TFVarsPath)
	fmt.Printf("  Auto Backup: %v\n", env.Deployment.AutoBackup)
	fmt.Printf("  Require Approval: %v\n", env.Deployment.RequireApproval)
	fmt.Printf("  Backend Bucket: %s\n", env.Backend.Bucket)
	fmt.Printf("  Backend Key: %s\n", env.Backend.Key)
	fmt.Printf("  Backend Region: %s\n", env.Backend.Region)

	// Update fields
	fmt.Print("\nEnter new values (press Enter to keep current value)\n")

	// Environment Name
	var newEnvName string
	fmt.Printf("Environment Name [%s]: ", envName)
	if name, _ := reader.ReadString('\n'); strings.TrimSpace(name) != "" {
		newEnvName = strings.TrimSpace(name)
	}

	// Description
	fmt.Printf("Description [%s]: ", env.Description)
	if desc, _ := reader.ReadString('\n'); strings.TrimSpace(desc) != "" {
		env.Description = strings.TrimSpace(desc)
	}

	// AWS Account ID
	fmt.Printf("AWS Account ID [%s]: ", env.AWS.AccountID)
	if accountID, _ := reader.ReadString('\n'); strings.TrimSpace(accountID) != "" {
		env.AWS.AccountID = strings.TrimSpace(accountID)
	}

	// AWS Region
	fmt.Printf("AWS Region [%s]: ", env.AWS.Region)
	if region, _ := reader.ReadString('\n'); strings.TrimSpace(region) != "" {
		env.AWS.Region = strings.TrimSpace(region)
	}

	// S3 Bucket
	fmt.Printf("S3 bucket name [%s]: ", env.S3.Bucket)
	if bucket, _ := reader.ReadString('\n'); strings.TrimSpace(bucket) != "" {
		env.S3.Bucket = strings.TrimSpace(bucket)
	}

	// S3 Prefix
	fmt.Printf("S3 prefix [%s]: ", env.S3.Prefix)
	if prefix, _ := reader.ReadString('\n'); strings.TrimSpace(prefix) != "" {
		env.S3.Prefix = strings.TrimSpace(prefix)
	}

	// TFVars Key
	fmt.Printf("tfvars file name [%s]: ", env.S3.TFVarsKey)
	if tfvarsKey, _ := reader.ReadString('\n'); strings.TrimSpace(tfvarsKey) != "" {
		env.S3.TFVarsKey = strings.TrimSpace(tfvarsKey)
	}

	// Local Configuration
	fmt.Printf("Local tfvars path [%s]: ", env.Local.TFVarsPath)
	if path, _ := reader.ReadString('\n'); strings.TrimSpace(path) != "" {
		env.Local.TFVarsPath = strings.TrimSpace(path)
	}

	// Deployment Configuration
	env.Deployment.AutoBackup = prompt.PromptYesNo("Enable auto backup?", env.Deployment.AutoBackup)
	env.Deployment.RequireApproval = prompt.PromptYesNo("Require deployment approval?", env.Deployment.RequireApproval)

	// Backend Configuration
	fmt.Printf("Backend bucket name [%s]: ", env.Backend.Bucket)
	if backendBucket, _ := reader.ReadString('\n'); strings.TrimSpace(backendBucket) != "" {
		env.Backend.Bucket = strings.TrimSpace(backendBucket)
	}

	fmt.Printf("Backend key [%s]: ", env.Backend.Key)
	if backendKey, _ := reader.ReadString('\n'); strings.TrimSpace(backendKey) != "" {
		env.Backend.Key = strings.TrimSpace(backendKey)
	}

	fmt.Printf("Backend region [%s]: ", env.Backend.Region)
	if backendRegion, _ := reader.ReadString('\n'); strings.TrimSpace(backendRegion) != "" {
		env.Backend.Region = strings.TrimSpace(backendRegion)
	}

	// Update configuration
	if newEnvName != "" && newEnvName != envName {
		// Updating with new environment name
		env.Name = newEnvName // Update the name in the environment struct

		// First remove the old environment, then add the new one
		cfg, err := config.NewManager()
		if err != nil {
			return fmt.Errorf("failed to initialize config manager: %w", err)
		}

		if err := cfg.RemoveEnvironment(envName); err != nil {
			return fmt.Errorf("failed to remove old environment: %w", err)
		}

		if err := cfg.AddEnvironment(newEnvName, env); err != nil {
			return fmt.Errorf("failed to add updated environment: %w", err)
		}
	} else {
		// Updating existing environment
		cfg, err := config.NewManager()
		if err != nil {
			return fmt.Errorf("failed to initialize config manager: %w", err)
		}
		if err := cfg.UpdateEnvironment(envName, env); err != nil {
			return fmt.Errorf("failed to update environment: %w", err)
		}
	}

	fmt.Println("\nEnvironment updated successfully!")

	fmt.Printf("\nFile locations:\n")
	fmt.Printf("  Local: %s\n", env.Local.TFVarsPath)
	fmt.Printf("  Remote: %s\n", env.GetFullS3Path())

	fmt.Println("\nUse the following commands to manage tfvars:")
	fmt.Printf("- Download: tfvarenv download %s\n", env.Name)
	fmt.Printf("- Upload:   tfvarenv upload %s\n", env.Name)
	fmt.Printf("- Plan:     tfvarenv plan %s\n", env.Name)
	fmt.Printf("- Apply:    tfvarenv apply %s\n", env.Name)

	return nil
}
