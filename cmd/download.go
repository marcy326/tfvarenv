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
	var (
		versionID string
		force     bool
	)

	downloadCmd := &cobra.Command{
		Use:   "download [environment]",
		Short: "Download tfvars file from S3 to local path",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			envName := args[0]
			if err := runDownload(envName, versionID, force); err != nil {
				fmt.Printf("Error downloading tfvars: %v\n", err)
				os.Exit(1)
			}
		},
	}

	downloadCmd.Flags().StringVar(&versionID, "version-id", "", "Specific version ID to download")
	downloadCmd.Flags().BoolVarP(&force, "force", "f", false, "Force download without confirmation")
	return downloadCmd
}

func runDownload(envName, versionID string, force bool) error {
	env, err := config.GetEnvironmentInfo(envName)
	if err != nil {
		return fmt.Errorf("environment not found: %w", err)
	}

	// Get version information
	var version *utils.Version
	if versionID != "" {
		versions, err := utils.GetVersions(env, &utils.VersionQueryOptions{})
		if err != nil {
			return fmt.Errorf("failed to get versions: %w", err)
		}
		for _, v := range versions {
			if v.VersionID == versionID {
				version = &v
				break
			}
		}
		if version == nil {
			return fmt.Errorf("version ID %s not found", versionID)
		}
	} else {
		versions, err := utils.GetVersions(env, &utils.VersionQueryOptions{
			LatestOnly: true,
		})
		if err != nil {
			return fmt.Errorf("failed to get versions: %w", err)
		}
		if len(versions) == 0 {
			return fmt.Errorf("no versions found")
		}
		version = &versions[0]
		versionID = version.VersionID
	}

	// Display version information
	fmt.Printf("\nDownloading version:\n")
	fmt.Printf("  Version ID: %s\n", version.VersionID[:8])
	fmt.Printf("  Uploaded: %s by %s\n", version.Timestamp.Format("2006-01-02 15:04:05"), version.UploadedBy)
	if version.Description != "" {
		fmt.Printf("  Description: %s\n", version.Description)
	}

	// Get and display deployment status
	if latestDeploy, err := utils.GetLatestDeployment(env); err == nil && latestDeploy != nil {
		if latestDeploy.VersionID == version.VersionID {
			fmt.Printf("  Deployment Status: Currently deployed (since %s)\n",
				latestDeploy.Timestamp.Format("2006-01-02 15:04:05"))
		}
	}

	// Check local file
	if _, err := os.Stat(env.Local.TFVarsPath); err == nil {
		localHash, err := utils.CalculateFileHash(env.Local.TFVarsPath)
		if err != nil {
			return fmt.Errorf("failed to calculate local file hash: %w", err)
		}

		if localHash == version.Hash {
			fmt.Println("\nLocal file is identical to the requested version. No download needed.")
			return nil
		}

		if !force {
			fmt.Printf("\nLocal file %s already exists.\n", env.Local.TFVarsPath)
			if !promptYesNo("Do you want to overwrite it?", false) {
				return fmt.Errorf("download cancelled by user")
			}
		}

		// Create backup
		backupDir := filepath.Join(".backups", envName)
		if err := os.MkdirAll(backupDir, 0755); err != nil {
			return fmt.Errorf("failed to create backup directory: %w", err)
		}

		backupPath := filepath.Join(backupDir,
			fmt.Sprintf("terraform.tfvars.backup.%s", filepath.Base(env.Local.TFVarsPath)))
		if err := utils.CopyFile(env.Local.TFVarsPath, backupPath); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
		fmt.Printf("\nCreated backup: %s\n", backupPath)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(env.Local.TFVarsPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Download the file
	if err := utils.DownloadTFVars(envName, versionID); err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}

	fmt.Printf("\nSuccessfully downloaded to: %s\n", env.Local.TFVarsPath)
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  Review changes: cat %s\n", env.Local.TFVarsPath)
	fmt.Printf("  Plan changes:   tfvarenv plan %s\n", envName)
	fmt.Printf("  Apply changes:  tfvarenv apply %s\n", envName)

	return nil
}
