package cmd

import (
	"fmt"
	"os"
	"tfvarenv/config"
	"tfvarenv/utils"

	"github.com/spf13/cobra"
)

func NewUploadCmd() *cobra.Command {
	uploadCmd := &cobra.Command{
		Use:   "upload [environment]",
		Short: "Upload local tfvars file to S3",
		Run: func(cmd *cobra.Command, args []string) {
			envName := args[0]
			envInfo, err := config.GetEnvironmentInfo(envName)
			if err != nil {
				fmt.Println("Error:", err)
				os.Exit(1)
			}

			err = utils.UploadToS3(envInfo.LocalFile, envInfo.S3Key, envInfo.Region)
			if err != nil {
				fmt.Println("Error uploading file:", err)
				os.Exit(1)
			}

			fmt.Printf("Successfully uploaded %s to %s\n", envInfo.LocalFile, envInfo.S3Key)
		},
	}

	return uploadCmd
}
