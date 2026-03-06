package coordinator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfigPath_EnvOverride(t *testing.T) {
	// Save and restore env var
	orig := os.Getenv("AILANG_CONFIG")
	defer os.Setenv("AILANG_CONFIG", orig)

	// Set AILANG_CONFIG
	os.Setenv("AILANG_CONFIG", "/etc/ailang-config/config.yaml")
	got := defaultConfigPath()
	if got != "/etc/ailang-config/config.yaml" {
		t.Errorf("defaultConfigPath() = %q, want /etc/ailang-config/config.yaml", got)
	}
}

func TestDefaultConfigPath_DefaultFallback(t *testing.T) {
	// Save and restore env var
	orig := os.Getenv("AILANG_CONFIG")
	defer os.Setenv("AILANG_CONFIG", orig)

	// Unset AILANG_CONFIG
	os.Unsetenv("AILANG_CONFIG")
	got := defaultConfigPath()
	homeDir, _ := os.UserHomeDir()
	want := filepath.Join(homeDir, ".ailang", "config.yaml")
	if got != want {
		t.Errorf("defaultConfigPath() = %q, want %q", got, want)
	}
}

func TestLoadCoordinatorConfig_WithAILANGConfig(t *testing.T) {
	// Create a temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `coordinator:
  default_provider: gemini
  merge_branch: main
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Set AILANG_CONFIG to point to it
	orig := os.Getenv("AILANG_CONFIG")
	defer os.Setenv("AILANG_CONFIG", orig)
	os.Setenv("AILANG_CONFIG", configPath)

	cfg, err := LoadCoordinatorConfig()
	if err != nil {
		t.Fatalf("LoadCoordinatorConfig() error: %v", err)
	}
	if cfg.DefaultProvider != "gemini" {
		t.Errorf("DefaultProvider = %q, want gemini", cfg.DefaultProvider)
	}
	if cfg.MergeBranch != "main" {
		t.Errorf("MergeBranch = %q, want main", cfg.MergeBranch)
	}
}

func TestLoadCoordinatorConfig_MissingFileReturnsDefaults(t *testing.T) {
	orig := os.Getenv("AILANG_CONFIG")
	defer os.Setenv("AILANG_CONFIG", orig)

	os.Setenv("AILANG_CONFIG", "/nonexistent/path/config.yaml")
	cfg, err := LoadCoordinatorConfig()
	if err != nil {
		t.Fatalf("LoadCoordinatorConfig() error: %v", err)
	}
	// Should get default config, not nil or error
	if cfg == nil {
		t.Fatal("LoadCoordinatorConfig() returned nil for missing file")
	}
}
