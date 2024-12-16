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
	"tfvarenv/utils/file"
	"tfvarenv/utils/prompt"
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

	// Deployment Configuration
	fmt.Println("\nDeployment Configuration:")
	env.Deployment = config.DeploymentConfig{
		AutoBackup:      prompt.PromptYesNo("Enable auto backup?", true),
		RequireApproval: prompt.PromptYesNo("Require deployment approval?", envName != "dev"),
	}

	// Setup local environment
	fmt.Print("\nSetting up local environment")
	if err := setupLocalEnvironment(utils.GetFileUtils(), env); err != nil {
		return fmt.Errorf("failed to setup local environment: %w", err)
	}
	fmt.Println(": done")

	// Backend Configuration
	fmt.Println("\nBackend Configuration:")

	defaultBackendBucket := env.S3.Bucket
	fmt.Printf("Enter backend bucket name [%s]: ", defaultBackendBucket)
	backendBucket, _ := reader.ReadString('\n')
	backendBucket = strings.TrimSpace(backendBucket)
	if backendBucket == "" {
		backendBucket = defaultBackendBucket
	}

	defaultBackendKey := filepath.Join(env.S3.Prefix, "terraform.tfstate")
	fmt.Printf("Enter backend key [%s]: ", defaultBackendKey)
	backendKey, _ := reader.ReadString('\n')
	backendKey = strings.TrimSpace(backendKey)
	if backendKey == "" {
		backendKey = defaultBackendKey
	}

	defaultBackendRegion := env.AWS.Region
	fmt.Printf("Enter backend region [%s]: ", defaultBackendRegion)
	backendRegion, _ := reader.ReadString('\n')
	backendRegion = strings.TrimSpace(backendRegion)
	if backendRegion == "" {
		backendRegion = defaultBackendRegion
	}

	env.Backend = config.BackendConfig{
		Bucket: backendBucket,
		Key:    backendKey,
		Region: backendRegion,
	}

	// Add environment to configuration
	fmt.Print("\nAdding environment to configuration")
	if err := utils.AddEnvironment(env); err != nil {
		return fmt.Errorf("failed to add environment: %w", err)
	}
	fmt.Println(": done")

	errChan := make(chan error, 1)
	go func() {
		errChan <- checkFilesStatus(ctx, utils, env)
	}()

	// Check file status
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		if err != nil {
			fmt.Printf("Warning: Failed to check file status: %v\n", err)
		}
	}
	fmt.Printf("\nEnvironment '%s' added successfully.\n", envName)

	fmt.Printf("\nFile locations:\n")
	fmt.Printf("  Local: %s\n", env.Local.TFVarsPath)
	fmt.Printf("  Remote: %s\n", env.GetFullS3Path())

	fmt.Println("\nUse the following commands to manage tfvars:")
	fmt.Printf("- Download: tfvarenv download %s\n", envName)
	fmt.Printf("- Upload:   tfvarenv upload %s\n", envName)
	fmt.Printf("- Plan:     tfvarenv plan %s\n", envName)
	fmt.Printf("- Apply:    tfvarenv apply %s\n", envName)

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

func checkFilesStatus(ctx context.Context, utils command.Utils, env *config.Environment) error {
	// Check local file existence
	fileUtils := utils.GetFileUtils()
	localExists, err := fileUtils.FileExists(env.Local.TFVarsPath)
	if err != nil {
		return fmt.Errorf("failed to check local file: %w", err)
	}

	// Check remote file existence
	downloadInput := &aws.DownloadInput{
		Bucket: env.S3.Bucket,
		Key:    env.GetS3Path(),
	}
	_, err = utils.GetAWSClient().DownloadFile(ctx, downloadInput)
	remoteExists := err == nil

	fmt.Println("\nFile Status Check:")

	switch {
	case !localExists && !remoteExists:
		return handleNoFiles(fileUtils, env)
	case !localExists && remoteExists:
		return handleRemoteOnly(ctx, utils, env)
	case localExists && !remoteExists:
		return handleLocalOnly(ctx, utils, env)
	default:
		return handleBothExist(ctx, utils, env)
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
func handleRemoteOnly(ctx context.Context, utils command.Utils, env *config.Environment) error {
	fmt.Println("Found remote tfvars file but no local file")

	if prompt.PromptYesNo("\nWould you like to download it now?", true) {
		downloadInput := &aws.DownloadInput{
			Bucket: env.S3.Bucket,
			Key:    env.GetS3Path(),
		}

		output, err := utils.GetAWSClient().DownloadFile(ctx, downloadInput)
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

		fmt.Printf("Successfully downloaded tfvars file to %s\n", env.Local.TFVarsPath)
	} else {
		fmt.Printf("Action needed: Use 'tfvarenv download %s' when ready to sync\n", env.Name)
	}

	return nil
}

// ローカルのみ存在する場合の処理
func handleLocalOnly(ctx context.Context, utils command.Utils, env *config.Environment) error {
	fmt.Println("Found local tfvars file but no remote file.")

	if prompt.PromptYesNo("Would you like to upload it now?", true) {
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
func handleBothExist(ctx context.Context, utils command.Utils, env *config.Environment) error {
	fmt.Println("Found tfvars files in both local and remote locations")

	// Download remote file to check content
	downloadInput := &aws.DownloadInput{
		Bucket: env.S3.Bucket,
		Key:    env.GetS3Path(),
	}
	_, err := utils.GetAWSClient().DownloadFile(ctx, downloadInput)
	if err != nil {
		return fmt.Errorf("failed to check remote file: %w", err)
	}

	// Calculate local file hash
	localHash, err := utils.GetFileUtils().CalculateHash(env.Local.TFVarsPath, nil)
	if err != nil {
		return fmt.Errorf("failed to calculate local file hash: %w", err)
	}

	// Compare contents
	remoteHash, err := utils.GetFileUtils().CalculateHash(env.Local.TFVarsPath, nil)
	if err != nil {
		return fmt.Errorf("failed to calculate remote file hash: %w", err)
	}

	if localHash == remoteHash {
		fmt.Println("Local and remote files are in sync")
	} else {
		fmt.Println("Warning: Local and remote files are different!")
		fmt.Printf("\nAction needed: Use one of the following commands to sync files:\n")
		fmt.Printf("  Download remote version: tfvarenv download %s\n", env.Name)
		fmt.Printf("  Upload local version:    tfvarenv upload %s\n", env.Name)
	}

	return nil
}
