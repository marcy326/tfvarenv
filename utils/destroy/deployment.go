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
	record := &deployment.Record{
		Timestamp:   time.Now(),
		VersionID:   versionInfo.Version.VersionID,
		DeployedBy:  os.Getenv("USER"),
		Command:     "destroy",
		Status:      status,
		Environment: opts.Environment.Name,
		Parameters: map[string]string{
			"AutoApprove": fmt.Sprintf("%v", opts.AutoApprove),
		},
	}

	if err != nil {
		record.ErrorMessage = err.Error()
	}

	if recordErr := deploymentManager.AddRecord(ctx, record); recordErr != nil {
		fmt.Printf("Warning: Failed to record deployment: %v\n", recordErr)
		return
	}

	if status == "success" {
		fmt.Printf("\nDestroy operation recorded:\n")
		fmt.Printf("  Version: %s\n", versionInfo.Version.VersionID[:8])
		fmt.Printf("  Time: %s\n", record.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("  By: %s\n", record.DeployedBy)
	}
}
