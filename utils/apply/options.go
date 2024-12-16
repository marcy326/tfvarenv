package apply

import (
	"fmt"
	"strings"
	"tfvarenv/config"
)

// Options represents apply command options
type Options struct {
	Environment   *config.Environment
	Remote        bool
	VersionID     string
	VarFile       string
	AutoApprove   bool
	TerraformOpts []string
}

func (m *Manager) checkApproval(opts *Options) error {
	if opts.Environment.Deployment.RequireApproval && !opts.AutoApprove {
		if !promptYesNo(fmt.Sprintf("\nDo you want to proceed with applying to %s environment?",
			opts.Environment.Name), false) {
			return fmt.Errorf("deployment cancelled by user")
		}
	}
	return nil
}

func promptYesNo(prompt string, defaultValue bool) bool {
	defaultStr := "Y/n"
	if !defaultValue {
		defaultStr = "y/N"
	}
	fmt.Printf("%s [%s]: ", prompt, defaultStr)

	var input string
	fmt.Scanln(&input)
	input = strings.ToLower(strings.TrimSpace(input))

	if input == "" {
		return defaultValue
	}
	return input == "y" || input == "yes"
}
