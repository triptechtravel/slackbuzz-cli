package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// AuthConfig stores authentication state on disk as a fallback when the OS keyring is unavailable.
type AuthConfig struct {
	BotToken    string `yaml:"bot_token,omitempty"`
	UserToken   string `yaml:"user_token,omitempty"`
	ConfigToken string `yaml:"config_token,omitempty"` // App configuration token for manifest API
	TeamID      string `yaml:"team_id,omitempty"`
	TeamName    string `yaml:"team_name,omitempty"`
	UserID      string `yaml:"user_id,omitempty"`      // Human user ID (from user token)
	UserName    string `yaml:"user_name,omitempty"`    // Human username (from user token)
	BotUserID   string `yaml:"bot_user_id,omitempty"`  // Bot user ID (from bot token)
	BotUserName string `yaml:"bot_user_name,omitempty"` // Bot username (from bot token)
}

// AuthFile returns the path to the auth config file.
func AuthFile() string {
	return filepath.Join(ConfigDir(), "auth.yml")
}

// LoadAuth reads the auth config from disk.
func LoadAuth() (*AuthConfig, error) {
	ac := &AuthConfig{}

	data, err := os.ReadFile(AuthFile())
	if err != nil {
		if os.IsNotExist(err) {
			return ac, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, ac); err != nil {
		return nil, err
	}
	return ac, nil
}

// Save writes the auth config to disk.
func (a *AuthConfig) Save() error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := yaml.Marshal(a)
	if err != nil {
		return err
	}

	return os.WriteFile(AuthFile(), data, 0o600)
}

// Clear removes the stored auth config.
func (a *AuthConfig) Clear() error {
	a.BotToken = ""
	a.UserToken = ""
	a.TeamID = ""
	a.TeamName = ""
	a.UserID = ""
	a.UserName = ""
	a.BotUserID = ""
	a.BotUserName = ""
	return os.Remove(AuthFile())
}
