package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"tfvarenv/config"
	"tfvarenv/utils"

	"github.com/spf13/cobra"
)

func NewDownloadCmd() *cobra.Command {
	var versionID string

	downloadCmd := &cobra.Command{
		Use:   "download [environment]",
		Short: "Download tfvars file from S3 to local path",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			envName := args[0]
			if err := downloadTFVars(envName, versionID); err != nil {
				fmt.Printf("Error downloading tfvars: %v\n", err)
				os.Exit(1)
			}
		},
	}

	downloadCmd.Flags().StringVar(&versionID, "version-id", "", "Specific version ID to download")
	return downloadCmd
}

func downloadTFVars(envName, versionID string) error {
	env, err := config.GetEnvironmentInfo(envName)
	if err != nil {
		return fmt.Errorf("environment not found: %w", err)
	}

	// Ensure local directory exists
	if err := os.MkdirAll(filepath.Dir(env.Local.TFVarsPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(env.Local.TFVarsPath); err == nil {
		if !promptYesNo(fmt.Sprintf("File %s already exists. Overwrite?", env.Local.TFVarsPath), false) {
			return fmt.Errorf("download cancelled by user")
		}
	}

	// Download file
	var downloadPath string
	if versionID != "" {
		downloadPath, err = utils.DownloadFromS3WithVersion(
			env.GetS3Path(),
			env.Local.TFVarsPath,
			env.AWS.Region,
			versionID,
		)
	} else {
		downloadPath, err = utils.DownloadFromS3(
			env.GetS3Path(),
			env.Local.TFVarsPath,
			env.AWS.Region,
		)
	}

	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	fmt.Printf("Successfully downloaded %s to %s\n", env.GetS3Path(), downloadPath)
	return nil
}
