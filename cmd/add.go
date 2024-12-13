package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"tfvarenv/config"
	"tfvarenv/utils/aws"
	"tfvarenv/utils/command"
	"tfvarenv/utils/deployment"
	"tfvarenv/utils/file"
	"tfvarenv/utils/terraform"
	"tfvarenv/utils/version"
)

func NewAddCmd() *cobra.Command {
	utils, err := command.NewUtils()
	if err != nil {
		fmt.Printf("Error initializing command utils: %v\n", err)
		os.Exit(1)
	}

	return &cobra.Command{
		Use:   "add",
		Short: "Add a new environment",
		Run: func(cmd *cobra.Command, args []string) {
			if err := runAdd(cmd.Context(), utils); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		},
	}
}

func runAdd(ctx context.Context, utils command.Utils) error {
	reader := bufio.NewReader(os.Stdin)
	env := &config.Environment{}

	// Basic information
	fmt.Print("Enter environment name: ")
	envName, _ := reader.ReadString('\n')
	envName = strings.TrimSpace(envName)

	fmt.Print("Enter environment description (optional): ")
	description, _ := reader.ReadString('\n')
	env.Name = envName
	env.Description = strings.TrimSpace(description)

	// S3 Configuration
	fmt.Println("\nS3 Configuration:")
	fmt.Print("Enter bucket name: ")
	bucket, _ := reader.ReadString('\n')
	bucket = strings.TrimSpace(bucket)

	defaultPrefix := fmt.Sprintf("terraform/%s", envName)
	fmt.Printf("Enter prefix [%s]: ", defaultPrefix)
	prefix, _ := reader.ReadString('\n')
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = defaultPrefix
	}

	defaultTFVarsKey := "terraform.tfvars"
	fmt.Printf("Enter tfvars file name [%s]: ", defaultTFVarsKey)
	tfvarsKey, _ := reader.ReadString('\n')
	tfvarsKey = strings.TrimSpace(tfvarsKey)
	if tfvarsKey == "" {
		tfvarsKey = defaultTFVarsKey
	}

	// Verify S3 bucket and versioning
	if err := utils.GetAWSClient().CheckBucketVersioning(ctx, bucket); err != nil {
		return fmt.Errorf("S3 bucket verification failed: %w", err)
	}

	env.S3 = config.EnvironmentS3Config{
		Bucket:    bucket,
		Prefix:    prefix,
		TFVarsKey: tfvarsKey,
	}

	// AWS Configuration
	fmt.Println("\nAWS Configuration:")
	defaultRegion, err := utils.GetDefaultRegion()
	if err != nil {
		return fmt.Errorf("failed to get default region: %w", err)
	}

	fmt.Printf("Region [%s]: ", defaultRegion)
	region, _ := reader.ReadString('\n')
	region = strings.TrimSpace(region)
	if region == "" {
		region = defaultRegion
	}

	// Get AWS Account ID using the specified region
	awsClient, err := utils.GetAWSClientWithRegion(region)
	if err != nil {
		return fmt.Errorf("failed to initialize AWS client: %w", err)
	}

	accountID, err := awsClient.GetAccountID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get AWS account ID: %w", err)
	}

	env.AWS = config.AWSConfig{
		AccountID: accountID,
		Region:    region,
	}

	// Local Configuration
	defaultLocalPath := filepath.Join("envs", envName, tfvarsKey)
	fmt.Printf("\nLocal Configuration:\nEnter local tfvars path [%s]: ", defaultLocalPath)
	localPath, _ := reader.ReadString('\n')
	localPath = strings.TrimSpace(localPath)
	if localPath == "" {
		localPath = defaultLocalPath
	}

	env.Local = config.LocalConfig{
		TFVarsPath: localPath,
	}

	// Backend Configuration
	defaultBackendPath := filepath.Join("envs", envName, "backend.tfvars")
	env.Backend = config.BackendConfig{
		ConfigPath: defaultBackendPath,
	}

	// Deployment Configuration
	fmt.Println("\nDeployment Configuration:")
	env.Deployment = config.DeploymentConfig{
		AutoBackup:      promptYesNo("Enable auto backup?", true),
		RequireApproval: promptYesNo("Require deployment approval?", envName != "dev"),
	}

	// Setup local environment
	fmt.Print("\nSetting up local environment")
	if err := setupLocalEnvironment(utils.GetFileUtils(), env); err != nil {
		return fmt.Errorf("failed to setup local environment: %w", err)
	}
	fmt.Println(": done")

	// Create backend configuration
	backendData := &terraform.BackendTemplateData{
		BucketName: bucket,
		Region:     region,
		Key:        fmt.Sprintf("%s/terraform.tfstate", prefix),
	}
	if err := terraform.CreateBackendConfig(env.Backend.ConfigPath, backendData); err != nil {
		return fmt.Errorf("failed to create backend configuration: %w", err)
	}

	// Add environment to configuration
	fmt.Print("\nAdding environment to configuration")
	if err := utils.AddEnvironment(env); err != nil {
		return fmt.Errorf("failed to add environment: %w", err)
	}
	fmt.Println(": done")

	// Check file status
	if err := checkFilesStatus(ctx, utils, env); err != nil {
		fmt.Printf("Warning: Failed to check file status: %v\n", err)
	} else {
		fmt.Println("\nUse the following commands to manage tfvars:")
		fmt.Printf("- Download: tfvarenv download %s\n", envName)
		fmt.Printf("- Upload:   tfvarenv upload %s\n", envName)
		fmt.Printf("- Plan:     tfvarenv plan %s\n", envName)
		fmt.Printf("- Apply:    tfvarenv apply %s\n", envName)
	}

	fmt.Printf("\nEnvironment '%s' added successfully.\n", envName)
	return nil
}

