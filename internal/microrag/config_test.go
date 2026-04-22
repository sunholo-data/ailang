package microrag

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig_MissingFileUsesDefaults(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
	if !cfg.Enabled {
		t.Error("expected default enabled=true")
	}
	if cfg.SessionBudget != defaultSessionBudget {
		t.Errorf("session_budget: got %d want %d", cfg.SessionBudget, defaultSessionBudget)
	}
	if len(cfg.Routes) == 0 || cfg.Routes[0].Glob != "**/*.ail" {
		t.Errorf("expected fallback **/*.ail route, got %+v", cfg.Routes)
	}
	if cfg.WindowFor("nonexistent-ns") != defaultWindow {
		t.Errorf("default window: got %d want %d", cfg.WindowFor("nonexistent-ns"), defaultWindow)
	}
}

func TestLoadConfig_ParsesYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "microrag.yaml")
	yaml := `enabled: false
routes:
  - glob: "**/*.go"
    kb: project-resolutions
    max_tokens_per_injection: 200
    relevance_floor: 0.25
  - glob: "**/CLAUDE.md"
    kb: skip
dedup:
  windows:
    project-resolutions: 40000
  relevance_bypass:
    project-resolutions: 0.65
session_budget: 7000
marker_style: ascii
`
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Enabled {
		t.Error("expected enabled=false from yaml")
	}
	if cfg.SessionBudget != 7000 {
		t.Errorf("session_budget: got %d want 7000", cfg.SessionBudget)
	}
	if cfg.MarkerStyle != "ascii" {
		t.Errorf("marker_style: got %q", cfg.MarkerStyle)
	}
	if cfg.WindowFor("project-resolutions") != 40000 {
		t.Errorf("project-resolutions window: got %d", cfg.WindowFor("project-resolutions"))
	}
	if cfg.BypassFor("project-resolutions") != 0.65 {
		t.Errorf("project-resolutions bypass: got %v", cfg.BypassFor("project-resolutions"))
	}
	if cfg.WindowFor("unknown") != defaultWindow {
		t.Errorf("unknown ns should fall back to default")
	}
}

func TestMatchRoute(t *testing.T) {
	cfg := (&Config{
		Routes: []Route{
			{Glob: "**/*.ail", KB: "ailang-syntax"},
			{Glob: "**/CLAUDE.md", KB: "skip"},
			{Glob: "**/*.go", KB: "project-resolutions"},
		},
	}).applyDefaults()

	cases := []struct {
		path string
		want string
	}{
		{"examples/foo.ail", "ailang-syntax"},
		{"deep/nested/path/bar.ail", "ailang-syntax"},
		{"CLAUDE.md", "skip"},
		{"sub/dir/CLAUDE.md", "skip"},
		{"internal/foo/bar.go", "project-resolutions"},
		{"README.md", ""}, // no match
	}
	for _, c := range cases {
		got := cfg.MatchRoute(c.path)
		switch {
		case c.want == "" && got != nil:
			t.Errorf("path=%q expected no match, got %+v", c.path, got)
		case c.want != "" && got == nil:
			t.Errorf("path=%q expected match %q, got nil", c.path, c.want)
		case got != nil && got.KB != c.want:
			t.Errorf("path=%q kb=%q want=%q", c.path, got.KB, c.want)
		}
	}
}

func TestMatchGlobEdgeCases(t *testing.T) {
	if !matchGlob("**/*.ail", "foo.ail") {
		t.Error("**/*.ail should match top-level foo.ail")
	}
	if !matchGlob("**/*.ail", "a/b/c/foo.ail") {
		t.Error("**/*.ail should match deeply nested foo.ail")
	}
	if matchGlob("**/*.ail", "foo.go") {
		t.Error("**/*.ail must not match foo.go")
	}
	if !strings.Contains("**/*.ail", "**") {
		t.Fatal("test invariant")
	}
}
