package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"tfvarenv/utils/command"
	"tfvarenv/utils/deployment"
	"tfvarenv/utils/terraform"
	"tfvarenv/utils/version"
)

func NewPlanCmd() *cobra.Command {
	utils, err := command.NewUtils()
	if err != nil {
		fmt.Printf("Error initializing command utils: %v\n", err)
		os.Exit(1)
	}

	var opts terraform.PlanOptions

	planCmd := &cobra.Command{
		Use:   "plan [environment]",
		Short: "Run terraform plan for the specified environment",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			opts.Environment, err = utils.GetEnvironment(args[0])
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}

			if err := runPlan(cmd.Context(), utils, &opts); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	planCmd.Flags().BoolVar(&opts.Remote, "remote", false, "Use remote tfvars file from S3")
	planCmd.Flags().StringVarP(&opts.VarFile, "var-file", "v", "", "Path to terraform.tfvars file")
	planCmd.Flags().StringVar(&opts.VersionID, "version-id", "", "Specific version ID to use (only with --remote)")
	planCmd.Flags().StringSliceVar(&opts.Options, "options", nil, "Additional options for terraform plan")

	return planCmd
}

func runPlan(ctx context.Context, utils command.Utils, opts *terraform.PlanOptions) error {
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
		fmt.Printf("\nPlanning with version:\n")
		fmt.Printf("  Version ID: %s\n", ver.VersionID[:8])
		fmt.Printf("  Uploaded: %s\n", ver.Timestamp.Format("2006-01-02 15:04:05"))
		if ver.Description != "" {
			fmt.Printf("  Description: %s\n", ver.Description)
		}

		// Get deployment status
		deploymentManager := deployment.NewManager(utils.GetAWSClient(), opts.Environment)
		if latestDeploy, err := deploymentManager.GetLatestDeployment(ctx); err == nil && latestDeploy != nil {
			if latestDeploy.VersionID == ver.VersionID {
				fmt.Printf("  Deployment Status: Currently deployed (since %s)\n",
					latestDeploy.Timestamp.Format("2006-01-02 15:04:05"))
			} else {
				fmt.Printf("  Deployment Status: Not currently deployed (latest deployment is version %s)\n",
					latestDeploy.VersionID[:8])
			}
		}

		// Check if using older version
		if latestVer, err := versionManager.GetLatestVersion(ctx); err == nil &&
			latestVer.VersionID != ver.VersionID {
			fmt.Printf("\nWarning: You are planning with version %s, but a newer version exists:\n",
				ver.VersionID[:8])
			fmt.Printf("  Latest Version: %s\n", latestVer.VersionID[:8])
			fmt.Printf("  Uploaded: %s by %s\n",
				latestVer.Timestamp.Format("2006-01-02 15:04:05"),
				latestVer.UploadedBy)
			if latestVer.Description != "" {
				fmt.Printf("  Description: %s\n", latestVer.Description)
			}
		}

		fmt.Println() // Add empty line for readability
		opts.VersionID = ver.VersionID
	} else {
		// ローカルファイルのパスを自動設定
		if opts.VarFile == "" { // --var-fileで明示的に指定されていない場合
			opts.VarFile = opts.Environment.Local.TFVarsPath

			// ファイルの存在確認
			exists, err := utils.GetFileUtils().FileExists(opts.VarFile)
			if err != nil {
				return fmt.Errorf("failed to check tfvars file: %w", err)
			}
			if !exists {
				return fmt.Errorf("tfvars file not found at: %s", opts.VarFile)
			}

			fmt.Printf("Using local tfvars file: %s\n", opts.VarFile)
		}
	}

	// Run terraform plan
	_, err := utils.GetTerraformRunner().Plan(ctx, opts)
	if err != nil {
		return fmt.Errorf("terraform plan failed: %w", err)
	}

	// Show next steps if using remote version
	if opts.Remote {
		fmt.Printf("\nTo apply this plan with the same version:\n")
		fmt.Printf("  tfvarenv apply %s --remote --version-id %s\n",
			opts.Environment.Name, opts.VersionID)
	}

	return nil
}
