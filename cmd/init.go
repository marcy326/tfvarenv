package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"tfvarenv/config"

	"github.com/spf13/cobra"
)

func NewInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize the tfvarenv configuration",
		Run: func(cmd *cobra.Command, args []string) {
			cwd, err := os.Getwd()
			if err != nil {
				fmt.Println("Error determining current directory:", err)
				os.Exit(1)
			}

			configPath := filepath.Join(cwd, ".tfvarenv.yaml")

			if _, err := os.Stat(configPath); err == nil {
				fmt.Println("Configuration already initialized.")
				return
			}

			file, err := os.Create(configPath)
			if err != nil {
				fmt.Println("Error creating configuration file:", err)
				os.Exit(1)
			}
			defer file.Close()

			defaultConfig := config.Config{
				Environments: []config.Environment{},
			}

			if err := config.SaveConfigToFile(file, &defaultConfig); err != nil {
				fmt.Println("Error writing default configuration:", err)
				os.Exit(1)
			}

			gitignorePath := filepath.Join(cwd, ".gitignore")
			addTfvarsToGitignore(gitignorePath)

			fmt.Println("Configuration initialized successfully.")
		},
	}
}

func addTfvarsToGitignore(gitignorePath string) {
	file, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Error opening or creating .gitignore file:", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	found := false
	for scanner.Scan() {
		if scanner.Text() == "*.tfvars" {
			found = true
			break
		}
	}

	if !found {
		if _, err := file.WriteString("\n*.tfvars\n"); err != nil {
			fmt.Println("Error writing to .gitignore file:", err)
		} else {
			fmt.Println("Added '*.tfvars' to .gitignore.")
		}
	}
}
