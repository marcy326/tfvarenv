package utils

import (
	"fmt"
	"os"
	"os/exec"
)

// RunCommand executes a shell command and returns an error if it fails.
func RunCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("Executing command: %s %v\n", name, args)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("command execution failed: %w", err)
	}
	return nil
}
