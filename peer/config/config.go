package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the application.
type Config struct {
	TrackerURL string `yaml:"tracker_url"`
	AuthToken  string `yaml:"auth_token"`
}

// defaultConfigFilePath returns the default path for the config file.
func DefaultConfigFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	configDir := filepath.Join(homeDir, ".peernet")
	return filepath.Join(configDir, "config.yaml"), nil
}

// Load loads the configuration from the specified path.
// If configFilePath is empty, it uses the default path.
func Load(configFilePath string) (*Config, error) {
	path := configFilePath
	if path == "" {
		var err error
		path, err = DefaultConfigFilePath()
		if err != nil {
			return nil, err
		}
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &Config{}, nil // Return an empty config if file doesn't exist
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config file %s: %w", path, err)
	}
	return &cfg, nil
}

// Save saves the configuration to the specified path.
// If configFilePath is empty, it uses the default path.
func (c *Config) Save(configFilePath string) error {
	path := configFilePath
	if path == "" {
		var err error
		path, err = DefaultConfigFilePath()
		if err != nil {
			return err
		}
	}

	configDir := filepath.Dir(path)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", configDir, err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", path, err)
	}
	return nil
}
