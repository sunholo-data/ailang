package modelreg

import "testing"

// M-MODEL-REGISTRY-SINGLE-SOURCE M3, revised M8 (Mark, 2026-08-27).
//
// Chain order is PREFERENCE: entry 1 is the subscription/OAuth lane, later
// entries the metered API-key lane to fall through to when OAuth is unavailable.
// Entry 1 therefore reproduces what the MISSION DRIVER runs today — the values
// Mark called "ok for first flush" — and the previous cloud-side values survive
// as the fallbacks.
//
// This replaces an assertion of byte-identity with the old config.cloud.yaml
// table. That assertion was correct while the chains were a pure transcription;
// it is obsolete now that the content changed by direction, and keeping it would
// have blocked a decision rather than protected one.
func TestResolveRole_LeadsWithSubscriptionLaneAndFallsBackToMetered(t *testing.T) {
	if err := InitModelsConfig(); err != nil {
		t.Fatalf("InitModelsConfig: %v", err)
	}
	c := GlobalModelsConfig

	// Entry 1: the model the mission driver runs today, by REGISTRY ROW — not by
	// string. Two of the four differ in string SHAPE from the driver's env value
	// (`claude:fable` vs `claude:claude-fable-5`; `claude:sonnet` vs bare
	// `sonnet`) while naming the same model. That gap is why M8 lands in shadow
	// mode rather than flipping the driver's values outright.
	wantPrimary := map[string]string{
		"designer":  "claude-fable-5",
		"planner":   "gpt5-6-sol",
		"executor":  "gpt5-6-sol",
		"evaluator": "claude-sonnet-4-6",
	}
	// Entry 2+: the metered lane, i.e. what the cloud fleet ran before this.
	wantFallback := map[string]string{
		"designer":  "openrouter/moonshotai/kimi-k3",
		"planner":   "openrouter/moonshotai/kimi-k3",
		"executor":  "openrouter/deepseek/deepseek-v4-flash-0731",
		"evaluator": "openrouter/minimax/minimax-m3",
	}

	for role, wantRow := range wantPrimary {
		chain, err := c.ResolveRole(role, LaneCloud)
		if err != nil {
			t.Errorf("ResolveRole(%q): %v", role, err)
			continue
		}
		if chain[0].FriendlyName != wantRow {
			t.Errorf("role %q entry 0 = %q, want %q (the subscription lane leads)",
				role, chain[0].FriendlyName, wantRow)
		}
		if len(chain) < 2 {
			t.Errorf("role %q has no fallback; every role needs a metered lane to "+
				"fall through to when OAuth is spent", role)
			continue
		}
		if chain[1].ModelName != wantFallback[role] {
			t.Errorf("role %q entry 1 = %q, want %q", role, chain[1].ModelName, wantFallback[role])
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
