package cmd

import (
	"fmt"
	"os"
	"tfvarenv/config"
	"tfvarenv/utils"

	"github.com/spf13/cobra"
)

func NewUploadCmd() *cobra.Command {
	var description string

	uploadCmd := &cobra.Command{
		Use:   "upload [environment]",
		Short: "Upload local tfvars file to S3",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			envName := args[0]
			if err := uploadTFVars(envName, description); err != nil {
				fmt.Printf("Error uploading tfvars: %v\n", err)
				os.Exit(1)
			}
		},
	}

	uploadCmd.Flags().StringVarP(&description, "description", "d", "", "Description for this version")
	return uploadCmd
}

func uploadTFVars(envName, description string) error {
	env, err := config.GetEnvironmentInfo(envName)
	if err != nil {
		return fmt.Errorf("environment not found: %w", err)
	}

	// Check if local file exists
	if _, err := os.Stat(env.Local.TFVarsPath); err != nil {
		return fmt.Errorf("local file not found: %w", err)
	}

	// Upload file with versioning
	versionInfo, err := utils.UploadToS3WithVersioning(
		env.Local.TFVarsPath,
		env.GetS3Path(),
		env.AWS.Region,
		description,
	)
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	fmt.Printf("Successfully uploaded %s to %s\n", env.Local.TFVarsPath, env.GetS3Path())
	fmt.Printf("Version ID: %s\n", versionInfo.VersionID)
	if description != "" {
		fmt.Printf("Description: %s\n", description)
	}

	return nil
}
