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

func TestAgentConfig_WorkerTagsAndHostID_BackwardsCompat(t *testing.T) {
	// Config without worker_tags / worker_host_id should round-trip cleanly
	// and produce empty WorkerTags + empty WorkerHostID (match-all behavior).
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `coordinator:
  default_provider: claude
  agents:
    - id: legacy-agent
      label: "Legacy agent (no worker config)"
      inbox: legacy
      workspace: /tmp/ws
      provider: claude
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	orig := os.Getenv("AILANG_CONFIG")
	defer os.Setenv("AILANG_CONFIG", orig)
	os.Setenv("AILANG_CONFIG", configPath)

	cfg, err := LoadCoordinatorConfig()
	if err != nil {
		t.Fatalf("LoadCoordinatorConfig() error: %v", err)
	}
	if len(cfg.Agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(cfg.Agents))
	}
	a := cfg.Agents[0]
	if len(a.WorkerTags) != 0 {
		t.Errorf("WorkerTags = %v, want empty (backwards compat default)", a.WorkerTags)
	}
	if a.WorkerHostID != "" {
		t.Errorf("WorkerHostID = %q, want empty (backwards compat default)", a.WorkerHostID)
	}
	// ResolveHostID on an empty value should yield a non-empty fallback —
	// proves the helper is plumbed correctly without requiring a specific hostname.
	if got := ResolveHostID(a.WorkerHostID); got == "" {
		t.Errorf("ResolveHostID(empty) returned empty; expected fallback")
	}
}

func TestAgentConfig_WorkerTagsAndHostID_Explicit(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `coordinator:
  default_provider: claude
  agents:
    - id: eval-rig
      label: "Studio eval rig"
      inbox: eval-rig
      workspace: /tmp/ws
      provider: claude
      worker_host_id: studio.eval-rig
      worker_tags:
        - ollama:gemma4-26b-ailang
        - gpu:m4-max-40core
        - local-models
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	orig := os.Getenv("AILANG_CONFIG")
	defer os.Setenv("AILANG_CONFIG", orig)
	os.Setenv("AILANG_CONFIG", configPath)

	cfg, err := LoadCoordinatorConfig()
	if err != nil {
		t.Fatalf("LoadCoordinatorConfig() error: %v", err)
	}
	if len(cfg.Agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(cfg.Agents))
	}
	a := cfg.Agents[0]
	if a.WorkerHostID != "studio.eval-rig" {
		t.Errorf("WorkerHostID = %q, want studio.eval-rig", a.WorkerHostID)
	}
	wantTags := []string{"ollama:gemma4-26b-ailang", "gpu:m4-max-40core", "local-models"}
	if len(a.WorkerTags) != len(wantTags) {
		t.Fatalf("WorkerTags len = %d, want %d", len(a.WorkerTags), len(wantTags))
	}
	for i, tag := range wantTags {
		if a.WorkerTags[i] != tag {
			t.Errorf("WorkerTags[%d] = %q, want %q", i, a.WorkerTags[i], tag)
		}
	}
	// Sanity: ResolveHostID returns the explicit value.
	if got := ResolveHostID(a.WorkerHostID); got != "studio.eval-rig" {
		t.Errorf("ResolveHostID(explicit) = %q, want studio.eval-rig", got)
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
