package coordinator

import "testing"

// The startup audit exists because the dispatcher's guard, while correct, only
// speaks one agent at a time and only on demand. Two pipeline stages were
// undispatchable and nothing said so until a task was aimed at one.

func TestAudit_CatchesTheAgentsThatCouldNotDispatch(t *testing.T) {
	reg := NewAgentRegistry()
	for _, a := range []*AgentConfig{
		{ID: "sprint-planner", Inbox: "sprint-planner", Provider: "codex", ExecutorVariant: "codex"},
		{ID: "sprint-executor", Inbox: "sprint-executor", Provider: "codex", ExecutorVariant: "codex-go"},
		{ID: "design-doc-creator", Inbox: "design-doc-creator", Provider: "pi", ExecutorVariant: "pi"},
	} {
		if err := reg.Register(a); err != nil {
			t.Fatalf("register %s: %v", a.ID, err)
		}
	}

	// With the fix, the agent's own provider wins, so nothing is misconfigured.
	if got := AuditProviderMismatches(reg, "pi"); len(got) != 0 {
		t.Errorf("audit reported %d mismatches for a correct config: %v", len(got), got)
	}
}

// TestAudit_ReportsAnAgentWhoseCLICannotExistInItsImage is the real defect
// shape: a declaration the image cannot satisfy.
func TestAudit_ReportsAnAgentWhoseCLICannotExistInItsImage(t *testing.T) {
	reg := NewAgentRegistry()
	if err := reg.Register(&AgentConfig{
		ID: "broken", Inbox: "broken", Provider: "opencode", ExecutorVariant: "codex-go",
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	got := AuditProviderMismatches(reg, "pi")
	if len(got) != 1 {
		t.Fatalf("audit found %d mismatches, want 1: %v", len(got), got)
	}
	if got[0].AgentID != "broken" || got[0].Provider != "opencode" {
		t.Errorf("wrong mismatch reported: %v", got[0])
	}
}

// TestAudit_UsesTheSameResolutionAsDispatch: auditing the declared config rather
// than the EFFECTIVE provider would miss the exact bug this exists for — an
// agent declaring nothing and inheriting a plane default its image cannot run.
func TestAudit_UsesTheSameResolutionAsDispatch(t *testing.T) {
	reg := NewAgentRegistry()
	if err := reg.Register(&AgentConfig{
		ID: "inherits", Inbox: "inherits", ExecutorVariant: "codex-go", // declares no provider
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	got := AuditProviderMismatches(reg, "pi")
	if len(got) != 1 {
		t.Fatalf("an agent inheriting an incompatible plane default was not reported: %v", got)
	}
	if got[0].Provider != "pi" {
		t.Errorf("reported provider %q, want the INHERITED \"pi\" — auditing the declaration instead of the effective value misses this entirely", got[0].Provider)
	}
}

func TestAudit_IgnoresTheAllCLIImage(t *testing.T) {
	reg := NewAgentRegistry()
	if err := reg.Register(&AgentConfig{
		ID: "evaluator", Inbox: "evaluator", Provider: "opencode", ExecutorVariant: "eval",
	}); err != nil {
		t.Fatalf("register: %v", err)
	}
	if got := AuditProviderMismatches(reg, "pi"); len(got) != 0 {
		t.Errorf("agent-eval carries every CLI and must never be reported: %v", got)
	}
}
