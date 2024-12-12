package utils

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"tfvarenv/config"
	"time"
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

// RunTerraformCommand executes terraform command with specified environment and options
func RunTerraformCommand(command string, env *config.Environment, remote bool, versionID string, options string) error {
	// Check AWS account ID
	currentAccountID, err := GetAWSAccountID(env.AWS.Region)
	if err != nil {
		return fmt.Errorf("failed to get AWS account ID: %w", err)
	}

	if currentAccountID != env.AWS.AccountID {
		return fmt.Errorf("current AWS account (%s) does not match the environment configuration (%s)",
			currentAccountID, env.AWS.AccountID)
	}

	var varFile string
	if remote {
		// Create temporary directory for downloaded files
		tmpDir := filepath.Join(".tmp", env.Name)
		if err := os.MkdirAll(tmpDir, 0755); err != nil {
			return fmt.Errorf("failed to create temporary directory: %w", err)
		}
		defer os.RemoveAll(tmpDir)

		// Download tfvars file
		tmpFile := filepath.Join(tmpDir, "terraform.tfvars")
		if versionID != "" {
			// Download specific version
			varFile, err = DownloadFromS3WithVersion(env.GetS3Path(), tmpFile, env.AWS.Region, versionID)
		} else {
			// Get latest version info
			versionInfo, err := GetLatestS3Version(env.S3.Bucket, env.GetS3Path(), env.AWS.Region)
			if err != nil {
				return fmt.Errorf("failed to get latest version info: %w", err)
			}
			versionID = versionInfo.VersionID
			varFile, err = DownloadFromS3WithVersion(env.GetS3Path(), tmpFile, env.AWS.Region, versionID)
		}
		if err != nil {
			return fmt.Errorf("failed to download tfvars: %w", err)
		}

		fmt.Printf("Using tfvars version: %s\n", versionID)
	} else {
		varFile = env.Local.TFVarsPath
		fmt.Println("Using local tfvars file")
	}

	// Create backup if enabled for apply command
	if command == "apply" && remote && env.Deployment.AutoBackup {
		if err := createBackup(env.Name, varFile); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	// Construct terraform command
	args := []string{command, "-var-file=" + varFile}
	if options != "" {
		args = append(args, options)
	}

	// Run terraform command
	fmt.Printf("Running terraform %s for environment '%s'...\n", command, env.Name)
	if err := RunCommand("terraform", args...); err != nil {
		return fmt.Errorf("terraform command failed: %w", err)
	}

	// Record deployment if apply was successful
	if command == "apply" && remote {
		deploymentInfo := DeploymentInfo{
			VersionID:    versionID,
			DeployedAt:   time.Now(),
			DeployedBy:   os.Getenv("USER"),
			TerraformCmd: command,
		}

		if err := RecordDeployment(env.S3.Bucket, env.GetDeploymentHistoryKey(), env.AWS.Region, deploymentInfo); err != nil {
			fmt.Printf("Warning: Failed to record deployment: %v\n", err)
		}
	}

	return nil
}

// createBackup creates a backup of the tfvars file
func createBackup(envName, sourceFile string) error {
	backupDir := filepath.Join(".backups", envName)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return err
	}

	backupFile := filepath.Join(backupDir, fmt.Sprintf("terraform.tfvars.%s", time.Now().Format("20060102150405")))

	source, err := os.Open(sourceFile)
	if err != nil {
		return err
	}
	defer source.Close()

	dest, err := os.Create(backupFile)
	if err != nil {
		return err
	}
	defer dest.Close()

	if _, err := io.Copy(dest, source); err != nil {
		return err
	}

	fmt.Printf("Created backup: %s\n", backupFile)
	return nil
}

func UploadTFVars(envName, description string) error {
	env, err := config.GetEnvironmentInfo(envName)
	if err != nil {
		return fmt.Errorf("environment not found: %w", err)
	}

	// Check if local file exists
	if _, err := os.Stat(env.Local.TFVarsPath); err != nil {
		return fmt.Errorf("local file not found: %w", err)
	}

	// Calculate file hash and size
	hash, err := CalculateFileHash(env.Local.TFVarsPath)
	if err != nil {
		return fmt.Errorf("failed to calculate file hash: %w", err)
	}

	fileInfo, err := os.Stat(env.Local.TFVarsPath)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Upload to S3 and get version information
	versionInfo, err := UploadToS3WithVersioning(
		env.Local.TFVarsPath,
		env.GetS3Path(),
		env.AWS.Region,
		description,
	)
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	// Create version record
	version := &Version{
		VersionID:   versionInfo.VersionID,
		Timestamp:   time.Now(),
		Hash:        hash,
		Description: description,
		UploadedBy:  os.Getenv("USER"),
		Size:        fileInfo.Size(),
	}

	// Add version to version management
	if err := AddVersion(env, version); err != nil {
		return fmt.Errorf("failed to record version: %w", err)
	}

	fmt.Printf("Successfully uploaded %s to %s\n", env.Local.TFVarsPath, env.GetS3Path())
	fmt.Printf("Version ID: %s\n", versionInfo.VersionID)
	if description != "" {
		fmt.Printf("Description: %s\n", description)
	}

	return nil
}

func DownloadTFVars(envName, versionID string) error {
	env, err := config.GetEnvironmentInfo(envName)
	if err != nil {
		return fmt.Errorf("environment not found: %w", err)
	}

	// Ensure local directory exists
	if err := os.MkdirAll(filepath.Dir(env.Local.TFVarsPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Download file
	var downloadPath string
	if versionID != "" {
		downloadPath, err = DownloadFromS3WithVersion(
			env.GetS3Path(),
			env.Local.TFVarsPath,
			env.AWS.Region,
			versionID,
		)
	} else {
		downloadPath, err = DownloadFromS3(
			env.GetS3Path(),
			env.Local.TFVarsPath,
			env.AWS.Region,
		)
	}

	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	fmt.Printf("Successfully downloaded %s to %s\n", env.GetS3Path(), downloadPath)
	return nil
}

func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
