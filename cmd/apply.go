package cmd

import (
	"fmt"
	"os"
	"tfvarenv/config"
	"tfvarenv/utils"

	"github.com/spf13/cobra"
)

func NewApplyCmd() *cobra.Command {
	var (
		remote    bool
		options   string
		versionID string
	)

	applyCmd := &cobra.Command{
		Use:   "apply [environment]",
		Short: "Run terraform apply for the specified environment",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			envName := args[0]
			env, err := config.GetEnvironmentInfo(envName)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}

			// Check if deployment requires approval
			if env.Deployment.RequireApproval {
				if !promptYesNo(fmt.Sprintf("Do you want to apply changes to %s environment?", envName), false) {
					fmt.Println("Deployment cancelled by user")
					os.Exit(1)
				}
			}

			if err := utils.RunTerraformCommand("apply", env, remote, versionID, options); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	applyCmd.Flags().BoolVar(&remote, "remote", false, "Use remote tfvars file from S3")
	applyCmd.Flags().StringVar(&options, "options", "", "Additional options for terraform apply")
	applyCmd.Flags().StringVar(&versionID, "version-id", "", "Specific version ID to use (only with --remote)")

	return applyCmd
}
