package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"tfvarenv/utils/command"
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

	// Verify backend config exists
	fileUtils := utils.GetFileUtils()
	exists, err := fileUtils.FileExists(env.Backend.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to check backend config: %w", err)
	}
	if !exists {
		return fmt.Errorf("backend config file not found at %s", env.Backend.ConfigPath)
	}

	// Initialize Terraform with backend configuration
	initOpts := &terraform.InitOptions{
		BackendConfig: env.Backend.ConfigPath,
		Reconfigure:   true,
		ForceCopy:     force,
	}

	fmt.Printf("Initializing Terraform backend...\n")
	result, err := utils.GetTerraformRunner().Init(ctx, initOpts)
	if err != nil {
		return fmt.Errorf("terraform init failed: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("terraform init failed: %s", result.ErrorOutput)
	}

	fmt.Printf("\nSuccessfully switched to environment '%s'\n", envName)

	// Show environment details
	fmt.Printf("\nEnvironment Details:\n")
	fmt.Printf("  AWS Account: %s\n", env.AWS.AccountID)
	fmt.Printf("  Region: %s\n", env.AWS.Region)
	fmt.Printf("  Backend Config: %s\n", env.Backend.ConfigPath)
	fmt.Printf("  Local tfvars: %s\n", env.Local.TFVarsPath)
	fmt.Printf("  S3 Path: %s\n", env.GetS3Path())

	// Get latest version information if available
	versionManager := version.NewManager(utils.GetAWSClient(), fileUtils, env)
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
