package command

import (
	"context"

	"tfvarenv/config"
	"tfvarenv/utils/aws"
	"tfvarenv/utils/file"
	"tfvarenv/utils/terraform"
)

type Utils interface {
	GetEnvironment(name string) (*config.Environment, error)
	GetDefaultRegion() (string, error)
	GetAWSClient() aws.Client
	GetAWSClientWithRegion(region string) (aws.Client, error)
	GetFileUtils() file.Utils
	GetTerraformRunner() terraform.Runner
	GetContext() context.Context
	AddEnvironment(env *config.Environment) error
	ListEnvironments() ([]string, error)
}

// CommandInput represents common input parameters for commands
type CommandInput struct {
	EnvName     string
	Remote      bool
	VersionID   string
	Description string
	Options     string
	Force       bool
}

// ExecutionContext represents the context for command execution
type ExecutionContext struct {
	Context     context.Context
	Config      config.Manager
	AWSClient   aws.Client
	FileUtils   file.Utils
	TFRunner    terraform.Runner
	Environment *config.Environment
}

// ValidationResult represents the result of command validation
type ValidationResult struct {
	IsValid bool
	Errors  []string
}

// CommandResult represents the result of command execution
type CommandResult struct {
	Success bool
	Message string
	Error   error
	Data    interface{}
}
