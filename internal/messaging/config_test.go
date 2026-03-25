package messaging

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLoadGitHubConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
github:
  default_repo: test-owner/test-repo
  expected_user: test-user
  create_labels:
    - label1
    - label2
  watch_labels:
    - watch1
  auto_import: false
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Read and parse the config
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	// Validate GitHub config
	if config.GitHub == nil {
		t.Fatal("expected GitHub config to be present")
	}

	if config.GitHub.DefaultRepo != "test-owner/test-repo" {
		t.Errorf("expected default_repo 'test-owner/test-repo', got '%s'", config.GitHub.DefaultRepo)
	}

	if config.GitHub.ExpectedUser != "test-user" {
		t.Errorf("expected expected_user 'test-user', got '%s'", config.GitHub.ExpectedUser)
	}

	if len(config.GitHub.CreateLabels) != 2 {
		t.Errorf("expected 2 create_labels, got %d", len(config.GitHub.CreateLabels))
	}

	if len(config.GitHub.WatchLabels) != 1 {
		t.Errorf("expected 1 watch_label, got %d", len(config.GitHub.WatchLabels))
	}

	if config.GitHub.IsAutoImportEnabled() {
		t.Error("expected auto_import to be false")
	}
}

func TestValidateGitHubConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    *GitHubConfig
		wantError bool
	}{
		{
			name:      "nil config",
			config:    nil,
			wantError: true,
		},
		{
			name: "missing expected_user",
			config: &GitHubConfig{
				DefaultRepo: "owner/repo",
			},
			wantError: true,
		},
		{
			name: "missing default_repo",
			config: &GitHubConfig{
				ExpectedUser: "user",
			},
			wantError: true,
		},
		{
			name: "valid config",
			config: &GitHubConfig{
				DefaultRepo:  "owner/repo",
				ExpectedUser: "user",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGitHubConfig(tt.config)
			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestIsAutoImportEnabled(t *testing.T) {
	// Default (nil) should be true
	config := &GitHubConfig{}
	if !config.IsAutoImportEnabled() {
		t.Error("expected default auto_import to be true")
	}

	// Explicit false
	falseVal := false
	config.AutoImport = &falseVal
	if config.IsAutoImportEnabled() {
		t.Error("expected auto_import false")
	}

	// Explicit true
	trueVal := true
	config.AutoImport = &trueVal
	if !config.IsAutoImportEnabled() {
		t.Error("expected auto_import true")
	}
}

func TestRepoForInbox(t *testing.T) {
	config := &GitHubConfig{
		DefaultRepo: "sunholo-data/ailang",
		InboxRepos: map[string]string{
			"http-helpers":        "sunholo-data/ailang-packages",
			"pkg:":                "sunholo-data/ailang-packages",
			"twilight-design-doc": "MarkEdmondson1234/TwilightGame",
		},
	}

	tests := []struct {
		inbox string
		want  string
	}{
		{"http-helpers", "sunholo-data/ailang-packages"},          // exact match
		{"pkg:sunholo/auth", "sunholo-data/ailang-packages"},      // prefix match
		{"pkg:sunholo/test_pkg", "sunholo-data/ailang-packages"},  // prefix match
		{"twilight-design-doc", "MarkEdmondson1234/TwilightGame"}, // exact match
		{"design-doc-creator", "sunholo-data/ailang"},             // fallback to default
		{"sprint-executor", "sunholo-data/ailang"},                // fallback to default
	}

	for _, tt := range tests {
		t.Run(tt.inbox, func(t *testing.T) {
			got := config.RepoForInbox(tt.inbox)
			if got != tt.want {
				t.Errorf("RepoForInbox(%q) = %q, want %q", tt.inbox, got, tt.want)
			}
		})
	}

	// Nil config edge case
	var nilConfig *GitHubConfig
	if got := nilConfig.RepoForInbox("anything"); got != "" {
		t.Errorf("nil config should return empty, got %q", got)
	}

	// No inbox_repos should return default
	noMapping := &GitHubConfig{DefaultRepo: "default/repo"}
	if got := noMapping.RepoForInbox("anything"); got != "default/repo" {
		t.Errorf("expected default/repo, got %q", got)
	}
}

func TestConfigWithoutGitHub(t *testing.T) {
	// Test parsing a config without github section
	configContent := `
other_setting: value
`

	var config Config
	if err := yaml.Unmarshal([]byte(configContent), &config); err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	if config.GitHub != nil {
		t.Error("expected GitHub config to be nil")
	}
}
