package destroy

import (
	"context"
	"fmt"
	"os"
	"time"

	"tfvarenv/utils/deployment"
)

func (m *Manager) recordDeployment(ctx context.Context, opts *Options, versionInfo *VersionInfo, status string, err error) {
	deploymentManager := deployment.NewManager(m.awsClient, opts.Environment)

	if status == "success" {
		if err := deploymentManager.MarkAsDestroyed(ctx); err != nil {
			fmt.Printf("Warning: Failed to mark environment as destroyed: %v\n", err)
			return
		}

		fmt.Printf("\nEnvironment marked as destroyed:\n")
		fmt.Printf("  Time: %s\n", time.Now().Format("2006-01-02 15:04:05"))
		fmt.Printf("  By: %s\n", os.Getenv("USER"))
	}
}
