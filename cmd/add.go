package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"tfvarenv/config"
	"tfvarenv/utils"

	"github.com/spf13/cobra"
)

func NewAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add",
		Short: "Add a new environment",
		Run:   addEnvironment,
	}
}

func addEnvironment(cmd *cobra.Command, args []string) {
	// Check if tfvarenv is initialized
	if _, err := os.Stat(".tfvarenv.json"); err != nil {
		if os.IsNotExist(err) {
			fmt.Println("tfvarenv is not initialized. Please run 'tfvarenv init' first.")
			os.Exit(1)
		}
		fmt.Printf("Error checking .tfvarenv.json: %v\n", err)
		os.Exit(1)
	}

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
	if err := utils.CheckS3BucketVersioning(bucket); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	env.S3 = config.EnvironmentS3Config{
		Bucket:    bucket,
		Prefix:    prefix,
		TFVarsKey: tfvarsKey,
	}

	// AWS Configuration
	fmt.Println("\nAWS Configuration:")

	// Get default region from config
	defaultRegion, err := config.GetDefaultRegion()
	if err != nil {
		fmt.Printf("Error getting default region: %v\n", err)
		os.Exit(1)
	}

	// AWS Configuration
	fmt.Printf("\nRegion [%s]: ", defaultRegion)
	region, _ := reader.ReadString('\n')
	region = strings.TrimSpace(region)
	if region == "" {
		region = defaultRegion
	}

	// Get AWS Account ID using the specified region
	accountID, err := utils.GetAWSAccountID(region)
	if err != nil {
		fmt.Printf("Error getting AWS account ID: %v\n", err)
		os.Exit(1)
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
		AutoBackup:      promptYesNo("Enable auto backup?", true),
		RequireApproval: promptYesNo("Require deployment approval?", envName != "dev"),
	}

	// Setup local environment
	fmt.Print("\nSetting up local environment")
	if err := setupLocalEnvironment(env); err != nil {
		fmt.Printf("Error setting up local environment: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("done")

	// Add environment to configuration
	fmt.Print("\nAdding environment to .tfvarenv.json")
	if err := config.AddEnvironment(envName, env); err != nil {
		fmt.Printf("Error adding environment: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(": done")

	// Check file status and provide guidance
	if err := checkFilesStatus(env); err != nil {
		fmt.Printf("Warning: Failed to check file status: %v\n", err)
	} else {
		fmt.Println("\nUse the following commands to manage tfvars:")
		fmt.Printf("- Download: tfvarenv download %s\n", envName)
		fmt.Printf("- Upload:   tfvarenv upload %s\n", envName)
		fmt.Printf("- Plan:     tfvarenv plan %s\n", envName)
		fmt.Printf("- Apply:    tfvarenv apply %s\n", envName)
	}

	fmt.Printf("\nEnvironment '%s' added successfully.\n", envName)
}

func setupLocalEnvironment(env *config.Environment) error {
	// Create directory structure
	dir := filepath.Dir(env.Local.TFVarsPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Update .gitignore
	if err := updateGitignore(); err != nil {
		return fmt.Errorf("failed to update .gitignore: %w", err)
	}

	return nil
}

func updateGitignore() error {
	entries := []string{
		"*.tfvars",
	}

	// まず、ファイルの内容を読み込む
	content := []string{}
	existing := make(map[string]bool)

	if file, err := os.Open(".gitignore"); err == nil {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			content = append(content, line)
			existing[line] = true
		}
		file.Close()
	}

	// 新しいエントリーを追加
	needsNewline := len(content) > 0 && len(content[len(content)-1]) > 0
	newEntries := false

	for _, entry := range entries {
		if !existing[entry] {
			if needsNewline {
				content = append(content, "")
				needsNewline = false
			}
			content = append(content, entry)
			newEntries = true
		}
	}

	// 変更がある場合のみファイルを書き直す
	if newEntries {
		return os.WriteFile(".gitignore", []byte(strings.Join(content, "\n")+"\n"), 0644)
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
	return input == "y" || input == "yes" || input == "Y"
}

func checkFilesStatus(env *config.Environment) error {
	// ローカルファイルの存在確認
	localExists := false
	if _, err := os.Stat(env.Local.TFVarsPath); err == nil {
		localExists = true
	}

	// リモートファイルの存在確認
	remoteExists := false
	versionInfo, err := utils.GetLatestS3Version(env.S3.Bucket, env.GetS3Path(), env.AWS.Region)
	if err == nil {
		remoteExists = true
	}

	fmt.Println("\nFile Status Check:")

	switch {
	case !localExists && !remoteExists:
		// 両方なし: 空のファイルを作成
		fmt.Println("No tfvars file found in either location.")

		// ディレクトリの作成
		dir := filepath.Dir(env.Local.TFVarsPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		// 空のファイルを作成
		if err := os.WriteFile(env.Local.TFVarsPath, []byte(""), 0644); err != nil {
			return fmt.Errorf("failed to create empty tfvars file: %w", err)
		}

		fmt.Printf("Created empty tfvars file at: %s\n", env.Local.TFVarsPath)
		fmt.Println("Action needed: Edit the tfvars file and use 'tfvarenv upload' to sync.")

	case !localExists && remoteExists:
		// リモートのみ存在: ダウンロードを提案
		fmt.Printf("Found remote tfvars file (Version: %s)\n", versionInfo.VersionID)
		if promptYesNo("Would you like to download it now?", true) {
			if err := downloadTFVars(env.Name, ""); err != nil {
				return fmt.Errorf("failed to download tfvars: %w", err)
			}
			fmt.Println("Successfully downloaded tfvars file.")
		} else {
			fmt.Println("Action needed: Use 'tfvarenv download' when ready to sync.")
		}

	case localExists && !remoteExists:
		// ローカルのみ存在: アップロードを提案
		fmt.Println("Found local tfvars file but no remote file.")
		if promptYesNo("Would you like to upload it now?", true) {
			if err := uploadTFVars(env.Name, "first upload"); err != nil {
				return fmt.Errorf("failed to upload tfvars: %w", err)
			}
			fmt.Println("Successfully uploaded tfvars file.")
		} else {
			fmt.Println("Action needed: Use 'tfvarenv upload' when ready to sync.")
		}

	case localExists && remoteExists:
		// 両方存在: ハッシュ比較
		localHash, err := utils.CalculateFileHash(env.Local.TFVarsPath)
		if err != nil {
			return fmt.Errorf("failed to calculate local file hash: %w", err)
		}

		if localHash == versionInfo.Hash {
			fmt.Println("Local and remote files are in sync.")
		} else {
			fmt.Println("Warning: Local and remote files are different!")
			fmt.Printf("Remote version: %s (Last modified: %s)\n",
				versionInfo.VersionID, versionInfo.Timestamp.Format("2006-01-02 15:04:05"))
			fmt.Println("Action needed: Use 'tfvarenv download' or 'tfvarenv upload' to sync files.")
		}
	}

	return nil
}
