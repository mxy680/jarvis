package daemon

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds the daemon configuration
type Config struct {
	ClaudeAPIKey       string `yaml:"claude_api_key"`
	PorcupineAccessKey string `yaml:"porcupine_access_key"`
	WakeWord           string `yaml:"wake_word"`
	Voice              string `yaml:"voice"`
	TerminalApp        string `yaml:"terminal_app"`
	SummaryMaxWords    int    `yaml:"summary_max_words"`
	STTTimeoutSeconds  int    `yaml:"stt_timeout_seconds"`
	SocketPath         string `yaml:"socket_path,omitempty"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		WakeWord:          "jarvis",
		Voice:             "Samantha",
		TerminalApp:       "Terminal",
		SummaryMaxWords:   100,
		STTTimeoutSeconds: 30,
	}
}

// DefaultConfigPath returns the default config file path
func DefaultConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".jarvis", "config.yaml")
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	config := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil // Return defaults if config doesn't exist
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

// SaveConfig saves configuration to a YAML file
func SaveConfig(config *Config, path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate checks if the configuration has required fields
func (c *Config) Validate() error {
	if c.ClaudeAPIKey == "" {
		return fmt.Errorf("claude_api_key is required")
	}
	if c.PorcupineAccessKey == "" {
		return fmt.Errorf("porcupine_access_key is required")
	}
	return nil
}