func setupLocalEnvironment(fileUtils file.Utils, env *config.Environment) error {
	// Create directory structure
	dir := filepath.Dir(env.Local.TFVarsPath)
	if err := fileUtils.EnsureDirectory(dir); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Update .gitignore
	if err := updateGitignore(fileUtils); err != nil {
		return fmt.Errorf("failed to update .gitignore: %w", err)
	}

	return nil
}

func updateGitignore(fileUtils file.Utils) error {
	entries := []string{
		"*.tfvars",
		".terraform/",
		".terraform.lock.hcl",
	}

	content, err := fileUtils.ReadFile(".gitignore")
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read .gitignore: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	existing := make(map[string]bool)
	for _, line := range lines {
		existing[strings.TrimSpace(line)] = true
	}

	var newContent []string
	newContent = append(newContent, lines...)

	added := false
	for _, entry := range entries {
		if !existing[entry] {
			if !added {
				// Add a blank line before new entries if the file is not empty
				if len(newContent) > 0 && newContent[len(newContent)-1] != "" {
					newContent = append(newContent, "")
				}
				added = true
			}
			newContent = append(newContent, entry)
		}
	}

	if added {
		return fileUtils.WriteFile(".gitignore", []byte(strings.Join(newContent, "\n")+"\n"), nil)
	}

	return nil
}

func promptYesNo(prompt string, defaultValue bool) bool {
	defaultStr := "Y/n"
	if !defaultValue {
		defaultStr = "y/N"
	}
	fmt.Printf("%s [%s]: ", prompt, defaultStr)

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.ToLower(strings.TrimSpace(input))

	if input == "" {
		return defaultValue
	}
	return input == "y" || input == "yes"
}

func checkFilesStatus(ctx context.Context, utils command.Utils, env *config.Environment) error {
	// Check local file existence
	fileUtils := utils.GetFileUtils()
	localExists, err := fileUtils.FileExists(env.Local.TFVarsPath)
	if err != nil {
		return fmt.Errorf("failed to check local file: %w", err)
	}

	// Check remote file existence
	versionManager := version.NewManager(utils.GetAWSClient(), fileUtils, env)
	latestVersion, err := versionManager.GetLatestVersion(ctx)
	remoteExists := err == nil && latestVersion != nil

	fmt.Println("\nFile Status Check:")

	switch {
	case !localExists && !remoteExists:
		return handleNoFiles(fileUtils, env)
	case !localExists && remoteExists:
		return handleRemoteOnly(ctx, utils, env, latestVersion)
	case localExists && !remoteExists:
		return handleLocalOnly(ctx, utils, env)
	default:
		return handleBothExist(ctx, utils, env, latestVersion)
	}
}

// ファイルが存在しない場合の処理
func handleNoFiles(fileUtils file.Utils, env *config.Environment) error {
	fmt.Println("No tfvars file found in either location.")

	// Create empty file
	opts := &file.Options{
		CreateDirs: true,
		Overwrite:  false,
	}
	if err := fileUtils.WriteFile(env.Local.TFVarsPath, []byte(""), opts); err != nil {
		return fmt.Errorf("failed to create empty tfvars file: %w", err)
	}

	fmt.Printf("Created empty tfvars file at: %s\n", env.Local.TFVarsPath)
	fmt.Println("Action needed: Edit the tfvars file and use 'tfvarenv upload' to sync.")
	return nil
}

