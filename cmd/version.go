package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	ver       = "dev"
	commit    = "none"
	buildDate = "unknown"
)

func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("tfvarenv version %s\n", ver)
			fmt.Printf("  Git commit: %s\n", commit)
			fmt.Printf("  Built: %s\n", buildDate)
			fmt.Printf("  Go version: %s\n", runtime.Version())
			fmt.Printf("  OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		},
	}
}
