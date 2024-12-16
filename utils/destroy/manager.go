package destroy

import (
	"context"
	"fmt"

	"tfvarenv/utils/aws"
	"tfvarenv/utils/deployment"
	"tfvarenv/utils/file"
	"tfvarenv/utils/terraform"
	"tfvarenv/utils/version"
)

type Manager struct {
	awsClient aws.Client
	fileUtils file.Utils
	tfRunner  terraform.Runner
}

func NewManager(awsClient aws.Client, fileUtils file.Utils, tfRunner terraform.Runner) *Manager {
	return &Manager{
		awsClient: awsClient,
		fileUtils: fileUtils,
		tfRunner:  tfRunner,
	}
}

func (m *Manager) Execute(ctx context.Context, opts *Options) error {
	// Get version information
	versionInfo, err := m.getVersionToDestroy(ctx, opts)
	if err != nil {
		return err
	}

	// Display destroy plan
	m.displayDestroyPlan(opts, versionInfo)

	// Get confirmation
	if !opts.AutoApprove {
		if !m.confirmDestroy(opts.Environment.Name) {
			return fmt.Errorf("destroy cancelled by user")
		}
	}

	// Run terraform destroy
	result, err := m.runTerraformDestroy(ctx, opts, versionInfo)
	if err != nil {
		m.recordDeployment(ctx, opts, versionInfo, "failed", err)
		return err
	}

	// Record successful destruction
	m.recordDeployment(ctx, opts, versionInfo, "success", nil)

	// Display result
	m.displayResult(result, versionInfo)

	return nil
}

func (m *Manager) getVersionToDestroy(ctx context.Context, opts *Options) (*VersionInfo, error) {
	versionManager := version.NewManager(m.awsClient, m.fileUtils, opts.Environment)
	deploymentManager := deployment.NewManager(m.awsClient, opts.Environment)

	// If version ID is specified, use it
	if opts.VersionID != "" {
		ver, err := versionManager.GetVersion(ctx, opts.VersionID)
		if err != nil {
			return nil, fmt.Errorf("failed to get specified version: %w", err)
		}
		return &VersionInfo{Version: ver}, nil
	}

	// Get last deployed version
	lastDeploy, err := deploymentManager.GetLatestDeployment(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest deployment: %w", err)
	}
	if lastDeploy == nil {
		return nil, fmt.Errorf("no deployment history found")
	}

	ver, err := versionManager.GetVersion(ctx, lastDeploy.VersionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get last deployed version: %w", err)
	}

	return &VersionInfo{
		Version:          ver,
		LastDeployedTime: lastDeploy.Timestamp,
		LastDeployedBy:   lastDeploy.DeployedBy,
	}, nil
}

func (m *Manager) displayDestroyPlan(opts *Options, versionInfo *VersionInfo) {
	fmt.Printf("\nDestroy Plan for environment '%s':\n", opts.Environment.Name)
	fmt.Printf("  AWS Account: %s (%s)\n", opts.Environment.AWS.AccountID, opts.Environment.AWS.Region)
	fmt.Printf("  Using Version: %s\n", versionInfo.Version.VersionID[:8])
	fmt.Printf("  Last Deployed: %s by %s\n",
		versionInfo.LastDeployedTime.Format("2006-01-02 15:04:05"),
		versionInfo.LastDeployedBy)
	if versionInfo.Version.Description != "" {
		fmt.Printf("  Version Description: %s\n", versionInfo.Version.Description)
	}
	fmt.Println("\nThis operation will DESTROY all resources managed by Terraform!")
}

func (m *Manager) confirmDestroy(envName string) bool {
	fmt.Printf("\nAre you absolutely sure you want to destroy all resources in '%s'?\n", envName)
	fmt.Println("  Type the environment name to confirm: ")

	var input string
	fmt.Scanln(&input)
	return input == envName
}
