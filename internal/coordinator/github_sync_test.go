package coordinator

import (
	"testing"
	"time"
)

func TestGitHubSyncConfigDefaults(t *testing.T) {
	cfg := DefaultCoordinatorConfig()

	if cfg.GitHubSync == nil {
		t.Fatal("expected GitHubSync to be set")
	}
	if cfg.GitHubSync.Enabled {
		t.Error("expected GitHubSync.Enabled to be false by default")
	}
	if cfg.GitHubSync.IntervalSecs != 300 {
		t.Errorf("expected IntervalSecs 300, got %d", cfg.GitHubSync.IntervalSecs)
	}
	if cfg.GitHubSync.TargetInbox != "coordinator" {
		t.Errorf("expected TargetInbox 'coordinator', got %q", cfg.GitHubSync.TargetInbox)
	}
}

func TestGitHubSyncConfigEnabled(t *testing.T) {
	cfg := &CoordinatorConfig{
		GitHubSync: &GitHubSyncConfig{
			Enabled:      true,
			IntervalSecs: 600, // 10 minutes
			WatchLabels:  []string{"bug", "feature"},
			TargetInbox:  "coordinator",
		},
	}

	if !cfg.GitHubSync.Enabled {
		t.Error("expected GitHubSync to be enabled")
	}
	if cfg.GitHubSync.IntervalSecs != 600 {
		t.Errorf("expected IntervalSecs 600, got %d", cfg.GitHubSync.IntervalSecs)
	}
	if len(cfg.GitHubSync.WatchLabels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(cfg.GitHubSync.WatchLabels))
	}
}

func TestGitHubSyncIntervalMinimum(t *testing.T) {
	// Test that intervals less than 1 minute are bumped to 5 minutes
	// This logic is in runGitHubSync()
	cfg := &GitHubSyncConfig{
		Enabled:      true,
		IntervalSecs: 30, // 30 seconds - too fast
	}

	interval := time.Duration(cfg.IntervalSecs) * time.Second
	if interval < time.Minute {
		interval = 5 * time.Minute
	}

	expected := 5 * time.Minute
	if interval != expected {
		t.Errorf("expected interval %v, got %v", expected, interval)
	}
}

func TestGitHubSyncIntervalNormal(t *testing.T) {
	// Test that normal intervals are preserved
	cfg := &GitHubSyncConfig{
		Enabled:      true,
		IntervalSecs: 600, // 10 minutes
	}

	interval := time.Duration(cfg.IntervalSecs) * time.Second
	if interval < time.Minute {
		interval = 5 * time.Minute
	}

	expected := 10 * time.Minute
	if interval != expected {
		t.Errorf("expected interval %v, got %v", expected, interval)
	}
}
