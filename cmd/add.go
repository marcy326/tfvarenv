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
	fmt.Printf("Region [%s]: ", defaultRegion)
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
	fmt.Println(": done")

	// Backend Configuration
	if err := configureBackend(envName, env); err != nil {
		fmt.Printf("Error configuring backend: %v\n", err)
		return
	}

	// Add environment to configuration
	fmt.Print("\nAdding environment to .tfvarenv.json")
	if err := config.AddEnvironment(envName, env); err != nil {
		fmt.Printf("Error adding environment: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(": done")

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
	var versionInfo *utils.VersionInfo
	vInfo, err := utils.GetLatestS3Version(env.S3.Bucket, env.GetS3Path(), env.AWS.Region)
	if err == nil {
		remoteExists = true
		versionInfo = vInfo
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
		fmt.Printf("\nFound remote tfvars file:\n")
		fmt.Printf("  Version ID: %s\n", versionInfo.VersionID[:8])
		fmt.Printf("  Uploaded: %s\n", versionInfo.Timestamp.Format("2006-01-02 15:04:05"))
		if versionInfo.Description != "" {
			fmt.Printf("  Description: %s\n", versionInfo.Description)
		}

		if promptYesNo("\nWould you like to download it now?", true) {
			if err := utils.DownloadTFVars(env.Name, ""); err != nil {
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
			description := "Initial upload during environment setup"
			// ファイルをアップロード（S3のバージョンIDが生成される）
			if err := utils.UploadTFVars(env.Name, description); err != nil {
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
			fmt.Printf("\nCurrent version information:\n")
			fmt.Printf("  Version ID: %s\n", versionInfo.VersionID[:8])
			fmt.Printf("  Uploaded: %s\n", versionInfo.Timestamp.Format("2006-01-02 15:04:05"))
			if versionInfo.Description != "" {
				fmt.Printf("  Description: %s\n", versionInfo.Description)
			}
		} else {
			fmt.Println("Warning: Local and remote files are different!")
			fmt.Printf("\nRemote version information:\n")
			fmt.Printf("  Version ID: %s\n", versionInfo.VersionID[:8])
			fmt.Printf("  Uploaded: %s\n", versionInfo.Timestamp.Format("2006-01-02 15:04:05"))
			if versionInfo.Description != "" {
				fmt.Printf("  Description: %s\n", versionInfo.Description)
			}

			fmt.Println("\nAction needed: Use one of the following commands to sync files:")
			fmt.Printf("  Download remote version: tfvarenv download %s\n", env.Name)
			fmt.Printf("  Upload local changes:    tfvarenv upload %s\n", env.Name)
		}
	}

	// Get deployment status if remote exists
	if remoteExists {
		deployments, err := utils.ListDeployments(env.S3.Bucket, env.GetDeploymentHistoryKey(), env.AWS.Region)
		if err == nil && len(deployments) > 0 {
			// Find the latest deployment for the current version
			for _, d := range deployments {
				if d.VersionID == versionInfo.VersionID {
					fmt.Printf("\nDeployment Status: Last deployed on %s by %s\n",
						d.DeployedAt.Format("2006-01-02 15:04:05"), d.DeployedBy)
					break
				}
			}
		}
	}

	return nil
}

func configureBackend(envName string, env *config.Environment) error {
	fmt.Println("\nBackend Configuration:")
	defaultBackendPath := filepath.Join("envs", envName, "terraform.tfbackend")
	env.Backend = config.BackendConfig{
		ConfigPath: defaultBackendPath,
	}

	// Check if backend file already exists
	if _, err := os.Stat(defaultBackendPath); err == nil {
		fmt.Printf("Backend configuration file already exists at: %s\n", defaultBackendPath)
		fmt.Println("Please review and modify if needed.")
		return nil
	}

	// Create backend directory
	backendDir := filepath.Dir(defaultBackendPath)
	if err := os.MkdirAll(backendDir, 0755); err != nil {
		return fmt.Errorf("Error creating backend directory: %w", err)
	}

	fmt.Printf("Backend configuration file not found at: %s\n", defaultBackendPath)
	fmt.Print("Would you like to create it interactively? [Y/n]: ")
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response == "" || response == "y" || response == "yes" {
		// Interactive backend configuration
		fmt.Print("Enter bucket name: ")
		bucket, _ := reader.ReadString('\n')
		bucket = strings.TrimSpace(bucket)

		defaultStateKey := fmt.Sprintf("%s/terraform.tfstate", env.S3.Prefix)
		fmt.Printf("Enter state file key [%s]: ", defaultStateKey)
		key, _ := reader.ReadString('\n')
		key = strings.TrimSpace(key)
		if key == "" {
			key = defaultStateKey
		}

		fmt.Printf("Enter region [%s]: ", env.AWS.Region)
		region, _ := reader.ReadString('\n')
		region = strings.TrimSpace(region)
		if region == "" {
			region = env.AWS.Region
		}

		// Create backend configuration file
		content := fmt.Sprintf(`bucket = "%s"
key    = "%s"
region = "%s"`, bucket, key, region)

		if err := os.WriteFile(defaultBackendPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("Error creating backend configuration: %w", err)
		}
	} else {
		// Create empty file
		if err := os.WriteFile(defaultBackendPath, []byte(""), 0644); err != nil {
			return fmt.Errorf("Error creating empty backend configuration: %w", err)
		}
	}

	fmt.Printf("\nBackend configuration created at: %s\n", defaultBackendPath)
	fmt.Println("Please review and modify if needed before running 'tfvarenv use'")
	return nil
}
