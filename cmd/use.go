package cmd

import (
    "fmt"
    "os"
    "tfvarenv/config"
    "github.com/spf13/cobra"
)

func NewUseCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "use [environment_name]",
        Short: "Switch to a specific environment",
        Args:  cobra.ExactArgs(1),
        Run: func(cmd *cobra.Command, args []string) {
            envName := args[0]

            err := config.UseEnvironment(envName)
            if err != nil {
                fmt.Printf("Error switching to environment: %s\n", err)
                os.Exit(1)
            }
            fmt.Printf("Switched to environment '%s'.\n", envName)
        },
    }
}