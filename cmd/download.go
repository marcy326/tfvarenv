package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"tfvarenv/utils/aws"
	"tfvarenv/utils/command"
	"tfvarenv/utils/deployment"
	"tfvarenv/utils/file"
	"tfvarenv/utils/prompt"
	"tfvarenv/utils/version"
)

func NewDownloadCmd() *cobra.Command {
	utils, err := command.NewUtils()
	if err != nil {
		fmt.Printf("Error initializing command utils: %v\n", err)
		os.Exit(1)
	}

	var (
		versionID string
		force     bool
	)

	downloadCmd := &cobra.Command{
		Use:   "download [environment]",
		Short: "Download tfvars file from S3 to local path",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := runDownload(cmd.Context(), utils, args[0], versionID, force); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	downloadCmd.Flags().StringVar(&versionID, "version-id", "", "Specific version ID to download")
	downloadCmd.Flags().BoolVarP(&force, "force", "f", false, "Force download without confirmation")

	return downloadCmd
}

func runDownload(ctx context.Context, utils command.Utils, envName, versionID string, force bool) error {
	env, err := utils.GetEnvironment(envName)
	if err != nil {
		return fmt.Errorf("environment not found: %w", err)
	}

	versionManager := version.NewManager(utils.GetAWSClient(), utils.GetFileUtils(), env)

	// Get version information
	var ver *version.Version
	if versionID != "" {
		ver, err = versionManager.GetVersion(ctx, versionID)
	} else {
		ver, err = versionManager.GetLatestVersion(ctx)
	}
	if err != nil {
		return fmt.Errorf("failed to get version information: %w", err)
	}

	// Display version information
	fmt.Printf("\nDownloading version:\n")
	fmt.Printf("  Version ID: %s\n", ver.VersionID[:8])
	fmt.Printf("  Uploaded: %s by %s\n",
		ver.Timestamp.Format("2006-01-02 15:04:05"),
		ver.UploadedBy)
	if ver.Description != "" {
		fmt.Printf("  Description: %s\n", ver.Description)
	}

	// Get deployment status
	deploymentManager := deployment.NewManager(utils.GetAWSClient(), env)
	if latestDeploy, err := deploymentManager.GetLatestDeployment(ctx); err == nil && latestDeploy != nil {
		if latestDeploy.VersionID == ver.VersionID {
			fmt.Printf("  Deployment Status: Currently deployed (since %s)\n",
				latestDeploy.Timestamp.Format("2006-01-02 15:04:05"))
		}
	}

	// Check local file
	fileUtils := utils.GetFileUtils()
	exists, err := fileUtils.FileExists(env.Local.TFVarsPath)
	if err != nil {
		return fmt.Errorf("failed to check local file: %w", err)
	}

	if exists {
		localHash, err := fileUtils.CalculateHash(env.Local.TFVarsPath, nil)
		if err != nil {
			return fmt.Errorf("failed to calculate local file hash: %w", err)
		}

		if localHash == ver.Hash {
			fmt.Println("\nLocal file is identical to the requested version. No download needed.")
			return nil
		}

		if !force {
			fmt.Printf("\nLocal file %s already exists.\n", env.Local.TFVarsPath)
			if !prompt.PromptYesNo("Do you want to overwrite it?", false) {
				return fmt.Errorf("download cancelled by user")
			}
		}

		// Create backup
		backupOpts := &file.BackupOptions{
			BasePath:   filepath.Join(".backups", envName),
			TimeFormat: "20060102150405",
		}
		if backupPath, err := fileUtils.CreateBackup(env.Local.TFVarsPath, backupOpts); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		} else {
			fmt.Printf("\nCreated backup: %s\n", backupPath)
		}
	}

	// Download the file
	downloadInput := &aws.DownloadInput{
		Bucket:    env.S3.Bucket,
		Key:       env.GetS3Path(),
		VersionID: ver.VersionID,
	}
	output, err := utils.GetAWSClient().DownloadFile(ctx, downloadInput)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}

	// Write the file
	writeOpts := &file.Options{
		CreateDirs: true,
		Overwrite:  true,
	}
	if err := fileUtils.WriteFile(env.Local.TFVarsPath, output.Content, writeOpts); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("\nSuccessfully downloaded to: %s\n", env.Local.TFVarsPath)
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  Review changes: cat %s\n", env.Local.TFVarsPath)
	fmt.Printf("  Plan changes:   tfvarenv plan %s\n", envName)
	fmt.Printf("  Apply changes:  tfvarenv apply %s\n", envName)

	return nil
}