// リモートのみ存在する場合の処理
func handleRemoteOnly(ctx context.Context, utils command.Utils, env *config.Environment, ver *version.Version) error {
	fmt.Printf("\nFound remote tfvars file:\n")
	fmt.Printf("  Version ID: %s\n", ver.VersionID[:8])
	fmt.Printf("  Uploaded: %s\n", ver.Timestamp.Format("2006-01-02 15:04:05"))
	if ver.Description != "" {
		fmt.Printf("  Description: %s\n", ver.Description)
	}

	if promptYesNo("\nWould you like to download it now?", true) {
		downloadOpts := &aws.DownloadInput{
			Bucket:    env.S3.Bucket,
			Key:       env.GetS3Path(),
			VersionID: ver.VersionID,
		}

		output, err := utils.GetAWSClient().DownloadFile(ctx, downloadOpts)
		if err != nil {
			return fmt.Errorf("failed to download tfvars: %w", err)
		}

		opts := &file.Options{
			CreateDirs: true,
			Overwrite:  false,
		}
		if err := utils.GetFileUtils().WriteFile(env.Local.TFVarsPath, output.Content, opts); err != nil {
			return fmt.Errorf("failed to write tfvars file: %w", err)
		}

		fmt.Println("Successfully downloaded tfvars file.")
	} else {
		fmt.Println("Action needed: Use 'tfvarenv download' when ready to sync.")
	}

	return nil
}

// ローカルのみ存在する場合の処理
func handleLocalOnly(ctx context.Context, utils command.Utils, env *config.Environment) error {
	fmt.Println("Found local tfvars file but no remote file.")

	if promptYesNo("Would you like to upload it now?", true) {
		content, err := utils.GetFileUtils().ReadFile(env.Local.TFVarsPath)
		if err != nil {
			return fmt.Errorf("failed to read local file: %w", err)
		}

		uploadOpts := &aws.UploadInput{
			Bucket:      env.S3.Bucket,
			Key:         env.GetS3Path(),
			Content:     content,
			Description: "Initial upload during environment setup",
			Metadata: map[string]string{
				"Environment": env.Name,
				"UploadedBy":  os.Getenv("USER"),
			},
		}

		_, err = utils.GetAWSClient().UploadFile(ctx, uploadOpts)
		if err != nil {
			return fmt.Errorf("failed to upload tfvars: %w", err)
		}

		fmt.Println("Successfully uploaded tfvars file.")
	} else {
		fmt.Println("Action needed: Use 'tfvarenv upload' when ready to sync.")
	}

	return nil
}

// ローカルとリモートの両方が存在する場合の処理
func handleBothExist(ctx context.Context, utils command.Utils, env *config.Environment, remoteVer *version.Version) error {
	// Calculate local file hash
	localHash, err := utils.GetFileUtils().CalculateHash(env.Local.TFVarsPath, nil)
	if err != nil {
		return fmt.Errorf("failed to calculate local file hash: %w", err)
	}

	if localHash == remoteVer.Hash {
		fmt.Println("Local and remote files are in sync.")
		fmt.Printf("\nCurrent version information:\n")
		fmt.Printf("  Version ID: %s\n", remoteVer.VersionID[:8])
		fmt.Printf("  Uploaded: %s\n", remoteVer.Timestamp.Format("2006-01-02 15:04:05"))
		if remoteVer.Description != "" {
			fmt.Printf("  Description: %s\n", remoteVer.Description)
		}
	} else {
		fmt.Println("Warning: Local and remote files are different!")
		fmt.Printf("\nRemote version information:\n")
		fmt.Printf("  Version ID: %s\n", remoteVer.VersionID[:8])
		fmt.Printf("  Uploaded: %s\n", remoteVer.Timestamp.Format("2006-01-02 15:04:05"))
		if remoteVer.Description != "" {
			fmt.Printf("  Description: %s\n", remoteVer.Description)
		}

		fmt.Println("\nAction needed: Use one of the following commands to sync files:")
		fmt.Printf("  Download remote version: tfvarenv download %s\n", env.Name)
		fmt.Printf("  Upload local changes:    tfvarenv upload %s\n", env.Name)
	}

	// Get deployment status if remote exists
	deploymentManager := deployment.NewManager(utils.GetAWSClient(), env)
	latestDeployment, err := deploymentManager.GetLatestDeployment(ctx)
	if err == nil && latestDeployment != nil {
		if latestDeployment.VersionID == remoteVer.VersionID {
			fmt.Printf("\nDeployment Status: Last deployed on %s by %s\n",
				latestDeployment.Timestamp.Format("2006-01-02 15:04:05"),
				latestDeployment.DeployedBy)
		}
	}

	return nil
}

// テストのためにエクスポートする型を定義
type TestExports struct {
	HandleNoFiles    func(fileUtils file.Utils, env *config.Environment) error
	HandleRemoteOnly func(ctx context.Context, utils command.Utils, env *config.Environment, ver *version.Version) error
	HandleLocalOnly  func(ctx context.Context, utils command.Utils, env *config.Environment) error
	HandleBothExist  func(ctx context.Context, utils command.Utils, env *config.Environment, remoteVer *version.Version) error
}

// テスト用にエクスポートする関数
var Exports = TestExports{
	HandleNoFiles:    handleNoFiles,
	HandleRemoteOnly: handleRemoteOnly,
	HandleLocalOnly:  handleLocalOnly,
	HandleBothExist:  handleBothExist,
}
