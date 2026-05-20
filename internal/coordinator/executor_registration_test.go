package coordinator

import "testing"

// TestExecutorRegistration_AutoDiscovery verifies that all blank-imported
// executor packages register themselves with the global factory at init time,
// so NewExecutorProvider() resolves each name without any coordinator code change.
// This is the M-EXEC-EXPAND guarantee documented in docs/internal/EXECUTOR_SHAPE.md.
// Note: Gemini CLI was retired in v0.22.0 (M-MANAGED-AGENTS) and is no longer
// expected in this set.
func TestExecutorRegistration_AutoDiscovery(t *testing.T) {
	for _, name := range []string{"claude", "codex", "opencode", "motoko", "pi", "managed_agents"} {
		t.Run(name, func(t *testing.T) {
			p, err := NewExecutorProvider(name)
			if err != nil {
				t.Fatalf("NewExecutorProvider(%q) failed: %v", name, err)
			}
			if p.ExecutorName() != name {
				t.Errorf("ExecutorName()=%q, want %q", p.ExecutorName(), name)
			}
			if p.Name() != name+"-cli" {
				t.Errorf("Name()=%q, want %q", p.Name(), name+"-cli")
			}
		})
	}
}
