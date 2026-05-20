package eval_harness

import (
	"strings"
	"testing"
)

// TestGetExecutorForModel_GeminiRetired verifies the M-MANAGED-AGENTS (v0.22.0)
// validation hook: any model still requesting agent_cli: "gemini" gets a
// rejected with a clear next-step error message pointing at managed_agents.
//
// Gemini CLI was retired in v0.22.0 because Google deprecates it on 2026-06-18.
// All models.yml entries had their agent_cli: "gemini" stripped in M1, but a
// vendored or out-of-tree config could still try to use it — this validation
// gives those callers an actionable failure rather than a "unknown executor".
func TestGetExecutorForModel_GeminiRetired(t *testing.T) {
	cfg := &ModelsConfig{
		Models: map[string]ModelConfig{
			"legacy-gemini-config": {
				APIName:  "gemini-3-flash-preview",
				Provider: "google",
				AgentCLI: stringPtr("gemini"),
			},
		},
	}

	_, _, err := cfg.GetExecutorForModel("legacy-gemini-config")
	if err == nil {
		t.Fatal("expected error for agent_cli: gemini, got nil")
	}

	// The error message must mention the M-MANAGED-AGENTS replacement path
	// so a future user fixing a stale config has a clear next step.
	msg := err.Error()
	mustContain := []string{
		"retired in AILANG v0.22.0",
		"managed_agents",
		"gemini-3-5-flash",
	}
	for _, want := range mustContain {
		if !strings.Contains(msg, want) {
			t.Errorf("error message missing %q hint:\n  got: %s", want, msg)
		}
	}
}

// stringPtr returns a pointer to its string argument. Local helper for
// constructing in-memory ModelConfig values in tests.
func stringPtr(s string) *string { return &s }
