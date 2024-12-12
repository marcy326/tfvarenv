package cmd

import (
	"fmt"
	"os"
	"tfvarenv/config"
	"tfvarenv/utils"
	"time"

	"github.com/spf13/cobra"
)

func NewHistoryCmd() *cobra.Command {
	var (
		limit int
		since string
	)

	historyCmd := &cobra.Command{
		Use:   "history [environment]",
		Short: "Show deployment history for an environment",
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

			if err := showHistory(envName, limit, sinceTime); err != nil {
				fmt.Printf("Error showing history: %v\n", err)
				os.Exit(1)
			}
		},
	}

	historyCmd.Flags().IntVar(&limit, "limit", 0, "Limit the number of entries")
	historyCmd.Flags().StringVar(&since, "since", "", "Show entries since date (YYYY-MM-DD)")

	return historyCmd
}

func showHistory(envName string, limit int, since time.Time) error {
	env, err := config.GetEnvironmentInfo(envName)
	if err != nil {
		return fmt.Errorf("environment not found: %w", err)
	}

	// Get deployment history
	history, err := utils.GetDeploymentHistory(env)
	if err != nil {
		return fmt.Errorf("failed to get deployment history: %w", err)
	}

	fmt.Printf("Deployment history for environment '%s':\n", envName)
	if len(history.Deployments) == 0 {
		fmt.Println("No deployment history found")
		return nil
	}

	// Get version information for additional context
	versions, err := utils.GetVersions(env, nil)
	versionMap := make(map[string]*utils.Version)
	if err == nil {
		for i, v := range versions {
			versionMap[v.VersionID] = &versions[i]
		}
	}

	displayCount := 0
	for _, deploy := range history.Deployments {
		// Apply time filter if specified
		if !since.IsZero() && deploy.Timestamp.Before(since) {
			continue
		}

		fmt.Printf("\n%s\n", deploy.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Command: terraform %s\n", deploy.Command)
		fmt.Printf("  Version: %s\n", deploy.VersionID[:8])
		fmt.Printf("  By: %s\n", deploy.DeployedBy)
		fmt.Printf("  Status: %s\n", deploy.Status)

		// Add version information if available
		if version, ok := versionMap[deploy.VersionID]; ok {
			if version.Description != "" {
				fmt.Printf("  Description: %s\n", version.Description)
			}
		}

		displayCount++
		if limit > 0 && displayCount >= limit {
			break
		}
	}

	// Show summary if there are multiple deployments
	if displayCount > 1 {
		fmt.Printf("\nTotal deployments shown: %d\n", displayCount)
	}

	return nil
}
