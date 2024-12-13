package cmd

import (
	"fmt"
	"os"
	"tfvarenv/config"
	"tfvarenv/utils"

	"github.com/spf13/cobra"
)

func NewUseCmd() *cobra.Command {
	var force bool

	useCmd := &cobra.Command{
		Use:   "use [environment]",
		Short: "Use and initialize the specified environment",
		Long: `Use and initialize the specified environment. 
This command switches the Terraform backend configuration and initializes the workspace.
Example: tfvarenv use production`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			envName := args[0]
			if err := runUse(envName, force); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	useCmd.Flags().BoolVarP(&force, "force", "f", false, "Force re-initialization")
	return useCmd
}

func runUse(envName string, force bool) error {
	env, err := config.GetEnvironmentInfo(envName)
	if err != nil {
		return fmt.Errorf("failed to get environment info: %w", err)
	}

	fmt.Printf("Switching to environment: %s\n", envName)
	if env.Description != "" {
		fmt.Printf("Description: %s\n", env.Description)
	}

	// Check if backend config file exists
	if _, err := os.Stat(env.Backend.ConfigPath); err != nil {
		return fmt.Errorf("backend config file not found at %s: %w", env.Backend.ConfigPath, err)
	}

	// Construct terraform init command
	args := []string{"init", "-reconfigure"}

	// Add backend config
	args = append(args, fmt.Sprintf("-backend-config=%s", env.Backend.ConfigPath))

	// Force flag
	if force {
		args = append(args, "-force-copy")
	}

	// Run terraform init
	fmt.Printf("Initializing Terraform backend...\n")
	if err := utils.RunCommand("terraform", args...); err != nil {
		return fmt.Errorf("terraform init failed: %w", err)
	}

	fmt.Printf("\nSuccessfully switched to environment '%s'\n", envName)

	// Show the current AWS account and region
	accountID := env.AWS.AccountID
	region := env.AWS.Region
	fmt.Printf("\nEnvironment Details:\n")
	fmt.Printf("  AWS Account: %s\n", accountID)
	fmt.Printf("  Region: %s\n", region)
	fmt.Printf("  Backend Config: %s\n", env.Backend.ConfigPath)

	// Show available next steps
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  Plan changes:   tfvarenv plan %s\n", envName)
	fmt.Printf("  Apply changes:  tfvarenv apply %s\n", envName)
	fmt.Printf("  Show versions:  tfvarenv versions %s\n", envName)

	return nil
}
