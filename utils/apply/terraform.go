package apply

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"tfvarenv/utils/aws"
	"tfvarenv/utils/terraform"
)

func (m *Manager) runTerraformApply(ctx context.Context, opts *Options, versionInfo *VersionInfo) (*terraform.ExecutionResult, error) {
	tfOpts := &terraform.ApplyOptions{
		Environment: opts.Environment,
		Remote:      opts.Remote,
		VersionID:   versionInfo.Version.VersionID,
		AutoApprove: opts.AutoApprove,
		Options:     opts.TerraformOpts,
	}

	// リモートモードでtfvarsファイルを準備
	if opts.Remote {
		tmpDir := filepath.Join(".tmp", opts.Environment.Name)
		if err := m.fileUtils.EnsureDirectory(tmpDir); err != nil {
			return nil, fmt.Errorf("failed to create temporary directory: %w", err)
		}
		defer os.RemoveAll(tmpDir)

		// Download tfvars file
		input := &aws.DownloadInput{
			Bucket:    opts.Environment.S3.Bucket,
			Key:       opts.Environment.GetS3Path(),
			VersionID: versionInfo.Version.VersionID,
		}
		output, err := m.awsClient.DownloadFile(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to download tfvars: %w", err)
		}

		tmpVarFile := filepath.Join(tmpDir, "terraform.tfvars")
		if err := m.fileUtils.WriteFile(tmpVarFile, output.Content, nil); err != nil {
			return nil, fmt.Errorf("failed to write temporary tfvars file: %w", err)
		}
		tfOpts.VarFile = tmpVarFile
	} else {
		// ローカルモードではファイルパスをそのまま使用
		tfOpts.VarFile = versionInfo.SourceFile
	}

	// terraform apply実行
	return m.tfRunner.Apply(ctx, tfOpts)
}

func (m *Manager) displayResult(result *terraform.ExecutionResult, versionInfo *VersionInfo) {
	fmt.Printf("\nTerraform Apply Result:\n")
	fmt.Printf("  Status: Success\n")
	fmt.Printf("  Version: %s\n", versionInfo.Version.VersionID[:8])
	if versionInfo.IsNew {
		fmt.Printf("  Source: %s (newly uploaded)\n", versionInfo.SourceFile)
	} else if versionInfo.SourceFile != "" {
		fmt.Printf("  Source: %s\n", versionInfo.SourceFile)
	}
	fmt.Printf("  Execution Time: %dms\n", result.Duration)

	if result.Output != "" {
		fmt.Printf("\nTerraform Output:\n%s\n", result.Output)
	}
}
