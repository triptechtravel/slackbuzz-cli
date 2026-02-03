package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the user's CLI configuration.
type Config struct {
	DefaultWorkspace string `yaml:"default_workspace,omitempty"`
	DefaultChannel   string `yaml:"default_channel,omitempty"`
}

// ConfigDir returns the path to the config directory (~/.config/slack).
func ConfigDir() string {
	if dir := os.Getenv("SLACKBUZZ_CONFIG_DIR"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "slack")
}

// ConfigFile returns the path to the main config file.
func ConfigFile() string {
	return filepath.Join(ConfigDir(), "config.yml")
}

// Load reads the config from disk, returning defaults if the file doesn't exist.
func Load() (*Config, error) {
	cfg := &Config{}

	data, err := os.ReadFile(ConfigFile())
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Save writes the config to disk.
func (c *Config) Save() error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(ConfigFile(), data, 0o644)
}
