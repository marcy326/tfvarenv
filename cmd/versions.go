package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"tfvarenv/config"
	"tfvarenv/utils/command"
	"tfvarenv/utils/deployment"
	"tfvarenv/utils/version"
)

func NewVersionsCmd() *cobra.Command {
	utils, err := command.NewUtils()
	if err != nil {
		fmt.Printf("Error initializing command utils: %v\n", err)
		os.Exit(1)
	}

	var opts version.QueryOptions

	versionsCmd := &cobra.Command{
		Use:   "versions [environment]",
		Short: "List available versions for an environment",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Parse since date if provided
			if sinceStr, _ := cmd.Flags().GetString("since"); sinceStr != "" {
				t, err := time.Parse("2006-01-02", sinceStr)
				if err != nil {
					fmt.Printf("Error: Invalid date format for --since. Use YYYY-MM-DD\n")
					os.Exit(1)
				}
				opts.Since = t
			}

			env, err := utils.GetEnvironment(args[0])
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}

			if err := runVersions(cmd.Context(), utils, env, &opts); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	versionsCmd.Flags().BoolVar(&opts.LatestOnly, "latest-only", false, "Show only the latest version")
	versionsCmd.Flags().IntVar(&opts.Limit, "limit", 0, "Limit the number of versions shown")
	versionsCmd.Flags().String("since", "", "Show versions since date (YYYY-MM-DD)")
	versionsCmd.Flags().StringVar(&opts.SearchText, "search", "", "Search in version descriptions")
	versionsCmd.Flags().BoolVar(&opts.SortByDate, "sort-by-date", true, "Sort versions by date")

	return versionsCmd
}

func runVersions(ctx context.Context, utils command.Utils, env *config.Environment, opts *version.QueryOptions) error {
	versionManager := version.NewManager(utils.GetAWSClient(), utils.GetFileUtils(), env)

	// Get versions
	versions, err := versionManager.GetVersions(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to get versions: %w", err)
	}

	// Get deployment history for status information
	deploymentManager := deployment.NewManager(utils.GetAWSClient(), env)
	deployments, err := deploymentManager.GetHistory(ctx)
	if err != nil {
		fmt.Printf("Warning: Failed to get deployment history: %v\n", err)
	}

	fmt.Printf("Available versions for environment '%s':\n", env.Name)
	if len(versions) == 0 {
		fmt.Println("No versions found")
		return nil
	}

	// Create deployment lookup map
	deploymentMap := make(map[string]*deployment.Record)
	if deployments != nil {
		for i, d := range deployments.Deployments {
			if _, exists := deploymentMap[d.VersionID]; !exists {
				deploymentMap[d.VersionID] = &deployments.Deployments[i]
			}
		}
	}

	// Display version information
	displayCount := 0
	for i, v := range versions {
		// Display version header
		latestTag := ""
		if i == 0 {
			latestTag = " (Latest)"
		}
		fmt.Printf("\nVersion: %s%s\n", v.VersionID[:8], latestTag)
		fmt.Printf("  Uploaded: %s\n", v.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("  By: %s\n", v.UploadedBy)
		fmt.Printf("  Size: %d bytes\n", v.Size)
		if v.Description != "" {
			fmt.Printf("  Description: %s\n", v.Description)
		}

		// Show deployment status if available
		if deploy, isDeployed := deploymentMap[v.VersionID]; isDeployed {
			fmt.Printf("  Last Deployed: %s by %s\n",
				deploy.Timestamp.Format("2006-01-02 15:04:05"),
				deploy.DeployedBy)
		} else {
			fmt.Printf("  Status: Not deployed\n")
		}

		displayCount++
		if opts.Limit > 0 && displayCount >= opts.Limit {
			break
		}
	}

	// Show statistics
	if displayCount > 1 {
		stats, err := versionManager.GetStats(ctx)
		if err == nil {
			fmt.Printf("\nVersion Statistics:\n")
			fmt.Printf("  Total Versions: %d\n", stats.TotalVersions)
			fmt.Printf("  Average Size: %d bytes\n", stats.AverageSize)
			fmt.Printf("  Most Active User: %s\n", stats.MostActiveUser)
			fmt.Printf("  Last Updated: %s\n", stats.LastUpdated.Format("2006-01-02 15:04:05"))
		}
	}

	return nil
}
