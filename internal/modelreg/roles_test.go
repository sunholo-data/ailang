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
	//
	// Neither `evaluator` nor `executor` is in this map any more. It was deliberately repointed on
	// 2026-09-14 (attended, Mark) and both are asserted separately below, because a
	// transcription guard cannot also be the record of an intentional departure —
	// leaving it here would have meant either a silent edit to the "verbatim"
	// baseline or deleting the guard for the other three roles. The other three
	// remaining two are pure transcriptions and are still held byte-identical.
	want := map[string][]string{
		"designer": {"openrouter/moonshotai/kimi-k3"},
		"planner":  {"gpt-5.6-sol", "openrouter/moonshotai/kimi-k3"},
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

// The evaluator chain is a DELIBERATE departure from the transcription above.
//
// Repointed 2026-09-14 (attended, Mark) from opencode to pi. The binary mission path
// (internal/mission/iteration, LaneLocal) is this row's only local consumer, and it was
// dispatching evaluator stages to opencode while tools/launchd/mission-control.sh resolved
// `pi:openrouter/minimax/minimax-m3,claude:claude-sonnet-4-6,opus`. The two tables had
// silently diverged, so `ailang mission iterate` ran a harness that never received the
// workspace-trust fix that lets pi load the sprint-evaluator skill — the 100-point rubric
// that is the evaluator's terminator. That is how M-MISSION-ITERATION-RELIABILITY M4's
// canary failed 3/3 with no verdict.
//
// This assertion exists so the departure stays intentional: an accidental reorder or a
// revert to opencode-first reds here with the reason attached, rather than silently
// re-breaking the lane. It is NOT a transcription of config.cloud.yaml.
func TestResolveRole_EvaluatorIsDeliberatelyPiFirst(t *testing.T) {
	if err := InitModelsConfig(); err != nil {
		t.Fatalf("InitModelsConfig: %v", err)
	}
	want := []struct{ friendly, model, executor string }{
		{"pi-or-minimax-m3", "openrouter/minimax/minimax-m3", "pi"},
		{"pi-claude-sonnet-4-6", "anthropic/claude-sonnet-4-6", "pi"},
		// Kept as the last rung, not deleted: it is the route the 09-08 canary ran.
		{"opencode-or-minimax-m3", "openrouter/minimax/minimax-m3", "opencode"},
	}
	// Both lanes, because the coordinator reads this row on LaneCloud. It short-circuits
	// first for any agent with an explicit model (retry_chain.go:118) and the one
	// evaluator-role cloud agent pins its own, so nothing on the plane reaches this today —
	// but the chain must still be well-formed if something ever does.
	for _, lane := range []Lane{LaneLocal, LaneCloud} {
		got, err := GlobalModelsConfig.ResolveRole("evaluator", lane)
		if err != nil {
			t.Fatalf("ResolveRole(evaluator, %s): %v", lane, err)
		}
		if len(got) != len(want) {
			t.Fatalf("lane %s: chain length %d, want %d (%v)", lane, len(got), len(want), got)
		}
		for i, w := range want {
			if got[i].FriendlyName != w.friendly || got[i].ModelName != w.model || got[i].Executor != w.executor {
				t.Errorf("lane %s entry %d: got %s/%s/%s, want %s/%s/%s", lane, i,
					got[i].FriendlyName, got[i].ModelName, got[i].Executor, w.friendly, w.model, w.executor)
			}
		}
		// The point of the change: the FIRST rung must be pi, or the binary dispatches
		// its evaluator to a harness that cannot load the sprint-evaluator skill.
		if got[0].Executor != "pi" {
			t.Errorf("lane %s: first rung executor = %q, want pi", lane, got[0].Executor)
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

// The executor chain is a DELIBERATE departure too, for the same reason as evaluator.
//
// Measured 2026-09-14: with codex over ration, `mission iterate` reported "no available
// candidate" — gpt5-6-sol blocked on quota and opencode-or-deepseek-v4-flash refused with
// "executor budget contract is not admitted by role-run". The executor role therefore had
// NO usable route on the binary path whenever codex is rationed, which blocked
// M-MISSION-ITERATION-RELIABILITY M4's last acceptance criterion.
//
// The bare-id pi row is deliberate: `:floor` was dropped from the mission executor lane on
// 2026-08-18 because its two cheapest hosts carry negative health status and it returned
// rc=0 with zero bytes changed twice.
func TestResolveRole_ExecutorHasAnAdmissibleNonCodexRung(t *testing.T) {
	if err := InitModelsConfig(); err != nil {
		t.Fatalf("InitModelsConfig: %v", err)
	}
	want := []struct{ friendly, executor string }{
		{"gpt5-6-sol", "codex"},
		{"pi-or-deepseek-v4-flash-bare", "pi"},
		{"opencode-or-deepseek-v4-flash", "opencode"},
	}
	for _, lane := range []Lane{LaneLocal, LaneCloud} {
		got, err := GlobalModelsConfig.ResolveRole("executor", lane)
		if err != nil {
			t.Fatalf("ResolveRole(executor, %s): %v", lane, err)
		}
		if len(got) != len(want) {
			t.Fatalf("lane %s: chain length %d, want %d (%v)", lane, len(got), len(want), got)
		}
		for i, w := range want {
			if got[i].FriendlyName != w.friendly || got[i].Executor != w.executor {
				t.Errorf("lane %s entry %d: got %s/%s, want %s/%s", lane, i,
					got[i].FriendlyName, got[i].Executor, w.friendly, w.executor)
			}
		}
		// The point: at least one rung must be neither codex (rationable) nor opencode
		// (inadmissible on role-run), or the role has no route when codex is blocked.
		usable := false
		for _, e := range got {
			if e.Executor != "codex" && e.Executor != "opencode" {
				usable = true
			}
		}
		if !usable {
			t.Errorf("lane %s: no rung outside codex/opencode — executor has no route when codex is rationed", lane)
		}
	}
}
