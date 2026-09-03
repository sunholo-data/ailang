package coordinator

import "testing"

// The agent's declared provider must win over the plane default.
//
// Measured on the dev plane 2026-09-03: sprint-executor declares provider
// "codex" with executor_variant "codex-go"; the coordinator handed the
// dispatcher "pi" (the plane default) and the variant/provider guard correctly
// refused. The task retried every five minutes and never ran. Two of the four
// pipeline stages were undispatchable this way, so the pipeline could not have
// completed once regardless of what else was fixed.

func TestResolveDispatchProvider_AgentWinsOverPlaneDefault(t *testing.T) {
	agent := &AgentConfig{ID: "sprint-executor", Provider: "codex", ExecutorVariant: "codex-go"}

	got := ResolveDispatchProvider(agent, "pi")

	if got != "codex" {
		t.Errorf("provider = %q, want \"codex\" — the plane default overrode the agent that also chose the executor image, so the two can only disagree", got)
	}
}

func TestResolveDispatchProvider_FallsBackToPlaneDefault(t *testing.T) {
	// Most agents declare nothing and must keep their existing behaviour.
	agent := &AgentConfig{ID: "pkg-sunholo-auth"}

	if got := ResolveDispatchProvider(agent, "pi"); got != "pi" {
		t.Errorf("provider = %q, want \"pi\" for an agent that declares none", got)
	}
	if got := ResolveDispatchProvider(nil, "pi"); got != "pi" {
		t.Errorf("provider = %q, want \"pi\" for an unknown agent", got)
	}
}

func TestResolveDispatchProvider_LastResortIsClaude(t *testing.T) {
	if got := ResolveDispatchProvider(nil, ""); got != "claude" {
		t.Errorf("provider = %q, want \"claude\" when nothing is configured", got)
	}
}

// TestResolveDispatchProvider_TheThreeAgentsThatWereBlocked pins the specific
// configurations that could not dispatch, so a regression names them.
func TestResolveDispatchProvider_TheThreeAgentsThatWereBlocked(t *testing.T) {
	blocked := []struct {
		id       string
		provider string
		variant  string
	}{
		{"sprint-planner", "codex", "codex"},
		{"sprint-executor", "codex", "codex-go"},
		{"motoko", "motoko", "motoko"},
	}

	for _, b := range blocked {
		agent := &AgentConfig{ID: b.id, Provider: b.provider, ExecutorVariant: b.variant}
		if got := ResolveDispatchProvider(agent, "pi"); got != b.provider {
			t.Errorf("%s: provider = %q, want %q — this agent's variant %q requires it, and a mismatch is refused before the job starts",
				b.id, got, b.provider, b.variant)
		}
	}
}

// TestResolveDispatchProvider_RegistryLookup covers the call-site helper,
// including the nil-registry path the daemon can be in during startup.
func TestResolveDispatchProvider_RegistryLookup(t *testing.T) {
	reg := NewAgentRegistry()
	if err := reg.Register(&AgentConfig{ID: "sprint-executor", Inbox: "sprint-executor", Provider: "codex", ExecutorVariant: "codex-go"}); err != nil {
		t.Fatalf("register: %v", err)
	}
	cfg := &CoordinatorConfig{DefaultProvider: "pi"}

	if got := resolveDispatchProvider(reg, "sprint-executor", cfg); got != "codex" {
		t.Errorf("registered agent: provider = %q, want \"codex\"", got)
	}
	if got := resolveDispatchProvider(reg, "not-registered", cfg); got != "pi" {
		t.Errorf("unknown agent: provider = %q, want the plane default \"pi\"", got)
	}
	if got := resolveDispatchProvider(nil, "sprint-executor", cfg); got != "pi" {
		t.Errorf("nil registry: provider = %q, want the plane default \"pi\"", got)
	}
	if got := resolveDispatchProvider(reg, "sprint-executor", nil); got != "codex" {
		t.Errorf("nil config: provider = %q, want the agent's own \"codex\"", got)
	}
}
