package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"tfvarenv/utils/command"
	"tfvarenv/utils/deployment"
	"tfvarenv/utils/terraform"
	"tfvarenv/utils/version"
)

func NewApplyCmd() *cobra.Command {
	utils, err := command.NewUtils()
	if err != nil {
		fmt.Printf("Error initializing command utils: %v\n", err)
		os.Exit(1)
	}

	var opts terraform.ApplyOptions

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

			if err := runApply(cmd.Context(), utils, &opts); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	applyCmd.Flags().BoolVar(&opts.Remote, "remote", false, "Use remote tfvars file from S3")
	applyCmd.Flags().StringVarP(&opts.VarFile, "var-file", "v", "", "Path to terraform.tfvars file")
	applyCmd.Flags().StringVar(&opts.VersionID, "version-id", "", "Specific version ID to use (only with --remote)")
	applyCmd.Flags().StringSliceVar(&opts.Options, "options", nil, "Additional options for terraform apply")
	applyCmd.Flags().BoolVar(&opts.AutoApprove, "auto-approve", false, "Skip interactive approval of plan")

	return applyCmd
}

func runApply(ctx context.Context, utils command.Utils, opts *terraform.ApplyOptions) error {
	// Get version information if using remote
	if opts.Remote {
		versionManager := version.NewManager(utils.GetAWSClient(), utils.GetFileUtils(), opts.Environment)
		var ver *version.Version
		var err error

		if opts.VersionID != "" {
			ver, err = versionManager.GetVersion(ctx, opts.VersionID)
		} else {
			ver, err = versionManager.GetLatestVersion(ctx)
		}
		if err != nil {
			return fmt.Errorf("failed to get version information: %w", err)
		}

		// Display version information
		fmt.Printf("\nApplying version:\n")
		fmt.Printf("  Version ID: %s\n", ver.VersionID[:8])
		fmt.Printf("  Uploaded: %s by %s\n",
			ver.Timestamp.Format("2006-01-02 15:04:05"),
			ver.UploadedBy)
		if ver.Description != "" {
			fmt.Printf("  Description: %s\n", ver.Description)
		}

		opts.VersionID = ver.VersionID
	}

	// Get deployment approval if required
	if opts.Environment.Deployment.RequireApproval && !opts.AutoApprove {
		if !promptYesNo(fmt.Sprintf("\nDo you want to proceed with applying to %s environment?",
			opts.Environment.Name), false) {
			return fmt.Errorf("deployment cancelled by user")
		}
	}

	// Run terraform apply
	_, err := utils.GetTerraformRunner().Apply(ctx, opts)
	if err != nil {
		return fmt.Errorf("terraform apply failed: %w", err)
	}

	// Record deployment if successful and using remote version
	if opts.Remote {
		deploymentManager := deployment.NewManager(utils.GetAWSClient(), opts.Environment)
		record := &deployment.Record{
			Timestamp:  time.Now(),
			VersionID:  opts.VersionID,
			DeployedBy: os.Getenv("USER"),
			Command:    "apply",
			Status:     "success",
		}

		if err := deploymentManager.AddRecord(ctx, record); err != nil {
			fmt.Printf("Warning: Failed to record deployment: %v\n", err)
		} else {
			fmt.Printf("\nDeployment recorded:\n")
			fmt.Printf("  Version: %s\n", opts.VersionID[:8])
			fmt.Printf("  Time: %s\n", record.Timestamp.Format("2006-01-02 15:04:05"))
			fmt.Printf("  By: %s\n", record.DeployedBy)
		}
	}

	return nil
}
