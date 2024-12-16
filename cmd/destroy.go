package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"tfvarenv/utils/command"
	"tfvarenv/utils/destroy"
)

func NewDestroyCmd() *cobra.Command {
	utils, err := command.NewUtils()
	if err != nil {
		fmt.Printf("Error initializing command utils: %v\n", err)
		os.Exit(1)
	}

	var opts destroy.Options

	destroyCmd := &cobra.Command{
		Use:   "destroy [environment]",
		Short: "Destroy all resources in the specified environment",
		Long: `Destroy all resources in the specified environment using the last deployed version of tfvars.
This command will permanently delete all resources managed by Terraform.`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			opts.Environment, err = utils.GetEnvironment(args[0])
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}

			manager := destroy.NewManager(
				utils.GetAWSClient(),
				utils.GetFileUtils(),
				utils.GetTerraformRunner(),
			)

			if err := manager.Execute(cmd.Context(), &opts); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	destroyCmd.Flags().StringVar(&opts.VersionID, "version-id", "", "Specific version ID to use (defaults to last deployed version)")
	destroyCmd.Flags().BoolVar(&opts.AutoApprove, "auto-approve", false, "Skip interactive approval")
	destroyCmd.Flags().StringSliceVar(&opts.TerraformOpts, "options", nil, "Additional options for terraform destroy")

	return destroyCmd
}
