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
	if err := updateGitignore(fileUtils); err != nil {
		return fmt.Errorf("failed to update .gitignore: %w", err)
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

func updateGitignore(fileUtils file.Utils) error {
	entries := []string{
		"*.tfvars",
		".terraform/",
		".tmp/",
		".backups/",
	}

	content, err := fileUtils.ReadFile(".gitignore")
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read .gitignore: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	existing := make(map[string]bool)
	for _, line := range strings.Split(string(content), "\n") {
		existing[strings.TrimSpace(line)] = true
	}

	var newContent []string
	newContent = append(newContent, lines...)

	added := false
	for _, entry := range entries {
		if !existing[entry] {
			if !added {
				if len(newContent) > 0 && newContent[len(newContent)-1] != "" {
					newContent = append(newContent, "")
				}
				newContent = append(newContent, "# tfvarenv")
				added = true
			}
			newContent = append(newContent, entry)
		}
	}

	if added {
		opts := &file.Options{
			CreateDirs: true,
			Overwrite:  true,
		}
		return fileUtils.WriteFile(".gitignore", []byte(strings.Join(newContent, "\n")+"\n"), opts)
	}

	return nil
}
