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
			envName := args[0]
			remote, _ := cmd.Flags().GetBool("remote")
			options, _ := cmd.Flags().GetString("options")
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

			var varFile string
			if remote {
				// Download tfvars file from S3
				varFile, err = utils.DownloadFromS3(envInfo.S3Key, ".tmp/", envInfo.Region)
				if err != nil {
					fmt.Println("Error downloading tfvars file from S3:", err)
					os.Exit(1)
				}
			} else {
				varFile = envInfo.LocalFile
			}

			// Run terraform apply
			fmt.Printf("Running terraform apply for environment '%s' (remote: %v)...\n", envName, remote)
			err = utils.RunCommand("terraform", "apply", "-var-file", varFile, options)
			if err != nil {
				fmt.Println("Error running terraform apply:", err)
				os.Exit(1)
			}
		},
	}

	applyCmd.Flags().Bool("remote", false, "Run the apply in a remote environment")
	applyCmd.Flags().String("options", "", "Additional options for terraform apply")

	return applyCmd
}
