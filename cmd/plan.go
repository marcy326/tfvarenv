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
			if err := runPlan(envName, remote, versionID, options); err != nil {
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

func runPlan(envName string, remote bool, versionID, options string) error {
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
		fmt.Printf("\nPlanning with version:\n")
		fmt.Printf("  Version ID: %s\n", version.VersionID[:8])
		fmt.Printf("  Uploaded: %s by %s\n", version.Timestamp.Format("2006-01-02 15:04:05"), version.UploadedBy)
		if version.Description != "" {
			fmt.Printf("  Description: %s\n", version.Description)
		}

		// Get and display deployment status
		if latestDeploy, err := utils.GetLatestDeployment(env); err == nil && latestDeploy != nil {
			if latestDeploy.VersionID == version.VersionID {
				fmt.Printf("  Deployment Status: Currently deployed (since %s)\n",
					latestDeploy.Timestamp.Format("2006-01-02 15:04:05"))
			} else {
				fmt.Printf("  Deployment Status: Not currently deployed (latest deployment is version %s)\n",
					latestDeploy.VersionID[:8])
			}
		}

		// Check if using older version
		versions, err := utils.GetVersions(env, &utils.VersionQueryOptions{
			LatestOnly: true,
		})
		if err == nil && len(versions) > 0 && versions[0].VersionID != version.VersionID {
			fmt.Printf("\nWarning: You are planning with version %s, but a newer version exists:\n", version.VersionID[:8])
			fmt.Printf("  Latest Version: %s\n", versions[0].VersionID[:8])
			fmt.Printf("  Uploaded: %s by %s\n",
				versions[0].Timestamp.Format("2006-01-02 15:04:05"),
				versions[0].UploadedBy)
			if versions[0].Description != "" {
				fmt.Printf("  Description: %s\n", versions[0].Description)
			}
		}

		fmt.Println() // Add empty line for readability
	}

	// Run terraform plan
	if err := utils.RunTerraformCommand("plan", env, remote, versionID, options); err != nil {
		return fmt.Errorf("terraform plan failed: %w", err)
	}

	// Show next steps
	if remote {
		fmt.Printf("\nTo apply this plan with the same version:\n")
		fmt.Printf("  tfvarenv apply %s --remote --version-id %s\n", envName, versionID)
	}

	return nil
}
