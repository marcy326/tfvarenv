package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"tfvarenv/utils/apply"
	"tfvarenv/utils/command"
)

func NewApplyCmd() *cobra.Command {
	utils, err := command.NewUtils()
	if err != nil {
		fmt.Printf("Error initializing command utils: %v\n", err)
		os.Exit(1)
	}

	var opts apply.Options

	applyCmd := &cobra.Command{
		Use:   "apply [environment]",
		Short: "Run terraform apply for the specified environment",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			opts.Environment, err = utils.GetEnvironment(args[0])
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}

			manager := apply.NewManager(
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

	applyCmd.Flags().BoolVar(&opts.Remote, "remote", false, "Use remote tfvars file from S3")
	applyCmd.Flags().StringVarP(&opts.VarFile, "var-file", "v", "", "Path to terraform.tfvars file")
	applyCmd.Flags().StringVar(&opts.VersionID, "version-id", "", "Specific version ID to use (only with --remote)")
	applyCmd.Flags().StringSliceVar(&opts.TerraformOpts, "options", nil, "Additional options for terraform apply")
	applyCmd.Flags().BoolVar(&opts.AutoApprove, "auto-approve", false, "Skip interactive approval of plan")

	return applyCmd
}
