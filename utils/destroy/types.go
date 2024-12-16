package destroy

import (
	"time"

	"tfvarenv/config"
	"tfvarenv/utils/version"
)

// Options represents destroy command options
type Options struct {
	Environment   *config.Environment
	VersionID     string // Optional: specific version to use
	AutoApprove   bool
	TerraformOpts []string
}

// VersionInfo contains version and deployment information
type VersionInfo struct {
	Version          *version.Version
	LastDeployedTime time.Time
	LastDeployedBy   string
}
