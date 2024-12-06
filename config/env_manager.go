package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

const configFileName = ".tfvarenv.yaml"

type Config struct {
	Environments []Environment `yaml:"environments"`
}

type Environment struct {
	Name      string `yaml:"name"`
	S3Key     string `yaml:"s3_key"`
	LocalFile string `yaml:"local_file"`
	AccountID string `yaml:"account_id"`
	Region    string `yaml:"region"`
}

var (
	config     Config
	configPath string
	configLock sync.Mutex
)

// init initializes the configuration path and loads existing config if available.
func init() {
	cwd, err := os.Getwd()
	if err != nil {
		panic("could not determine current working directory")
	}
	configPath = filepath.Join(cwd, configFileName)
	if err := loadConfig(); err != nil && !os.IsNotExist(err) {
		panic(err)
	}
}

func loadConfig() error {
	file, err := os.Open(configPath)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	return decoder.Decode(&config)
}

func saveConfig() error {
	fmt.Println("Creating config file") // デバッグ出力

	file, err := os.Create(configPath)
	if err != nil {
		fmt.Println("Error creating config file:", err) // デバッグ出力
		return err
	}
	defer file.Close()

	fmt.Println("Encoding config to YAML") // デバッグ出力

	encoder := yaml.NewEncoder(file)
	return encoder.Encode(&config)
}

func AddEnvironment(envName, s3Key, localFile, accountID, region string) error {
	configLock.Lock()
	defer configLock.Unlock()

	fmt.Println("Checking if environment already exists") // デバッグ出力

	for _, env := range config.Environments {
		if env.Name == envName {
			fmt.Println("Environment already exists") // デバッグ出力
			return errors.New("environment already exists")
		}
	}

	fmt.Println("Adding new environment") // デバッグ出力

	config.Environments = append(config.Environments, Environment{
		Name:      envName,
		S3Key:     s3Key,
		LocalFile: localFile,
		AccountID: accountID,
		Region:    region,
	})

	fmt.Println("Saving config") // デバッグ出力
	return saveConfig()
}

func GetEnvironmentInfo(envName string) (*Environment, error) {
	for _, env := range config.Environments {
		if env.Name == envName {
			return &env, nil
		}
	}
	return nil, errors.New("environment not found")
}

func ListEnvironments() ([]string, error) {
	if len(config.Environments) == 0 {
		return nil, errors.New("no environments found")
	}

	var envs []string
	for _, env := range config.Environments {
		envs = append(envs, env.Name)
	}
	return envs, nil
}

func SaveConfigToFile(file *os.File, config *Config) error {
	encoder := yaml.NewEncoder(file)
	return encoder.Encode(config)
}
