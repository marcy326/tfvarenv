package cmd

import (
	"fmt"
	"os"
	"tfvarenv/config"
	"tfvarenv/utils"
	"time"

	"github.com/spf13/cobra"
)

func NewVersionsCmd() *cobra.Command {
	var (
		deployedOnly bool
		latestOnly   bool
		since        string
		limit        int
		searchText   string
	)

	versionsCmd := &cobra.Command{
		Use:   "versions [environment]",
		Short: "List available versions for an environment",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			envName := args[0]

			// Parse since date if provided
			var sinceTime time.Time
			if since != "" {
				var err error
				sinceTime, err = time.Parse("2006-01-02", since)
				if err != nil {
					fmt.Printf("Error: Invalid date format for --since. Use YYYY-MM-DD\n")
					os.Exit(1)
				}
			}

			options := &utils.VersionQueryOptions{
				LatestOnly: latestOnly,
				Since:      sinceTime,
				Limit:      limit,
				SearchText: searchText,
				SortByDate: true,
			}

			if err := listVersions(envName, options, deployedOnly); err != nil {
				fmt.Printf("Error listing versions: %v\n", err)
				os.Exit(1)
			}
		},
	}

	versionsCmd.Flags().BoolVar(&deployedOnly, "deployed-only", false, "Show only deployed versions")
	versionsCmd.Flags().BoolVar(&latestOnly, "latest-only", false, "Show only the latest version")
	versionsCmd.Flags().StringVar(&since, "since", "", "Show versions since date (YYYY-MM-DD)")
	versionsCmd.Flags().IntVar(&limit, "limit", 0, "Limit the number of versions shown")
	versionsCmd.Flags().StringVar(&searchText, "search", "", "Search in version descriptions")

	return versionsCmd
}

func listVersions(envName string, options *utils.VersionQueryOptions, deployedOnly bool) error {
	env, err := config.GetEnvironmentInfo(envName)
	if err != nil {
		return fmt.Errorf("environment not found: %w", err)
	}

	// Get versions
	versions, err := utils.GetVersions(env, options)
	if err != nil {
		return fmt.Errorf("failed to get versions: %w", err)
	}

	// Get deployment history if needed
	var deployedVersions map[string]*utils.DeploymentRecord
	if deployedOnly {
		history, err := utils.GetDeploymentHistory(env)
		if err != nil {
			return fmt.Errorf("failed to get deployment history: %w", err)
		}

		deployedVersions = make(map[string]*utils.DeploymentRecord)
		for i, d := range history.Deployments {
			if _, exists := deployedVersions[d.VersionID]; !exists {
				deployedVersions[d.VersionID] = &history.Deployments[i]
			}
		}
	}

	fmt.Printf("Available versions for environment '%s':\n", envName)
	if len(versions) == 0 {
		fmt.Println("No versions found")
		return nil
	}

	displayCount := 0
	for i, v := range versions {
		// Skip if filtering for deployed versions only
		if deployedOnly {
			if _, isDeployed := deployedVersions[v.VersionID]; !isDeployed {
				continue
			}
		}

		// Display version header
		latestTag := ""
		if i == 0 && !deployedOnly {
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
		if deploy, isDeployed := deployedVersions[v.VersionID]; isDeployed {
			fmt.Printf("  Last Deployed: %s by %s\n",
				deploy.Timestamp.Format("2006-01-02 15:04:05"),
				deploy.DeployedBy)
		} else if !deployedOnly {
			fmt.Printf("  Status: Not deployed\n")
		}

		displayCount++
		if options.Limit > 0 && displayCount >= options.Limit {
			break
		}
	}

	// Show summary if multiple versions
	if displayCount > 1 {
		fmt.Printf("\nTotal versions shown: %d\n", displayCount)
	}

	return nil
}
