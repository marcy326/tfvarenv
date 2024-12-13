package terraform

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

// BackendTemplateData represents the data needed for backend configuration
type BackendTemplateData struct {
	BucketName string
	Region     string
	Key        string
}

const backendTemplate = `bucket         = "{{ .BucketName }}"
region         = "{{ .Region }}"
key            = "{{ .Key }}"
`

// CreateBackendConfig creates a default backend configuration file
func CreateBackendConfig(path string, data *BackendTemplateData) error {
	// Check if file already exists
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("backend config file already exists at %s", path)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Parse template
	tmpl, err := template.New("backend").Parse(backendTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Create file
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Execute template
	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to write template: %w", err)
	}

	return nil
}

// ValidateBackendConfig validates the backend configuration
func ValidateBackendConfig(path string) error {
	// Check if file exists
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("backend config file not found at %s", path)
	}

	// TODO: Add more validation logic if needed
	return nil
}
