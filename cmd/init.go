package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"tfvarenv/config"
	"tfvarenv/utils/command"
	"tfvarenv/utils/file"
)

func NewInitCmd() *cobra.Command {
	utils, err := command.NewUtils()
	if err != nil {
		fmt.Printf("Error initializing command utils: %v\n", err)
		os.Exit(1)
	}

	return &cobra.Command{
		Use:   "init",
		Short: "Initialize the tfvarenv configuration",
		Run: func(cmd *cobra.Command, args []string) {
			if err := runInit(cmd.Context(), utils); err != nil {
				fmt.Printf("Error initializing project: %v\n", err)
				os.Exit(1)
			}
		},
	}
}

func runInit(ctx context.Context, utils command.Utils) error {
	fileUtils := utils.GetFileUtils()

	// Check if config already exists
	exists, err := fileUtils.FileExists(".tfvarenv.json")
	if err != nil {
		return fmt.Errorf("failed to check config file: %w", err)
	}
	if exists {
		return fmt.Errorf("configuration file already exists")
	}

	reader := bufio.NewReader(os.Stdin)

	// Get default region from user
	fmt.Print("Enter default AWS region [us-east-1]: ")
	region, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	region = strings.TrimSpace(region)
	if region == "" {
		region = "us-east-1"
	}

	// Create initial configuration
	initialConfig := config.Config{
		Version:       "1.0",
		DefaultRegion: region,
		Environments:  make(map[string]config.Environment),
	}

	// Convert config to JSON
	configData, err := json.MarshalIndent(initialConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write config file
	writeOpts := &file.Options{
		CreateDirs: true,
		Overwrite:  false,
	}
	if err := fileUtils.WriteFile(".tfvarenv.json", configData, writeOpts); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Create basic directory structure
	directories := []string{
		"envs",
		".backups",
		".tmp",
	}
	for _, dir := range directories {
		if err := fileUtils.EnsureDirectory(dir); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Update .gitignore
	gitignoreEntries := []string{
		"*.tfvars",
		".terraform/",
		".terraform.lock.hcl",
		".tmp/",
		".backups/",
	}

	var gitignoreContent string
	existingContent, err := fileUtils.ReadFile(".gitignore")
	if err == nil {
		gitignoreContent = string(existingContent)
		if !strings.HasSuffix(gitignoreContent, "\n") {
			gitignoreContent += "\n"
		}
	}

	// Add new entries
	addedEntries := false
	existing := make(map[string]bool)
	for _, line := range strings.Split(gitignoreContent, "\n") {
		existing[strings.TrimSpace(line)] = true
	}

	for _, entry := range gitignoreEntries {
		if !existing[entry] {
			if !addedEntries {
				gitignoreContent += "\n# tfvarenv\n"
				addedEntries = true
			}
			gitignoreContent += entry + "\n"
		}
	}

	if addedEntries {
		if err := fileUtils.WriteFile(".gitignore", []byte(gitignoreContent), writeOpts); err != nil {
			return fmt.Errorf("failed to update .gitignore: %w", err)
		}
	}

	fmt.Println("\nInitialization completed successfully:")
	fmt.Printf("- Created .tfvarenv.json with default region: %s\n", region)
	fmt.Println("- Created directory structure:")
	fmt.Println("  - envs/ (for environment-specific files)")
	fmt.Println("  - .backups/ (for local backups)")
	fmt.Println("  - .tmp/ (for temporary files)")
	fmt.Println("- Updated .gitignore")

	fmt.Println("\nNext steps:")
	fmt.Println("1. Use 'tfvarenv add' to add your first environment")
	fmt.Println("2. Configure AWS credentials if not already set")
	fmt.Println("3. Create your Terraform configuration")

	return nil
}
