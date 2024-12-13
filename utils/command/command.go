package command

import (
	"context"
	"fmt"

	"tfvarenv/config"
	"tfvarenv/utils/aws"
	"tfvarenv/utils/file"
	"tfvarenv/utils/terraform"
)

type commandUtils struct {
	ctx        context.Context
	awsClient  aws.Client
	fileUtils  file.Utils
	tfRunner   terraform.Runner
	cfgManager config.Manager
}

// NewUtils creates a new command utilities instance
func NewUtils() (Utils, error) {
	ctx := context.Background()

	cfgManager, err := config.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize config manager: %w", err)
	}

	defaultRegion, err := cfgManager.GetDefaultRegion()
	if err != nil {
		return nil, fmt.Errorf("failed to get default region: %w", err)
	}

	awsClient, err := aws.NewClient(defaultRegion)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize AWS client: %w", err)
	}

	fileUtils := file.NewUtils()
	tfRunner := terraform.NewRunner(awsClient, fileUtils)

	return &commandUtils{
		ctx:        ctx,
		awsClient:  awsClient,
		fileUtils:  fileUtils,
		tfRunner:   tfRunner,
		cfgManager: cfgManager,
	}, nil
}

func (c *commandUtils) GetExecutionContext(envName string) (*ExecutionContext, error) {
	env, err := c.cfgManager.GetEnvironment(envName)
	if err != nil {
		return nil, fmt.Errorf("failed to get environment: %w", err)
	}

	return &ExecutionContext{
		Context:     c.ctx,
		Config:      c.cfgManager,
		AWSClient:   c.awsClient,
		FileUtils:   c.fileUtils,
		TFRunner:    c.tfRunner,
		Environment: env,
	}, nil
}

func (c *commandUtils) ValidateInput(input *CommandInput) *ValidationResult {
	result := &ValidationResult{
		IsValid: true,
		Errors:  make([]string, 0),
	}

	if input.EnvName == "" {
		result.IsValid = false
		result.Errors = append(result.Errors, "environment name is required")
	}

	if input.Remote && input.VersionID == "" {
		result.IsValid = false
		result.Errors = append(result.Errors, "version ID is required when using remote flag")
	}

	return result
}

func (c *commandUtils) HandleError(err error) error {
	if err == nil {
		return nil
	}

	// Add context or additional information to the error if needed
	return fmt.Errorf("command execution failed: %w", err)
}

func (c *commandUtils) GetEnvironment(name string) (*config.Environment, error) {
	return c.cfgManager.GetEnvironment(name)
}

func (c *commandUtils) GetDefaultRegion() (string, error) {
	return c.cfgManager.GetDefaultRegion()
}

func (c *commandUtils) GetAWSClient() aws.Client {
	return c.awsClient
}

func (c *commandUtils) GetAWSClientWithRegion(region string) (aws.Client, error) {
	return aws.NewClient(region)
}

func (c *commandUtils) GetFileUtils() file.Utils {
	return c.fileUtils
}

func (c *commandUtils) GetTerraformRunner() terraform.Runner {
	return c.tfRunner
}

func (c *commandUtils) GetContext() context.Context {
	return c.ctx
}

func (c *commandUtils) AddEnvironment(env *config.Environment) error {
	// 環境を追加するためのロジックをここに実装します
	return c.cfgManager.AddEnvironment(env.Name, env)
}

func (c *commandUtils) ListEnvironments() ([]string, error) {
	return c.cfgManager.ListEnvironments()
}
