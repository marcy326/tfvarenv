package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const (
	configFileName = ".tfvarenv.json"
	defaultVersion = "1.0"
)

// Manager インターフェースの定義
type Manager interface {
	GetEnvironment(name string) (*Environment, error)
	AddEnvironment(name string, env *Environment) error
	ListEnvironments() ([]string, error)
	GetDefaultRegion() (string, error)
	Save() error
}

type manager struct {
	config     Config
	configPath string
	mu         sync.RWMutex
}

// NewManager creates a new config manager instance
func NewManager() (Manager, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("could not determine current working directory: %w", err)
	}

	m := &manager{
		configPath: filepath.Join(cwd, configFileName),
		config: Config{
			Version:       defaultVersion,
			DefaultRegion: "ap-northeast-1",
			Environments:  make(map[string]Environment),
		},
	}

	if err := m.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return m, nil
}

func (m *manager) load() error {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &m.config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return nil
}

func (m *manager) save() error {
	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(m.configPath, append(data, '\n'), 0644)
}

func (m *manager) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.save()
}

func (m *manager) GetEnvironment(name string) (*Environment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	env, exists := m.config.Environments[name]
	if !exists {
		return nil, fmt.Errorf("environment not found: %s", name)
	}

	return &env, nil
}

func (m *manager) AddEnvironment(name string, env *Environment) error {
	// ロックを取得する前に検証を行う
	if err := validateEnvironment(env); err != nil {
		return fmt.Errorf("invalid environment configuration: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock() // deferを使用してロックの解放を保証

	if _, exists := m.config.Environments[name]; exists {
		return fmt.Errorf("environment '%s' already exists", name)
	}

	m.config.Environments[name] = *env
	return m.save() // Save()をprivateのsave()に変更
}

func (m *manager) ListEnvironments() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.config.Environments) == 0 {
		return nil, errors.New("no environments found")
	}

	envs := make([]string, 0, len(m.config.Environments))
	for name := range m.config.Environments {
		envs = append(envs, name)
	}
	return envs, nil
}

func (m *manager) GetDefaultRegion() (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.config.DefaultRegion == "" {
		return "ap-northeast-1", nil
	}
	return m.config.DefaultRegion, nil
}
