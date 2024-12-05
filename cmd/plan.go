package cmd

import (
	"fmt"
	"os"
	"tfvarenv/config"
	"tfvarenv/utils"

	"github.com/spf13/cobra"
)

func NewPlanCmd() *cobra.Command {
	planCmd := &cobra.Command{
		Use:   "plan",
		Short: "Run terraform plan for the current environment",
		Run: func(cmd *cobra.Command, args []string) {
			envName := args[0]
			envInfo, err := config.GetEnvironmentInfo(envName)
			if err != nil {
				fmt.Println("Error:", err)
				os.Exit(1)
			}

			// Get the current AWS account ID
			currentAccountID, err := utils.GetAWSAccountID(envInfo.Region)
			if err != nil {
				fmt.Println("Error retrieving AWS account ID:", err)
				os.Exit(1)
			}

			// Check if the account IDs match
			if currentAccountID != envInfo.AccountID {
				fmt.Printf("Error: Current AWS account (%s) does not match the account configured for environment '%s' (%s).\n",
					currentAccountID, envName, envInfo.AccountID)
				os.Exit(1)
			}

			// Run terraform plan
			fmt.Printf("Running terraform plan for environment '%s'...\n", envName)
			err = utils.RunCommand("terraform", "plan", "-var-file", envInfo.LocalFile)
			if err != nil {
				fmt.Println("Error running terraform plan:", err)
				os.Exit(1)
			}
		},
	}

	return planCmd
}
