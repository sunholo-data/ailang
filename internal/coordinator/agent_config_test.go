package coordinator

import (
	"os"
	"path/filepath"
	"testing"
)

// The path resolution (AILANG_CONFIG, else ~/.ailang/config.yaml) lives in
// internal/config and is tested there; what this package owns is that a
// config file which exists but does not parse is an ERROR from every
// loader, never a silent default (M-V1-SIMPLIFY-S3 M3).
func TestLoadCoordinatorConfig_BrokenFileIsAnError(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("coordinator: [unclosed\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_CONFIG", configPath)

	if _, err := LoadCoordinatorConfig(); err == nil {
		t.Error("LoadCoordinatorConfig: a broken file must be an error, not the default config")
	}
	if _, err := LoadBudgetsConfig(); err == nil {
		t.Error("LoadBudgetsConfig: a broken file must be an error, not the default budgets")
	}
	if _, err := LoadCoordinatorConfigFrom(configPath); err == nil {
		t.Error("LoadCoordinatorConfigFrom: a broken file must be an error")
	}
}

func TestLoadCoordinatorConfig_NoFileIsTheDefault(t *testing.T) {
	t.Setenv("AILANG_CONFIG", filepath.Join(t.TempDir(), "absent.yaml"))
	cfg, err := LoadCoordinatorConfig()
	if err != nil || cfg.DefaultProvider != "claude" {
		t.Fatalf("no file: cfg=%+v err=%v; want the default config", cfg, err)
	}
	if b, err := LoadBudgetsConfig(); err != nil || b.Global == nil {
		t.Fatalf("no file: budgets=%+v err=%v", b, err)
	}
	if fb := LoadFirebaseConfig(); fb != nil {
		t.Fatalf("no file: firebase=%+v, want nil", fb)
	}
	if ws := LoadWorkspacesConfig(); ws == nil || len(ws.Mappings) == 0 {
		t.Fatalf("no file: workspaces=%+v, want defaults", ws)
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
