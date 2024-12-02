package cmd

import (
    "github.com/spf13/cobra"
)

// NewRootCmd creates the root command for tfvarenv CLI.
func NewRootCmd() *cobra.Command {
    rootCmd := &cobra.Command{
        Use:   "tfvarenv",
        Short: "Manage Terraform environments and tfvars files.",
        Long:  `tfvarenv simplifies the management of Terraform environments and tfvars files.`,
    }

    // Add individual commands
    rootCmd.AddCommand(NewAddCmd())
    rootCmd.AddCommand(NewListCmd())
    rootCmd.AddCommand(NewUseCmd())
    rootCmd.AddCommand(NewInitCmd())

    return rootCmd
}