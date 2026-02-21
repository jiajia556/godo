package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ConfigManager is a generic config manager that loads/saves typed config data.
type ConfigManager[T any] struct {
	data T
	path string
}

// NewManager creates a new ConfigManager.
func NewManager[T any]() *ConfigManager[T] {
	var zero T
	return &ConfigManager[T]{data: zero}
}

// Load reads and unmarshals the config file (JSON/YAML) into the manager.
func (cm *ConfigManager[T]) Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &cm.data); err != nil {
			return fmt.Errorf("failed to parse json: %w", err)
		}

	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &cm.data); err != nil {
			return fmt.Errorf("failed to parse yaml: %w", err)
		}

	default:
		return fmt.Errorf("unsupported format: %s", ext)
	}

	cm.path = path
	return nil
}

// Save marshals and writes the config to the given path.
func (cm *ConfigManager[T]) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	var (
		data []byte
		err  error
	)

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		data, err = json.MarshalIndent(cm.data, "", "  ")
	case ".yaml", ".yml":
		data, err = yaml.Marshal(cm.data)
	default:
		return fmt.Errorf("unsupported format: %s", ext)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	cm.path = path
	return nil
}

// Get returns the current config data.
func (cm *ConfigManager[T]) Get() T {
	return cm.data
}

// Set replaces the current config data.
func (cm *ConfigManager[T]) Set(data T) {
	cm.data = data
}

// Path returns the currently loaded config file path.
func (cm *ConfigManager[T]) Path() string {
	return cm.path
}

type MysqlConfig struct {
	Host     string `json:"host" yaml:"host"`
	User     string `json:"user" yaml:"user"`
	Password string `json:"password" yaml:"password"`
	DBName   string `json:"db_name" yaml:"db_name"`
	Port     int    `json:"port" yaml:"port"`
	Prefix   string `json:"prefix" yaml:"prefix"`
	Charset  string `json:"charset" yaml:"charset"`
}

type Config struct {
	Mysql MysqlConfig `json:"mysql" yaml:"mysql"`
}

var cfg *ConfigManager[Config]

func LoadConfig(path string) error {
	cfg = NewManager[Config]()
	return cfg.Load(path)
}

func GetConfig() *Config {
	res := cfg.Get()
	return &res
}
