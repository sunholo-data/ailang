package coordinator

import "testing"

// TestExecutorRegistration_AutoDiscovery verifies that all blank-imported
// executor packages register themselves with the global factory at init time,
// so NewExecutorProvider() resolves each name without any coordinator code change.
// This is the M-EXEC-EXPAND guarantee documented in docs/internal/EXECUTOR_SHAPE.md.
func TestExecutorRegistration_AutoDiscovery(t *testing.T) {
	for _, name := range []string{"claude", "gemini", "codex", "opencode"} {
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
