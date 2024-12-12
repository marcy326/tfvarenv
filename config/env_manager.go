package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var (
	config     Config
	configLock sync.Mutex
	configPath string
)

// init function should also initialize the map
func init() {
	cwd, err := os.Getwd()
	if err != nil {
		panic("could not determine current working directory")
	}
	configPath = filepath.Join(cwd, configFileName)

	// Initialize empty config with default values
	config = Config{
		Version:       defaultVersion,
		DefaultRegion: "ap-northeast-1",
		S3: S3Config{
			Versioning:     true,
			MetadataSuffix: defaultMetadataExt,
		},
		Environments: make(map[string]Environment),
	}

	// Try to load existing config
	if err := loadConfig(); err != nil {
		if !os.IsNotExist(err) {
			panic(fmt.Sprintf("failed to load config: %v", err))
		}
	}

	// Ensure Environments map is initialized even after loading
	if config.Environments == nil {
		config.Environments = make(map[string]Environment)
	}
}

func loadConfig() error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	// Initialize temporary config with empty map
	var tmpConfig Config
	tmpConfig.Environments = make(map[string]Environment)

	if err := json.Unmarshal(data, &tmpConfig); err != nil {
		return err
	}

	config = tmpConfig
	return nil
}

// saveConfig is the public function that uses locking
func saveConfig() error {
	configLock.Lock()
	defer configLock.Unlock()
	return saveConfigDirect()
}

// saveConfigDirect is the internal function without locking
func saveConfigDirect() error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(configPath, append(data, '\n'), 0644)
}

// SaveConfigToFile saves the configuration to a specified file
func SaveConfigToFile(file *os.File, config *Config) error {
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(config)
}

// InitConfig initializes a new configuration file
func InitConfig() error {
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("configuration file already exists at %s", configPath)
	}

	config = Config{
		Version:       defaultVersion,
		DefaultRegion: "ap-northeast-1",
		S3: S3Config{
			Versioning:     true,
			MetadataSuffix: defaultMetadataExt,
		},
		Environments: make(map[string]Environment),
	}

	return saveConfig()
}

// AddEnvironment adds a new environment to the configuration
func AddEnvironment(name string, env *Environment) error {
	configLock.Lock()
	defer configLock.Unlock()

	// Initialize the Environments map if it's nil
	if config.Environments == nil {
		config.Environments = make(map[string]Environment)
	}

	// Check if environment already exists
	if _, exists := config.Environments[name]; exists {
		return fmt.Errorf("environment '%s' already exists", name)
	}

	// Validate environment configuration
	if err := validateEnvironment(env); err != nil {
		return fmt.Errorf("invalid environment configuration: %w", err)
	}

	// Add the environment to the configuration
	config.Environments[name] = *env
	return saveConfigDirect()
}

// GetEnvironmentInfo retrieves environment information by name
func GetEnvironmentInfo(envName string) (*Environment, error) {
	env, exists := config.Environments[envName]
	if !exists {
		return nil, errors.New("environment not found")
	}
	return &env, nil
}

// ListEnvironments returns a list of all environment names
func ListEnvironments() ([]string, error) {
	if len(config.Environments) == 0 {
		return nil, errors.New("no environments found")
	}

	var envs []string
	for name := range config.Environments {
		envs = append(envs, name)
	}
	return envs, nil
}

// validateEnvironment performs validation of environment configuration
func validateEnvironment(env *Environment) error {
	if env.Name == "" {
		return errors.New("environment name is required")
	}

	if env.S3.Bucket == "" {
		return errors.New("S3 bucket is required")
	}

	if env.S3.Prefix == "" {
		return errors.New("S3 prefix is required")
	}

	if env.S3.TFVarsKey == "" {
		env.S3.TFVarsKey = "terraform.tfvars"
	}

	if env.AWS.AccountID == "" {
		return errors.New("AWS account ID is required")
	}

	if env.AWS.Region == "" {
		env.AWS.Region = config.DefaultRegion
	}

	if env.Local.TFVarsPath == "" {
		env.Local.TFVarsPath = filepath.Join("environments", env.Name, "terraform.tfvars")
	}

	return nil
}

// GetDefaultRegion returns the default region from the configuration
func GetDefaultRegion() (string, error) {
	if config.DefaultRegion == "" {
		return "ap-northeast-1", nil
	}
	return config.DefaultRegion, nil
}
