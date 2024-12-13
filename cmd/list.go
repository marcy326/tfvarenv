package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"tfvarenv/utils/command"
	"tfvarenv/utils/deployment"
	"tfvarenv/utils/version"
)

func NewListCmd() *cobra.Command {
	utils, err := command.NewUtils()
	if err != nil {
		fmt.Printf("Error initializing command utils: %v\n", err)
		os.Exit(1)
	}

	return &cobra.Command{
		Use:   "list",
		Short: "List all environments",
		Run: func(cmd *cobra.Command, args []string) {
			if err := runList(cmd.Context(), utils); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		},
	}
}

func runList(ctx context.Context, utils command.Utils) error {
	envNames, err := utils.ListEnvironments()
	if err != nil {
		return fmt.Errorf("failed to list environments: %w", err)
	}

	fmt.Println("Available environments:")
	for _, envName := range envNames {
		env, err := utils.GetEnvironment(envName)
		if err != nil {
			continue
		}

		fmt.Printf("\nEnvironment: %s\n", envName)
		if env.Description != "" {
			fmt.Printf("  Description: %s\n", env.Description)
		}
		fmt.Printf("  AWS Account: %s (%s)\n", env.AWS.AccountID, env.AWS.Region)
		fmt.Printf("  S3 Path: %s\n", env.GetS3Path())
		fmt.Printf("  Local Path: %s\n", env.Local.TFVarsPath)

		// Get version information
		versionManager := version.NewManager(utils.GetAWSClient(), utils.GetFileUtils(), env)
		latestVer, err := versionManager.GetLatestVersion(ctx)
		if err == nil && latestVer != nil {
			fmt.Printf("  Latest Version: %s (uploaded %s)\n",
				latestVer.VersionID[:8],
				latestVer.Timestamp.Format("2006-01-02 15:04:05"))
		} else {
			fmt.Printf("  Latest Version: None\n")
		}

		// Get deployment information
		deploymentManager := deployment.NewManager(utils.GetAWSClient(), env)
		latestDeploy, err := deploymentManager.GetLatestDeployment(ctx)
		if err == nil && latestDeploy != nil {
			fmt.Printf("  Last Deployed: %s by %s\n",
				latestDeploy.Timestamp.Format("2006-01-02 15:04:05"),
				latestDeploy.DeployedBy)
		} else {
			fmt.Printf("  Last Deployed: Never\n")
		}

		// Check local file status
		fileUtils := utils.GetFileUtils()
		exists, err := fileUtils.FileExists(env.Local.TFVarsPath)
		if err == nil {
			if exists {
				if latestVer != nil {
					hash, err := fileUtils.CalculateHash(env.Local.TFVarsPath, nil)
					if err == nil {
						if hash == latestVer.Hash {
							fmt.Printf("  Local Status: In sync with remote\n")
						} else {
							fmt.Printf("  Local Status: Different from remote\n")
						}
					}
				}
			} else {
				fmt.Printf("  Local Status: File not found\n")
			}
		}
	}

	return nil
}
