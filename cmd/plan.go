package cmd

import (
	"fmt"
	"os"
	"tfvarenv/config"
	"tfvarenv/utils"

	"github.com/spf13/cobra"
)

func NewPlanCmd() *cobra.Command {
	var (
		remote    bool
		options   string
		versionID string
	)

	planCmd := &cobra.Command{
		Use:   "plan [environment]",
		Short: "Run terraform plan for the specified environment",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			envName := args[0]
			env, err := config.GetEnvironmentInfo(envName)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}

			if err := utils.RunTerraformCommand("plan", env, remote, versionID, options); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	planCmd.Flags().BoolVar(&remote, "remote", false, "Use remote tfvars file from S3")
	planCmd.Flags().StringVar(&options, "options", "", "Additional options for terraform plan")
	planCmd.Flags().StringVar(&versionID, "version-id", "", "Specific version ID to use (only with --remote)")

	return planCmd
}
