package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	// DefaultHost is the default Bitbucket host
	DefaultHost = "bitbucket.org"

	// ConfigFileName is the name of the config file
	ConfigFileName = "config.yml"

	// HostsFileName is the name of the hosts file
	HostsFileName = "hosts.yml"
)

// Config represents the main configuration
type Config struct {
	GitProtocol      string `yaml:"git_protocol,omitempty"`
	Editor           string `yaml:"editor,omitempty"`
	Prompt           string `yaml:"prompt,omitempty"`
	Pager            string `yaml:"pager,omitempty"`
	Browser          string `yaml:"browser,omitempty"`
	HTTPTimeout      int    `yaml:"http_timeout,omitempty"`
	DefaultWorkspace string `yaml:"default_workspace,omitempty"`
}

// HostConfig represents per-host configuration
type HostConfig struct {
	Users       map[string]*UserConfig `yaml:"users,omitempty"`
	User        string                 `yaml:"user,omitempty"`
	GitProtocol string                 `yaml:"git_protocol,omitempty"`
}

// UserConfig represents per-user configuration
type UserConfig struct {
	// Token is stored in keyring, not in config file
	// This struct is here for future per-user settings
}

// HostsConfig represents the hosts.yml file structure
type HostsConfig map[string]*HostConfig

// ConfigDir returns the directory where config files are stored
func ConfigDir() (string, error) {
	// Check BB_CONFIG_DIR first
	if dir := os.Getenv("BB_CONFIG_DIR"); dir != "" {
		return dir, nil
	}

	// Check XDG_CONFIG_HOME
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "bb"), nil
	}

	// Default to ~/.config/bb
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}

	return filepath.Join(home, ".config", "bb"), nil
}

// EnsureConfigDir creates the config directory if it doesn't exist
func EnsureConfigDir() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("could not create config directory: %w", err)
	}

	return dir, nil
}

// LoadConfig loads the main config file
func LoadConfig() (*Config, error) {
	dir, err := ConfigDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(dir, ConfigFileName)

	// Return default config if file doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return defaultConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("could not read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("could not parse config file: %w", err)
	}

	return &config, nil
}

// SaveConfig saves the main config file
func SaveConfig(config *Config) error {
	dir, err := EnsureConfigDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(dir, ConfigFileName)

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("could not marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("could not write config file: %w", err)
	}

	return nil
}

// LoadHostsConfig loads the hosts config file
func LoadHostsConfig() (HostsConfig, error) {
	dir, err := ConfigDir()
	if err != nil {
		return nil, err
	}

	hostsPath := filepath.Join(dir, HostsFileName)

	// Return empty config if file doesn't exist
	if _, err := os.Stat(hostsPath); os.IsNotExist(err) {
		return make(HostsConfig), nil
	}

	data, err := os.ReadFile(hostsPath)
	if err != nil {
		return nil, fmt.Errorf("could not read hosts file: %w", err)
	}

	var hosts HostsConfig
	if err := yaml.Unmarshal(data, &hosts); err != nil {
		return nil, fmt.Errorf("could not parse hosts file: %w", err)
	}

	return hosts, nil
}

// SaveHostsConfig saves the hosts config file
func SaveHostsConfig(hosts HostsConfig) error {
	dir, err := EnsureConfigDir()
	if err != nil {
		return err
	}

	hostsPath := filepath.Join(dir, HostsFileName)

	data, err := yaml.Marshal(hosts)
	if err != nil {
		return fmt.Errorf("could not marshal hosts config: %w", err)
	}

	if err := os.WriteFile(hostsPath, data, 0600); err != nil {
		return fmt.Errorf("could not write hosts file: %w", err)
	}

	return nil
}

// GetActiveUser returns the active user for a host
func (h HostsConfig) GetActiveUser(host string) string {
	if hostConfig, ok := h[host]; ok {
		return hostConfig.User
	}
	return ""
}

// SetActiveUser sets the active user for a host
func (h HostsConfig) SetActiveUser(host, user string) {
	if _, ok := h[host]; !ok {
		h[host] = &HostConfig{
			Users: make(map[string]*UserConfig),
		}
	}
	h[host].User = user

	// Ensure user exists in users map
	if h[host].Users == nil {
		h[host].Users = make(map[string]*UserConfig)
	}
	if _, ok := h[host].Users[user]; !ok {
		h[host].Users[user] = &UserConfig{}
	}
}

// GetGitProtocol returns the git protocol for a host
func (h HostsConfig) GetGitProtocol(host string) string {
	if hostConfig, ok := h[host]; ok && hostConfig.GitProtocol != "" {
		return hostConfig.GitProtocol
	}
	return "ssh" // default to ssh
}

func defaultConfig() *Config {
	return &Config{
		GitProtocol: "ssh",
		Prompt:      "enabled",
		HTTPTimeout: 30,
	}
}

// AuthenticatedHosts returns a list of hosts that have authenticated users
func (h HostsConfig) AuthenticatedHosts() []string {
	var hosts []string
	for host, config := range h {
		if config.User != "" {
			hosts = append(hosts, host)
		}
	}
	return hosts
}

// GetDefaultWorkspace returns the default workspace from config
func GetDefaultWorkspace() (string, error) {
	config, err := LoadConfig()
	if err != nil {
		return "", err
	}
	return config.DefaultWorkspace, nil
}

// SetDefaultWorkspace sets the default workspace in config
func SetDefaultWorkspace(workspace string) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}
	config.DefaultWorkspace = workspace
	return SaveConfig(config)
}
