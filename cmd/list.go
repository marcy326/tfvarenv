package cmd

import (
	"fmt"
	"os"
	"tfvarenv/config"
	"tfvarenv/utils"

	"github.com/spf13/cobra"
)

func NewListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all environments",
		Run: func(cmd *cobra.Command, args []string) {
			envs, err := config.ListEnvironments()
			if err != nil {
				fmt.Printf("Error listing environments: %s\n", err)
				os.Exit(1)
			}

			fmt.Println("Available environments:")
			for _, envName := range envs {
				env, err := config.GetEnvironmentInfo(envName)
				if err != nil {
					continue
				}

				// Get latest version information
				versionInfo, err := utils.GetLatestS3Version(env.S3.Bucket, env.GetS3Path(), env.AWS.Region)
				versionStatus := "Unknown"
				if err == nil {
					versionStatus = fmt.Sprintf("Version: %s", versionInfo.VersionID[:8])
				}

				fmt.Printf("\nEnvironment: %s\n", envName)
				fmt.Printf("  Description: %s\n", env.Description)
				fmt.Printf("  S3 Path: %s\n", env.GetS3Path())
				fmt.Printf("  AWS Account: %s (%s)\n", env.AWS.AccountID, env.AWS.Region)
				fmt.Printf("  Local Path: %s\n", env.Local.TFVarsPath)
				fmt.Printf("  Status: %s\n", versionStatus)
			}
		},
	}
}
