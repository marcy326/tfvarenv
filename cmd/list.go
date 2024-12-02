package cmd

import (
    "fmt"
    "os"
    "tfvarenv/config"
    "github.com/spf13/cobra"
)

func NewListCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "list",
        Short: "List all environments",
        Run: func(cmd *cobra.Command, args []string) {
            envs, err := config.ListEnvironments()
            if err != nil {
                fmt.Printf("Error listing environments: %s\n", err)
                os.Exit(1)
            }

            currentEnv := config.GetCurrentEnvironment()
            if err != nil {
                fmt.Printf("Error getting current environment: %s\n", err)
                os.Exit(1)
            }

            fmt.Println("Available environments:")
            for _, env := range envs {
                if env == currentEnv {
                    fmt.Println("* " + env)
                } else {
                    fmt.Println("  " + env)
                }
            }
        },
    }
}