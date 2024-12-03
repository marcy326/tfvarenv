package cmd

import (
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
                CurrentEnv:   "",
            }

            if err := config.SaveConfigToFile(file, &defaultConfig); err != nil {
                fmt.Println("Error writing default configuration:", err)
                os.Exit(1)
            }

            fmt.Println("Configuration initialized successfully.")
        },
    }
}