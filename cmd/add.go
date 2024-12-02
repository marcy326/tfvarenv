package cmd

import (
	"bufio"
    "fmt"
    "os"
    "tfvarenv/config"
    "github.com/spf13/cobra"
)

func NewAddCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "add",
        Short: "Add a new environment",
        Run: func(cmd *cobra.Command, args []string) {
            reader := bufio.NewReader(os.Stdin)

            fmt.Print("Enter environment name: ")
            envName, _ := reader.ReadString('\n')
            envName = envName[:len(envName)-1]

            fmt.Print("Enter S3 key: ")
            s3Key, _ := reader.ReadString('\n')
            s3Key = s3Key[:len(s3Key)-1]

            fmt.Print("Enter account ID: ")
            accountID, _ := reader.ReadString('\n')
            accountID = accountID[:len(accountID)-1]

            fmt.Print("Enter local file path: ")
            localFile, _ := reader.ReadString('\n')
            localFile = localFile[:len(localFile)-1]

            err := config.AddEnvironment(envName, s3Key, accountID, localFile)
            if err != nil {
                fmt.Printf("Error adding environment: %s\n", err)
                os.Exit(1)
            }
            fmt.Printf("Environment '%s' added successfully.\n", envName)
        },
    }
}