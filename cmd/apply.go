package cmd

import (
	"fmt"
	"os"
	"tfvarenv/config"
	"tfvarenv/utils"

	"github.com/spf13/cobra"
)

func NewApplyCmd() *cobra.Command {
	applyCmd := &cobra.Command{
		Use:   "apply",
		Short: "Run terraform apply for the current environment",
		Run: func(cmd *cobra.Command, args []string) {
			currentEnv := config.GetCurrentEnvironment()
			if currentEnv == "" {
				fmt.Println("Error: No environment selected. Use `tfvarenv env use` to select an environment.")
				os.Exit(1)
			}

			envInfo, err := config.GetEnvironmentInfo(currentEnv)
			if err != nil {
				fmt.Println("Error:", err)
				os.Exit(1)
			}

			// Get the current AWS account ID
			currentAccountID, err := utils.GetAWSAccountID()
			if err != nil {
				fmt.Println("Error retrieving AWS account ID:", err)
				os.Exit(1)
			}

			// Check if the account IDs match
			if currentAccountID != envInfo.AccountID {
				fmt.Printf("Error: Current AWS account (%s) does not match the account configured for environment '%s' (%s).\n",
					currentAccountID, currentEnv, envInfo.AccountID)
				os.Exit(1)
			}

			// Confirmation prompt
			fmt.Printf("You are about to apply changes to the '%s' environment (AWS account: %s).\n", currentEnv, envInfo.AccountID)
			fmt.Print("Type 'yes' to confirm: ")
			var input string
			fmt.Scanln(&input)
			if input != "yes" {
				fmt.Println("Operation cancelled.")
				return
			}

			// Run terraform apply (placeholder)
			fmt.Printf("Running terraform apply for environment '%s'...\n", currentEnv)
		},
	}

	return applyCmd
}
