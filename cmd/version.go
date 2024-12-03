package cmd

import (
    "fmt"
    "github.com/spf13/cobra"
)

var version = "dev"

func NewVersionCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "version",
        Short: "Print the version number of tfvarenv",
        Run: func(cmd *cobra.Command, args []string) {
            fmt.Printf("tfvarenv version %s\n", version)
        },
    }
}