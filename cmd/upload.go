package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"tfvarenv/utils/aws"
	"tfvarenv/utils/command"
	"tfvarenv/utils/file"
	"tfvarenv/utils/version"
)

func NewUploadCmd() *cobra.Command {
	utils, err := command.NewUtils()
	if err != nil {
		fmt.Printf("Error initializing command utils: %v\n", err)
		os.Exit(1)
	}

	var (
		description string
		autoBackup  bool
	)

	uploadCmd := &cobra.Command{
		Use:   "upload [environment]",
		Short: "Upload local tfvars file to S3",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := runUpload(cmd.Context(), utils, args[0], description, autoBackup); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	uploadCmd.Flags().StringVarP(&description, "description", "d", "", "Description for this version")
	uploadCmd.Flags().BoolVar(&autoBackup, "auto-backup", true, "Create local backup before upload")

	return uploadCmd
}

func runUpload(ctx context.Context, utils command.Utils, envName, description string, autoBackup bool) error {
	env, err := utils.GetEnvironment(envName)
	if err != nil {
		return fmt.Errorf("environment not found: %w", err)
	}

	fileUtils := utils.GetFileUtils()

	exists, err := fileUtils.FileExists(env.Local.TFVarsPath)
	if err != nil {
		return fmt.Errorf("failed to check local file: %w", err)
	}
	if !exists {
		return fmt.Errorf("local file not found: %s", env.Local.TFVarsPath)
	}

	fileInfo, err := fileUtils.GetFileInfo(env.Local.TFVarsPath)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	versionManager := version.NewManager(utils.GetAWSClient(), fileUtils, env)
	latestVer, _ := versionManager.GetLatestVersion(ctx)
	if latestVer != nil && latestVer.Hash == fileInfo.Hash {
		fmt.Println("Local file is identical to the latest version in S3. No upload needed.")
		fmt.Printf("Latest version: %s (uploaded at %s)\n",
			latestVer.VersionID[:8], latestVer.Timestamp.Format("2006-01-02 15:04:05"))
		return nil
	}

	if autoBackup {
		backupOpts := &file.BackupOptions{
			BasePath:   filepath.Join(".backups", envName),
			TimeFormat: "20060102150405",
		}
		backupPath, err := fileUtils.CreateBackup(env.Local.TFVarsPath, backupOpts)
		if err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
		fmt.Printf("Created backup: %s\n", backupPath)
	}

	content, err := fileUtils.ReadFile(env.Local.TFVarsPath)
	if err != nil {
		return fmt.Errorf("failed to read local file: %w", err)
	}

	uploadInput := &aws.UploadInput{
		Bucket:      env.S3.Bucket,
		Key:         env.GetS3Path(),
		Content:     content,
		Description: description,
		Metadata: map[string]string{
			"Hash":        fileInfo.Hash,
			"Description": description,
			"UploadedBy":  os.Getenv("USER"),
		},
	}

	uploadOutput, err := utils.GetAWSClient().UploadFile(ctx, uploadInput)
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	newVersion := &version.Version{
		VersionID:   uploadOutput.VersionID,
		Hash:        fileInfo.Hash,
		Timestamp:   time.Now(),
		Description: description,
		UploadedBy:  os.Getenv("USER"),
		Size:        fileInfo.Size,
	}

	if err := versionManager.AddVersion(ctx, newVersion); err != nil {
		fmt.Printf("Warning: Failed to record version information: %v\n", err)
		fmt.Println("The file was uploaded successfully, but version tracking may be incomplete.")
		fmt.Printf("Please run 'tfvarenv upload %s' again to ensure proper version tracking.\n", envName)
		return nil
	}

	fmt.Printf("\nSuccessfully uploaded %s to %s\n", env.Local.TFVarsPath, env.GetFullS3Path())
	fmt.Printf("Version Information:\n")
	fmt.Printf("  Version ID: %s\n", newVersion.VersionID[:8])
	fmt.Printf("  Timestamp: %s\n", newVersion.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Size: %d bytes\n", newVersion.Size)
	if newVersion.Description != "" {
		fmt.Printf("  Description: %s\n", newVersion.Description)
	}

	return nil
}
