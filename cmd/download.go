package cmd

import (
	"fmt"
	"os"
	"tfvarenv/config"
	"tfvarenv/utils"

	"github.com/spf13/cobra"
)

func NewDownloadCmd() *cobra.Command {
	downloadCmd := &cobra.Command{
		Use:   "download [environment]",
		Short: "Download tfvars file from S3 to local path",
		Run: func(cmd *cobra.Command, args []string) {
			envName := args[0]
			envInfo, err := config.GetEnvironmentInfo(envName)
			if err != nil {
				fmt.Println("Error:", err)
				os.Exit(1)
			}

			_, err = utils.DownloadFromS3(envInfo.S3Key, envInfo.LocalFile, envInfo.Region)
			if err != nil {
				fmt.Println("Error downloading file:", err)
				os.Exit(1)
			}

			fmt.Printf("Successfully downloaded %s to %s\n", envInfo.S3Key, envInfo.LocalFile)
		},
	}

	return downloadCmd
}
