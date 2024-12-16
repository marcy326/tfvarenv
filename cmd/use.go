package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"tfvarenv/config"
	"tfvarenv/utils/command"
	"tfvarenv/utils/file"
	"tfvarenv/utils/terraform"
	"tfvarenv/utils/version"
)

func NewUseCmd() *cobra.Command {
	utils, err := command.NewUtils()
	if err != nil {
		fmt.Printf("Error initializing command utils: %v\n", err)
		os.Exit(1)
	}

	var force bool

	useCmd := &cobra.Command{
		Use:   "use [environment]",
		Short: "Use and initialize the specified environment",
		Long: `Use and initialize the specified environment. 
This command switches the Terraform backend configuration and initializes the workspace.
Example: tfvarenv use production`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := runUse(cmd.Context(), utils, args[0], force); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	useCmd.Flags().BoolVarP(&force, "force", "f", false, "Force re-initialization")
	return useCmd
}

func runUse(ctx context.Context, utils command.Utils, envName string, force bool) error {
	env, err := utils.GetEnvironment(envName)
	if err != nil {
		return fmt.Errorf("failed to get environment info: %w", err)
	}

	fmt.Printf("Switching to environment: %s\n", envName)
	if env.Description != "" {
		fmt.Printf("Description: %s\n", env.Description)
	}

	// Initialize Terraform with backend configuration
	initOpts := &terraform.InitOptions{
		BackendConfigs: map[string]string{
			"bucket": env.Backend.Bucket,
			"key":    env.Backend.Key,
			"region": env.Backend.Region,
		},
		Reconfigure: true,
		ForceCopy:   force,
	}

	fmt.Printf("Initializing Terraform backend...\n")
	result, err := utils.GetTerraformRunner().Init(ctx, initOpts)
	if err != nil {
		return fmt.Errorf("terraform init failed: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("terraform init failed: %s", result.ErrorOutput)
	}
	// Check current backend configuration
	currentBackend, err := getCurrentBackendConfig(utils.GetFileUtils())
	if err != nil {
		fmt.Printf("Warning: Failed to get current backend config: %v\n", err)
	} else {
		if currentBackend.Bucket != env.Backend.Bucket ||
			currentBackend.Key != env.Backend.Key ||
			currentBackend.Region != env.Backend.Region {
			fmt.Println("\nWarning: Current backend configuration does not match the environment configuration.")
			fmt.Println("  Please run 'terraform init' again to update the backend configuration.")
		} else {
			fmt.Println("\nBackend configuration is up to date.")
		}
	}

	fmt.Printf("\nSuccessfully switched to environment '%s'\n", envName)

	// Show environment details
	fmt.Printf("\nEnvironment Details:\n")
	fmt.Printf("  AWS Account: %s\n", env.AWS.AccountID)
	fmt.Printf("  Region: %s\n", env.AWS.Region)
	fmt.Printf("  Backend Config:\n")
	fmt.Printf("    Bucket: %s\n", env.Backend.Bucket)
	fmt.Printf("    Key: %s\n", env.Backend.Key)
	fmt.Printf("    Region: %s\n", env.Backend.Region)
	fmt.Printf("  Local tfvars: %s\n", env.Local.TFVarsPath)
	fmt.Printf("  S3 Path: %s\n", env.GetS3Path())

	// Get latest version information if available
	versionManager := version.NewManager(utils.GetAWSClient(), utils.GetFileUtils(), env)
	if latestVer, err := versionManager.GetLatestVersion(ctx); err == nil {
		fmt.Printf("\nLatest Version Information:\n")
		fmt.Printf("  Version ID: %s\n", latestVer.VersionID[:8])
		fmt.Printf("  Uploaded: %s by %s\n",
			latestVer.Timestamp.Format("2006-01-02 15:04:05"),
			latestVer.UploadedBy)
		if latestVer.Description != "" {
			fmt.Printf("  Description: %s\n", latestVer.Description)
		}
	}

	// Show next steps
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  Plan changes:   tfvarenv plan %s\n", envName)
	fmt.Printf("  Apply changes:  tfvarenv apply %s\n", envName)
	fmt.Printf("  Show versions:  tfvarenv versions %s\n", envName)
	fmt.Printf("  Show history:   tfvarenv history %s\n", envName)

	return nil
}

func getCurrentBackendConfig(fileUtils file.Utils) (*config.BackendConfig, error) {
	statePath := ".terraform/terraform.tfstate"
	exists, err := fileUtils.FileExists(statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to check terraform state file: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("terraform state file not found at %s", statePath)
	}

	content, err := fileUtils.ReadFile(statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read terraform state file: %w", err)
	}

	var stateData map[string]interface{}
	if err := json.Unmarshal(content, &stateData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal terraform state file: %w", err)
	}

	backend, ok := stateData["backend"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("backend information not found in terraform state file")
	}

	backendConfig, ok := backend["config"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("backend config not found in terraform state file")
	}

	bucket, ok := backendConfig["bucket"].(string)
	if !ok {
		return nil, fmt.Errorf("bucket not found in terraform state file")
	}

	key, ok := backendConfig["key"].(string)
	if !ok {
		return nil, fmt.Errorf("key not found in terraform state file")
	}

	region, ok := backendConfig["region"].(string)
	if !ok {
		return nil, fmt.Errorf("region not found in terraform state file")
	}

	return &config.BackendConfig{
		Bucket: bucket,
		Key:    key,
		Region: region,
	}, nil
}
