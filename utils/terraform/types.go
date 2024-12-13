package terraform

import (
	"tfvarenv/config"
)

// InitOptions represents options for terraform init
type InitOptions struct {
	BackendConfig string
	Reconfigure   bool
	ForceCopy     bool
	NoColor       bool
	Options       []string
}

// PlanOptions represents options for terraform plan
type PlanOptions struct {
	Environment *config.Environment
	Remote      bool
	VersionID   string
	VarFile     string
	NoColor     bool
	Options     []string
}

// ApplyOptions represents options for terraform apply
type ApplyOptions struct {
	Environment *config.Environment
	Remote      bool
	VersionID   string
	VarFile     string
	AutoApprove bool
	NoColor     bool
	Options     []string
}

// ExecutionResult represents the result of a terraform command execution
type ExecutionResult struct {
	Success     bool
	ExitCode    int
	Output      string
	ErrorOutput string
	Duration    int64
	CommandLine string
}

// ValidationResult represents the result of terraform configuration validation
type ValidationResult struct {
	Valid    bool
	Errors   []string
	Warnings []string
}

// BackendConfig represents terraform backend configuration
type BackendConfig struct {
	Type   string
	Config map[string]interface{}
}
