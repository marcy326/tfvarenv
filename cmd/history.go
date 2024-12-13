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

func NewHistoryCmd() *cobra.Command {
	utils, err := command.NewUtils()
	if err != nil {
		fmt.Printf("Error initializing command utils: %v\n", err)
		os.Exit(1)
	}

	var (
		limit int
		since string
	)

	historyCmd := &cobra.Command{
		Use:   "history [environment]",
		Short: "Show deployment history for an environment",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			var sinceTime time.Time
			if since != "" {
				var err error
				sinceTime, err = time.Parse("2006-01-02", since)
				if err != nil {
					fmt.Printf("Error: Invalid date format for --since. Use YYYY-MM-DD\n")
					os.Exit(1)
				}
			}

			env, err := utils.GetEnvironment(args[0])
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}

			opts := &deployment.QueryOptions{
				Since: sinceTime,
				Limit: limit,
			}

			if err := runHistory(cmd.Context(), utils, env, opts); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	historyCmd.Flags().IntVar(&limit, "limit", 0, "Limit the number of entries")
	historyCmd.Flags().StringVar(&since, "since", "", "Show entries since date (YYYY-MM-DD)")

	return historyCmd
}

func runHistory(ctx context.Context, utils command.Utils, env *config.Environment, opts *deployment.QueryOptions) error {
	deploymentManager := deployment.NewManager(utils.GetAWSClient(), env)
	history, err := deploymentManager.GetHistory(ctx)
	if err != nil {
		return fmt.Errorf("failed to get deployment history: %w", err)
	}

	fmt.Printf("Deployment history for environment '%s':\n", env.Name)
	if len(history.Deployments) == 0 {
		fmt.Println("No deployment history found")
		return nil
	}

	// Get version information for additional context
	versionManager := version.NewManager(utils.GetAWSClient(), utils.GetFileUtils(), env)
	versionMap := make(map[string]*version.Version)
	versions, err := versionManager.GetVersions(ctx, nil)
	if err == nil {
		for i, v := range versions {
			versionMap[v.VersionID] = &versions[i]
		}
	}

	// Filter and display deployments
	deployments, err := deploymentManager.QueryDeployments(ctx, *opts)
	if err != nil {
		return fmt.Errorf("failed to query deployments: %w", err)
	}

	for _, deploy := range deployments {
		fmt.Printf("\n%s\n", deploy.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Command: terraform %s\n", deploy.Command)
		fmt.Printf("  Version: %s\n", deploy.VersionID[:8])
		fmt.Printf("  By: %s\n", deploy.DeployedBy)
		fmt.Printf("  Status: %s\n", deploy.Status)

		// Add version information if available
		if ver, ok := versionMap[deploy.VersionID]; ok {
			if ver.Description != "" {
				fmt.Printf("  Description: %s\n", ver.Description)
			}
		}

		if deploy.ErrorMessage != "" {
			fmt.Printf("  Error: %s\n", deploy.ErrorMessage)
		}
	}

	// Show summary statistics
	stats, err := deploymentManager.GetStats(ctx)
	if err == nil && stats.TotalDeployments > 1 {
		fmt.Printf("\nDeployment Statistics:\n")
		fmt.Printf("  Total Deployments: %d\n", stats.TotalDeployments)
		fmt.Printf("  Successful: %d\n", stats.SuccessfulCount)
		fmt.Printf("  Failed: %d\n", stats.FailedCount)
		if stats.AverageDuration > 0 {
			fmt.Printf("  Average Duration: %s\n", stats.AverageDuration)
		}
	}

	return nil
}
