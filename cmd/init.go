package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"tfvarenv/config"

	"github.com/spf13/cobra"
)

func NewInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize the tfvarenv configuration",
		Run: func(cmd *cobra.Command, args []string) {
			if err := initializeProject(); err != nil {
				fmt.Printf("Error initializing project: %v\n", err)
				os.Exit(1)
			}
		},
	}
}

func initializeProject() error {
	// Check if config already exists
	if _, err := os.Stat(".tfvarenv.json"); err == nil {
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

	// Create config file
	configFile, err := os.Create(".tfvarenv.json")
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer configFile.Close()

	if err := config.SaveConfigToFile(configFile, &initialConfig); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Update .gitignore
	if err := updateGitignore(); err != nil {
		return fmt.Errorf("failed to update .gitignore: %w", err)
	}

	fmt.Println("\nInitialization completed successfully:")
	fmt.Printf("- Created .tfvarenv.yaml with default region: %s\n", region)
	fmt.Println("- Created environments directory")
	fmt.Println("- Updated .gitignore")
	fmt.Println("\nUse 'tfvarenv add' to add a new environment.")

	return nil
}
