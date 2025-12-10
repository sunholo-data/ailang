package messaging

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// GitHubConfig holds configuration for GitHub integration
type GitHubConfig struct {
	// DefaultRepo is the default repository for --github flag (e.g., "sunholo-data/ailang")
	DefaultRepo string `yaml:"default_repo"`

	// ExpectedUser is the expected GitHub username (must match gh auth status)
	// This prevents accidentally creating issues under the wrong account
	ExpectedUser string `yaml:"expected_user"`

	// CreateLabels are labels to add when creating issues
	CreateLabels []string `yaml:"create_labels"`

	// WatchLabels are labels to watch for incoming messages (import)
	WatchLabels []string `yaml:"watch_labels"`

	// AutoImport enables automatic import on session start (default: true)
	AutoImport *bool `yaml:"auto_import"`
}

// IsAutoImportEnabled returns whether auto-import is enabled (defaults to true)
func (c *GitHubConfig) IsAutoImportEnabled() bool {
	if c.AutoImport == nil {
		return true // default
	}
	return *c.AutoImport
}

// Config holds the full AILANG configuration
type Config struct {
	GitHub *GitHubConfig `yaml:"github"`
}

// GetConfigPath returns the path to the AILANG config file
func GetConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".ailang", "config.yaml")
}

// LoadConfig loads configuration from ~/.ailang/config.yaml
// Returns nil if the config file doesn't exist (not an error)
func LoadConfig() (*Config, error) {
	configPath := GetConfigPath()
	if configPath == "" {
		return nil, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Config file doesn't exist, not an error
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// LoadGitHubConfig loads just the GitHub configuration
// Returns nil if GitHub config is not present (not an error)
func LoadGitHubConfig() (*GitHubConfig, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, nil
	}
	return config.GitHub, nil
}

// ValidateGitHubConfig validates the GitHub configuration
// Returns an error if required fields are missing
func ValidateGitHubConfig(config *GitHubConfig) error {
	if config == nil {
		return fmt.Errorf("github configuration not found in ~/.ailang/config.yaml")
	}

	if config.ExpectedUser == "" {
		return fmt.Errorf("github.expected_user is required in ~/.ailang/config.yaml")
	}

	if config.DefaultRepo == "" {
		return fmt.Errorf("github.default_repo is required in ~/.ailang/config.yaml")
	}

	return nil
}

// EnsureConfigDir creates the ~/.ailang directory if it doesn't exist
func EnsureConfigDir() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".ailang")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	return nil
}

// WriteExampleConfig writes an example config file if none exists
// Returns true if the file was created, false if it already exists
func WriteExampleConfig() (bool, error) {
	configPath := GetConfigPath()
	if configPath == "" {
		return false, fmt.Errorf("failed to determine config path")
	}

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		return false, nil // Already exists
	}

	if err := EnsureConfigDir(); err != nil {
		return false, err
	}

	exampleConfig := `# AILANG Configuration
# See: https://sunholo-data.github.io/ailang/docs/guides/configuration

github:
  # Default repository for --github flag
  default_repo: sunholo-data/ailang

  # REQUIRED: Expected GitHub username (must match 'gh auth status')
  # This prevents accidentally creating issues under wrong account
  expected_user: YOUR_GITHUB_USERNAME

  # Labels to add when creating issues
  create_labels:
    - ailang-message
    - from-agent

  # Labels to watch for incoming messages
  watch_labels:
    - ailang-message
    - needs-agent-response

  # Auto-import GitHub issues on session start (default: true)
  auto_import: true
`

	if err := os.WriteFile(configPath, []byte(exampleConfig), 0644); err != nil {
		return false, fmt.Errorf("failed to write config file: %w", err)
	}

	return true, nil
}
