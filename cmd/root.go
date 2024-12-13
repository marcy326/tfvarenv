package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "tfvarenv",
		Short: "Manage Terraform environments and tfvars files",
		Long: `tfvarenv simplifies the management of Terraform environments and tfvars files.
It provides version control for tfvars files and helps manage multiple environments.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Setup context with cancellation
			ctx, cancel := context.WithCancel(context.Background())

			// Handle interrupts
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt)
			go func() {
				<-sigChan
				fmt.Println("\nReceived interrupt signal. Cleaning up...")
				cancel()
			}()

			cmd.SetContext(ctx)
		},
	}

	// Add individual commands
	rootCmd.AddCommand(NewAddCmd())
	rootCmd.AddCommand(NewApplyCmd())
	rootCmd.AddCommand(NewDownloadCmd())
	rootCmd.AddCommand(NewHistoryCmd())
	rootCmd.AddCommand(NewInitCmd())
	rootCmd.AddCommand(NewListCmd())
	rootCmd.AddCommand(NewPlanCmd())
	rootCmd.AddCommand(NewUploadCmd())
	rootCmd.AddCommand(NewUseCmd())
	rootCmd.AddCommand(NewVersionCmd())
	rootCmd.AddCommand(NewVersionsCmd())

	return rootCmd
}
