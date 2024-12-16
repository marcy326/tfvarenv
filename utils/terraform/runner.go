package terraform

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"tfvarenv/utils/aws"
	"tfvarenv/utils/file"
)

type Runner interface {
	Init(ctx context.Context, opts *InitOptions) (*ExecutionResult, error)
	Plan(ctx context.Context, opts *PlanOptions) (*ExecutionResult, error)
	Apply(ctx context.Context, opts *ApplyOptions) (*ExecutionResult, error)
	Destroy(ctx context.Context, opts *DestroyOptions) (*ExecutionResult, error)
	Validate(ctx context.Context) (*ValidationResult, error)
}

type runner struct {
	awsClient aws.Client
	fileUtils file.Utils
	workDir   string
}

func NewRunner(awsClient aws.Client, fileUtils file.Utils) Runner {
	return &runner{
		awsClient: awsClient,
		fileUtils: fileUtils,
		workDir:   ".",
	}
}

func (r *runner) Init(ctx context.Context, opts *InitOptions) (*ExecutionResult, error) {
	args := []string{"init"}

	for key, value := range opts.BackendConfigs {
		args = append(args, fmt.Sprintf("-backend-config=%s=%s", key, value))
	}
	if opts.Reconfigure {
		args = append(args, "-reconfigure")
	}
	if opts.ForceCopy {
		args = append(args, "-force-copy")
	}
	if opts.NoColor {
		args = append(args, "-no-color")
	}
	if len(opts.Options) > 0 {
		args = append(args, opts.Options...)
	}

	return r.runCommand(ctx, args)
}

func (r *runner) Plan(ctx context.Context, opts *PlanOptions) (*ExecutionResult, error) {
	// AWS account verification
	accountID, err := r.awsClient.GetAccountID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS account ID: %w", err)
	}
	if accountID != opts.Environment.AWS.AccountID {
		return nil, fmt.Errorf("current AWS account (%s) does not match environment configuration (%s)",
			accountID, opts.Environment.AWS.AccountID)
	}

	args := []string{"plan"}

	// Handle remote vs local tfvars
	if opts.Remote {
		tmpDir := filepath.Join(".tmp", opts.Environment.Name)
		if err := r.fileUtils.EnsureDirectory(tmpDir); err != nil {
			return nil, fmt.Errorf("failed to create temporary directory: %w", err)
		}
		defer os.RemoveAll(tmpDir)

		// Download tfvars file
		input := &aws.DownloadInput{
			Bucket:    opts.Environment.S3.Bucket,
			Key:       opts.Environment.GetS3Path(),
			VersionID: opts.VersionID,
		}
		output, err := r.awsClient.DownloadFile(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to download tfvars: %w", err)
		}

		tmpVarFile := filepath.Join(tmpDir, "terraform.tfvars")
		if err := r.fileUtils.WriteFile(tmpVarFile, output.Content, nil); err != nil {
			return nil, fmt.Errorf("failed to write temporary tfvars file: %w", err)
		}
		opts.VarFile = tmpVarFile
	}

	if opts.VarFile != "" {
		args = append(args, "-var-file="+opts.VarFile)
	}
	if opts.NoColor {
		args = append(args, "-no-color")
	}
	if len(opts.Options) > 0 {
		args = append(args, opts.Options...)
	}

	return r.runCommand(ctx, args)
}

func (r *runner) Apply(ctx context.Context, opts *ApplyOptions) (*ExecutionResult, error) {
	// AWS account verification
	accountID, err := r.awsClient.GetAccountID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS account ID: %w", err)
	}
	if accountID != opts.Environment.AWS.AccountID {
		return nil, fmt.Errorf("current AWS account (%s) does not match environment configuration (%s)",
			accountID, opts.Environment.AWS.AccountID)
	}

	args := []string{"apply"}

	// Handle remote vs local tfvars
	if opts.Remote {
		tmpDir := filepath.Join(".tmp", opts.Environment.Name)
		if err := r.fileUtils.EnsureDirectory(tmpDir); err != nil {
			return nil, fmt.Errorf("failed to create temporary directory: %w", err)
		}
		defer os.RemoveAll(tmpDir)

		// Download tfvars file
		input := &aws.DownloadInput{
			Bucket:    opts.Environment.S3.Bucket,
			Key:       opts.Environment.GetS3Path(),
			VersionID: opts.VersionID,
		}
		output, err := r.awsClient.DownloadFile(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to download tfvars: %w", err)
		}

		tmpVarFile := filepath.Join(tmpDir, "terraform.tfvars")
		if err := r.fileUtils.WriteFile(tmpVarFile, output.Content, &file.Options{
			CreateDirs: true,
			Overwrite:  true,
		}); err != nil {
			return nil, fmt.Errorf("failed to write temporary tfvars file: %w", err)
		}
		opts.VarFile = tmpVarFile
	}

	if opts.VarFile != "" {
		args = append(args, "-var-file="+opts.VarFile)
	}
	if opts.AutoApprove {
		args = append(args, "-auto-approve")
	}
	if opts.NoColor {
		args = append(args, "-no-color")
	}
	if len(opts.Options) > 0 {
		args = append(args, opts.Options...)
	}

	return r.runCommand(ctx, args)
}

func (r *runner) Validate(ctx context.Context) (*ValidationResult, error) {
	args := []string{"validate", "-json"}
	result, err := r.runCommand(ctx, args)
	if err != nil {
		return nil, err
	}

	validation := &ValidationResult{
		Valid:    result.Success,
		Errors:   make([]string, 0),
		Warnings: make([]string, 0),
	}

	return validation, nil
}

func (r *runner) Destroy(ctx context.Context, opts *DestroyOptions) (*ExecutionResult, error) {
	args := []string{"destroy"}

	if opts.VarFile != "" {
		args = append(args, "-var-file="+opts.VarFile)
	}
	if opts.AutoApprove {
		args = append(args, "-auto-approve")
	}
	if opts.NoColor {
		args = append(args, "-no-color")
	}
	if len(opts.Options) > 0 {
		args = append(args, opts.Options...)
	}

	return r.runCommand(ctx, args)
}

func (r *runner) runCommand(ctx context.Context, args []string) (*ExecutionResult, error) {
	cmd := exec.CommandContext(ctx, "terraform", args...)
	cmd.Dir = r.workDir

	cmd.Stdin = os.Stdin

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

	cmd.Env = os.Environ()

	startTime := time.Now()
	err := cmd.Run()
	duration := time.Since(startTime).Milliseconds()

	result := &ExecutionResult{
		Success:     err == nil,
		ExitCode:    0,
		Output:      stdoutBuf.String(),
		ErrorOutput: stderrBuf.String(),
		Duration:    duration,
		CommandLine: fmt.Sprintf("terraform %s", strings.Join(args, " ")),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
		return result, fmt.Errorf("terraform command failed: %w", err)
	}

	return result, nil
}
