package apply

import (
	"context"

	"tfvarenv/utils/aws"
	"tfvarenv/utils/file"
	"tfvarenv/utils/terraform"
)

// Manager handles the apply command execution
type Manager struct {
	awsClient aws.Client
	fileUtils file.Utils
	tfRunner  terraform.Runner
}

// NewManager creates a new apply manager
func NewManager(awsClient aws.Client, fileUtils file.Utils, tfRunner terraform.Runner) *Manager {
	return &Manager{
		awsClient: awsClient,
		fileUtils: fileUtils,
		tfRunner:  tfRunner,
	}
}

// Execute runs the apply command with given options
func (m *Manager) Execute(ctx context.Context, opts *Options) error {
	// Get version information
	versionInfo, err := m.getVersionInfo(ctx, opts)
	if err != nil {
		return err
	}

	// Get deployment approval if required
	if err := m.checkApproval(opts); err != nil {
		return err
	}

	// Run terraform apply
	result, err := m.runTerraformApply(ctx, opts, versionInfo)
	if err != nil {
		m.recordDeployment(ctx, opts, versionInfo, "failed", err)
		return err
	}

	// Record successful deployment
	m.recordDeployment(ctx, opts, versionInfo, "success", nil)

	// Display result
	m.displayResult(result, versionInfo)

	return nil
}
