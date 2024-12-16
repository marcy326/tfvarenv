package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"tfvarenv/config"
	"tfvarenv/utils/command"
	"tfvarenv/utils/file"
	"tfvarenv/utils/prompt"
)

func NewRemoveCmd() *cobra.Command {
	utils, err := command.NewUtils()
	if err != nil {
		fmt.Printf("Error initializing command utils: %v\n", err)
		os.Exit(1)
	}

	var force bool

	removeCmd := &cobra.Command{
		Use:   "remove [environment]",
		Short: "Remove an environment",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := runRemove(cmd.Context(), utils, args[0], force); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	removeCmd.Flags().BoolVarP(&force, "force", "f", false, "Force removal without confirmation")

	return removeCmd
}

func runRemove(ctx context.Context, utils command.Utils, envName string, force bool) error {
	// Get environment configuration
	env, err := utils.GetEnvironment(envName)
	if err != nil {
		return fmt.Errorf("environment not found: %w", err)
	}

	// Show environment details before removal
	fmt.Printf("\nEnvironment to remove: %s\n", envName)
	if env.Description != "" {
		fmt.Printf("Description: %s\n", env.Description)
	}
	fmt.Printf("AWS Account: %s (%s)\n", env.AWS.AccountID, env.AWS.Region)
	fmt.Printf("S3 Path: %s\n", env.GetS3Path())
	fmt.Printf("Local Path: %s\n", env.Local.TFVarsPath)

	// Confirm removal
	if !force {
		if !prompt.PromptYesNo(fmt.Sprintf("\nAre you sure you want to remove environment '%s'?", envName), false) {
			return fmt.Errorf("removal cancelled by user")
		}
	}

	// Create backup of local files if they exist
	fileUtils := utils.GetFileUtils()
	localExists, err := fileUtils.FileExists(env.Local.TFVarsPath)
	if err != nil {
		return fmt.Errorf("failed to check local file: %w", err)
	}

	if localExists {
		backupOpts := &file.BackupOptions{
			BasePath:   filepath.Join(".backups", envName),
			TimeFormat: "20060102150405",
		}
		if backupPath, err := fileUtils.CreateBackup(env.Local.TFVarsPath, backupOpts); err != nil {
			fmt.Printf("Warning: Failed to create backup: %v\n", err)
		} else {
			fmt.Printf("Created backup: %s\n", backupPath)
		}
	}

	// Remove from configuration
	if err := removeEnvironment(utils, envName); err != nil {
		return fmt.Errorf("failed to remove environment from configuration: %w", err)
	}

	fmt.Printf("\nSuccessfully removed environment '%s'\n", envName)
	return nil
}

func removeEnvironment(utils command.Utils, name string) error {
	cfg, err := config.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize config manager: %w", err)
	}

	if err := cfg.RemoveEnvironment(name); err != nil {
		return fmt.Errorf("failed to remove environment: %w", err)
	}

	return nil
}
