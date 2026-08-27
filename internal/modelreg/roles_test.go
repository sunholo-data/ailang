package modelreg

import "testing"

// M-MODEL-REGISTRY-SINGLE-SOURCE M3.
//
// The chains are TRANSCRIBED from config.cloud.yaml's model_routing (the table
// M7 deleted). Changing which models a role runs is an explicit Non-Goal, so the
// test that matters is byte-identity with what the coordinator resolved before.
//
// A subscription-first reordering was drafted during M8 and reverted with it —
// see the note above `roles:` in models.yml. This assertion is what caught that
// the draft had made M7 non-inert on the role path.
func TestResolveRole_TranscribesLiveChainsByteIdentically(t *testing.T) {
	if err := InitModelsConfig(); err != nil {
		t.Fatalf("InitModelsConfig: %v", err)
	}
	c := GlobalModelsConfig

	// Verbatim from ailang-multivac/config/config.cloud.yaml:61-64 as measured
	// 2026-08-27 — the strings the coordinator handed an executor.
	want := map[string][]string{
		"designer":  {"openrouter/moonshotai/kimi-k3"},
		"planner":   {"gpt-5.6-sol", "openrouter/moonshotai/kimi-k3"},
		"executor":  {"gpt-5.6-sol", "openrouter/deepseek/deepseek-v4-flash-0731"},
		"evaluator": {"openrouter/minimax/minimax-m3", "openrouter/deepseek/deepseek-v4-flash-0731"},
	}

	for role, wantChain := range want {
		got, err := c.ResolveRole(role, LaneCloud)
		if err != nil {
			t.Errorf("ResolveRole(%q): %v", role, err)
			continue
		}
		if len(got) != len(wantChain) {
			t.Errorf("role %q: chain length %d, want %d (%v)", role, len(got), len(wantChain), got)
			continue
		}
		for i, w := range wantChain {
			if got[i].ModelName != w {
				t.Errorf("role %q entry %d: ModelName = %q, want %q", role, i, got[i].ModelName, w)
			}
			if got[i].Executor == "" {
				t.Errorf("role %q entry %d (%s): empty Executor", role, i, got[i].FriendlyName)
			}
		}
	}
}

// A role the registry does not know must name the ones it does. "The table has
// no entry for this" is a config gap, not a preference — the same fail-loud
// contract coordinator.ResolveModel already implements.
func TestResolveRole_MissingRoleNamesKnownRoles(t *testing.T) {
	if err := InitModelsConfig(); err != nil {
		t.Fatalf("InitModelsConfig: %v", err)
	}
	_, err := GlobalModelsConfig.ResolveRole("no-such-role", LaneCloud)
	if err == nil {
		t.Fatal("expected an error for an unknown role, got nil")
	}
	for _, known := range []string{"designer", "planner", "executor", "evaluator"} {
		if !contains(err.Error(), known) {
			t.Errorf("error should name known role %q so a human can fix the config; got: %v", known, err)
		}
	}
}

// Lane filtering is not decoration: ollama rows are the single shared GPU rig.
// Offering one to a Cloud Run job resolves to a model that host cannot reach.
func TestResolveRole_CloudLaneNeverReturnsLocalGPU(t *testing.T) {
	if err := InitModelsConfig(); err != nil {
		t.Fatalf("InitModelsConfig: %v", err)
	}
	c := GlobalModelsConfig

	// Control: the registry must actually CONTAIN local-GPU rows, or "no local
	// row was returned" is vacuously true and this test proves nothing.
	localRows := 0
	for _, name := range c.ListModels() {
		if c.UsesLocalGPU(name) {
			localRows++
		}
	}
	if localRows == 0 {
		t.Fatal("instrument check failed: registry has no local-GPU rows at all, so the " +
			"cloud-lane assertion below would pass vacuously")
	}

	for _, role := range c.ListRoles() {
		chain, err := c.ResolveRole(role, LaneCloud)
		if err != nil {
			continue // role may be local-only; covered by its own case
		}
		for _, e := range chain {
			if c.UsesLocalGPU(e.FriendlyName) {
				t.Errorf("role %q offers local-GPU model %q to the cloud lane", role, e.FriendlyName)
			}
		}
	}
}

func contains(hay, needle string) bool {
	return len(hay) >= len(needle) && (hay == needle || len(needle) == 0 ||
		func() bool {
			for i := 0; i+len(needle) <= len(hay); i++ {
				if hay[i:i+len(needle)] == needle {
					return true
				}
			}
			return false
		}())
}
