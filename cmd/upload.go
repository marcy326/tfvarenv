package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"tfvarenv/config"
	"tfvarenv/utils"
	"time"

	"github.com/spf13/cobra"
)

func NewUploadCmd() *cobra.Command {
	var (
		description string
		autoBackup  bool
	)

	uploadCmd := &cobra.Command{
		Use:   "upload [environment]",
		Short: "Upload local tfvars file to S3",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			envName := args[0]
			if err := runUpload(envName, description, autoBackup); err != nil {
				fmt.Printf("Error uploading tfvars: %v\n", err)
				os.Exit(1)
			}
		},
	}

	uploadCmd.Flags().StringVarP(&description, "description", "d", "", "Description for this version")
	uploadCmd.Flags().BoolVar(&autoBackup, "auto-backup", true, "Create local backup before upload")

	return uploadCmd
}

func runUpload(envName string, description string, autoBackup bool) error {
	env, err := config.GetEnvironmentInfo(envName)
	if err != nil {
		return fmt.Errorf("environment not found: %w", err)
	}

	// Check if local file exists
	if _, err := os.Stat(env.Local.TFVarsPath); err != nil {
		return fmt.Errorf("local file not found: %w", err)
	}

	// Calculate file hash
	hash, err := utils.CalculateFileHash(env.Local.TFVarsPath)
	if err != nil {
		return fmt.Errorf("failed to calculate file hash: %w", err)
	}

	// Check for duplicate content
	versions, err := utils.GetVersions(env, &utils.VersionQueryOptions{
		LatestOnly: true,
	})
	if err == nil && len(versions) > 0 {
		latestVersion := versions[0]
		if latestVersion.Hash == hash {
			fmt.Println("Local file is identical to the latest version in S3. No upload needed.")
			fmt.Printf("Latest version: %s (uploaded at %s)\n",
				latestVersion.VersionID[:8], latestVersion.Timestamp.Format("2006-01-02 15:04:05"))
			return nil
		}
	}

	// Create backup if enabled
	if autoBackup {
		backupDir := filepath.Join(".backups", envName)
		if err := os.MkdirAll(backupDir, 0755); err != nil {
			return fmt.Errorf("failed to create backup directory: %w", err)
		}

		backupPath := filepath.Join(backupDir,
			fmt.Sprintf("terraform.tfvars.%s", time.Now().Format("20060102150405")))

		if err := utils.CopyFile(env.Local.TFVarsPath, backupPath); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
		fmt.Printf("Created backup: %s\n", backupPath)
	}

	// Prompt for description if not provided
	if description == "" {
		fmt.Print("Enter version description (optional): ")
		fmt.Scanln(&description)
	}

	// Get file size
	fileInfo, err := os.Stat(env.Local.TFVarsPath)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Upload to S3
	versionInfo, err := utils.UploadToS3WithVersioning(
		env.Local.TFVarsPath,
		env.GetS3Path(),
		env.AWS.Region,
		description,
	)
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	// Create version record
	version := &utils.Version{
		VersionID:   versionInfo.VersionID,
		Timestamp:   time.Now(),
		Hash:        hash,
		Description: description,
		UploadedBy:  os.Getenv("USER"),
		Size:        fileInfo.Size(),
	}

	// Add version to management
	if err := utils.AddVersion(env, version); err != nil {
		return fmt.Errorf("failed to record version: %w", err)
	}

	fmt.Printf("\nSuccessfully uploaded %s to %s\n", env.Local.TFVarsPath, env.GetS3Path())
	fmt.Printf("Version Information:\n")
	fmt.Printf("  Version ID: %s\n", version.VersionID[:8])
	fmt.Printf("  Timestamp: %s\n", version.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Size: %d bytes\n", version.Size)
	if version.Description != "" {
		fmt.Printf("  Description: %s\n", version.Description)
	}

	// Show deployment guidance
	fmt.Printf("\nTo deploy this version:\n")
	fmt.Printf("  Plan:   tfvarenv plan %s --remote --version-id %s\n", envName, version.VersionID)
	fmt.Printf("  Apply:  tfvarenv apply %s --remote --version-id %s\n", envName, version.VersionID)

	return nil
}
