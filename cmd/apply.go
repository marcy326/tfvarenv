package cmd

import (
	"fmt"
	"os"
	"tfvarenv/config"
	"tfvarenv/utils"
	"time"

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
			if err := runApply(envName, remote, versionID, options); err != nil {
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

func runApply(envName string, remote bool, versionID, options string) error {
	env, err := config.GetEnvironmentInfo(envName)
	if err != nil {
		return fmt.Errorf("failed to get environment info: %w", err)
	}

	if remote {
		// Get version information
		var version *utils.Version
		if versionID != "" {
			versions, err := utils.GetVersions(env, &utils.VersionQueryOptions{})
			if err != nil {
				return fmt.Errorf("failed to get versions: %w", err)
			}
			for _, v := range versions {
				if v.VersionID == versionID {
					version = &v
					break
				}
			}
			if version == nil {
				return fmt.Errorf("version ID %s not found", versionID)
			}
		} else {
			versions, err := utils.GetVersions(env, &utils.VersionQueryOptions{
				LatestOnly: true,
			})
			if err != nil {
				return fmt.Errorf("failed to get versions: %w", err)
			}
			if len(versions) == 0 {
				return fmt.Errorf("no versions found")
			}
			version = &versions[0]
			versionID = version.VersionID
		}

		// Display version information
		fmt.Printf("\nVersion to be applied:\n")
		fmt.Printf("  Version ID: %s\n", version.VersionID[:8])
		fmt.Printf("  Uploaded: %s by %s\n", version.Timestamp.Format("2006-01-02 15:04:05"), version.UploadedBy)
		if version.Description != "" {
			fmt.Printf("  Description: %s\n", version.Description)
		}

		// Get and display latest deployment if exists
		if latestDeploy, err := utils.GetLatestDeployment(env); err == nil && latestDeploy != nil {
			fmt.Printf("  Last Deployed: %s by %s\n",
				latestDeploy.Timestamp.Format("2006-01-02 15:04:05"),
				latestDeploy.DeployedBy)
		}
	}

	// Get deployment approval
	if env.Deployment.RequireApproval {
		if !promptYesNo(fmt.Sprintf("\nDo you want to proceed with applying to %s environment?", envName), false) {
			return fmt.Errorf("deployment cancelled by user")
		}
	}

	// Run terraform apply
	if err := utils.RunTerraformCommand("apply", env, remote, versionID, options); err != nil {
		return fmt.Errorf("terraform apply failed: %w", err)
	}

	// Record deployment
	if remote {
		deployRecord := &utils.DeploymentRecord{
			Timestamp:  time.Now(),
			VersionID:  versionID,
			DeployedBy: os.Getenv("USER"),
			Command:    "apply",
			Status:     "success",
		}

		if err := utils.AddDeploymentRecord(env, deployRecord); err != nil {
			fmt.Printf("Warning: Failed to record deployment: %v\n", err)
		} else {
			fmt.Printf("\nDeployment recorded:\n")
			fmt.Printf("  Version: %s\n", versionID[:8])
			fmt.Printf("  Time: %s\n", deployRecord.Timestamp.Format("2006-01-02 15:04:05"))
			fmt.Printf("  By: %s\n", deployRecord.DeployedBy)
		}
	}

	return nil
}
