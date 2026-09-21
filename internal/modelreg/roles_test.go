package modelreg

import "testing"

// M-MODEL-REGISTRY-SINGLE-SOURCE M3 left a transcription guard here
// (TestResolveRole_TranscribesLiveChainsByteIdentically) asserting that every role chain was
// byte-identical to config.cloud.yaml's deleted model_routing table, because "changing which
// models a role runs" was an explicit Non-Goal of that sprint.
//
// It was DELETED 2026-09-21, when the last two roles left it. All four have since been
// deliberately repointed — executor and evaluator on 2026-09-14, designer and planner on
// 2026-09-21 — each because the transcribed chain dispatched to a harness that cannot load the
// role's skill. A transcription guard whose map is empty asserts nothing; keeping it would have
// been a vacuous green. What replaces it is stronger: one PROPERTY every role must satisfy
// (below), plus a per-role assertion pinning each deliberate departure with its reason.

// TestResolveRole_EveryRoleHasASkillCapableRung is the unified guard for a defect class that
// has now been found four times, once per role, always as an incident and never as a property:
//
//   - evaluator, 2026-09-14 (7423434b4): dispatched to opencode while the shell ran pi. That is
//     how M-MISSION-ITERATION-RELIABILITY M4's canary failed 3/3 with no verdict.
//   - executor, 2026-09-14 (e9e8ce32e): [codex, opencode] — with codex over ration, "no
//     available candidate".
//   - designer + planner, 2026-09-21 (this commit): [opencode] and [codex, opencode]. Found by
//     asking the property instead of waiting for the third incident.
//
// The rule is ratified: EVERY RUNG MUST LOAD SKILLS (2026-09-08, attended,
// tools/launchd/mission-control.sh:1266). Each mission role IS a skill — design-doc-creator,
// sprint-planner, sprint-executor, sprint-evaluator — so a harness that cannot load one receives
// the NAME of the procedure and none of its content: it is asked to apply a rubric it never sees
// and run scripts it is never told exist. That failure is silent. The stage explores, burns its
// budget and returns without the artifact, which reads as a model failure rather than a routing
// one.
//
// skillLess is deliberately a DENY list of the two harnesses measured unable to load skills,
// not an allow list of the one that can: a new harness added to a chain should have to be proven
// skill-less to be excluded, rather than silently failing a check it was never named in. Today
// only pi carries the workspace-trust extension (internal/executor/pi/isolation.go); the comment
// block at mission-control.sh:1273 records that the old pi->pi->codex chain was "EVERY rung
// skill-less", which is the measurement behind codex's entry here.
func TestResolveRole_EveryRoleHasASkillCapableRung(t *testing.T) {
	if err := InitModelsConfig(); err != nil {
		t.Fatalf("InitModelsConfig: %v", err)
	}
	skillLess := map[string]string{
		"opencode": "no workspace-trust extension; the fix that lets pi load a skill was never ported",
		"codex":    "measured skill-less at mission-control.sh:1273 (the pi->pi->codex chain was EVERY rung skill-less)",
	}
	for _, role := range []string{"designer", "planner", "executor", "evaluator"} {
		for _, lane := range []Lane{LaneLocal, LaneCloud} {
			got, err := GlobalModelsConfig.ResolveRole(role, lane)
			if err != nil {
				t.Errorf("ResolveRole(%q, %s): %v", role, lane, err)
				continue
			}
			if len(got) == 0 {
				t.Errorf("role %q lane %s: empty chain", role, lane)
				continue
			}
			capable := make([]string, 0, len(got))
			rungs := make([]string, 0, len(got))
			for _, e := range got {
				rungs = append(rungs, e.FriendlyName+"/"+e.Executor)
				if e.Executor == "" {
					t.Errorf("role %q lane %s: rung %q has empty Executor", role, lane, e.FriendlyName)
					continue
				}
				if _, bad := skillLess[e.Executor]; !bad {
					capable = append(capable, e.Executor)
				}
			}
			if len(capable) == 0 {
				t.Errorf("role %q lane %s: NO skill-capable rung in %v — every rung is a harness "+
					"that cannot load this role's skill, so the stage runs without its method and "+
					"fails silently", role, lane, rungs)
			}
		}
	}
}

// TestResolveRole_DesignerAndPlannerAreDeliberatelyPiBacked pins the 2026-09-21 departure, the
// same way the evaluator and executor assertions below pin theirs. The property test above says
// only that SOME skill-capable rung exists; this says WHICH, and that it is the one the shell
// driver already ratified rather than a third routing opinion invented here.
//
// Each role keeps the MODEL its chain already named and moves it from opencode to pi, because
// the defect is the harness. That also keeps the cloud side inert: ResolveModelChain reads these
// rows on LaneCloud and TestCloudAgents_RegistryMatchesTheDeletedRoutingTable asserts the
// resolved model still matches config.cloud.yaml's deleted table, which it does because the
// model did not move. Corroboration for planner: the shell independently resolves the same model
// on the same harness (mission-control.sh:1215). The shell's designer HEAD
// (claude:claude-fable-5-1) is NOT transcribed — see the note above `roles:` in models.yml for
// why that one needs M8's parked mission-form work.
func TestResolveRole_DesignerAndPlannerAreDeliberatelyPiBacked(t *testing.T) {
	if err := InitModelsConfig(); err != nil {
		t.Fatalf("InitModelsConfig: %v", err)
	}
	want := map[string][]struct{ friendly, executor string }{
		"designer": {
			{"pi-or-kimi-k3", "pi"},
			{"opencode-or-kimi-k3", "opencode"},
		},
		"planner": {
			{"gpt5-6-sol", "codex"},
			{"pi-or-kimi-k3", "pi"},
			{"opencode-or-kimi-k3", "opencode"},
		},
	}
	for role, wantChain := range want {
		for _, lane := range []Lane{LaneLocal, LaneCloud} {
			got, err := GlobalModelsConfig.ResolveRole(role, lane)
			if err != nil {
				t.Errorf("ResolveRole(%q, %s): %v", role, lane, err)
				continue
			}
			if len(got) != len(wantChain) {
				t.Errorf("role %q lane %s: chain length %d, want %d (%v)", role, lane, len(got), len(wantChain), got)
				continue
			}
			for i, w := range wantChain {
				if got[i].FriendlyName != w.friendly || got[i].Executor != w.executor {
					t.Errorf("role %q lane %s entry %d: got %s/%s, want %s/%s", role, lane, i,
						got[i].FriendlyName, got[i].Executor, w.friendly, w.executor)
				}
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

// TestPiCloudRows_ExpressTheDriversFlatRateTier is M-ONE-ROLE-TABLE Phase 0.
//
// The mission driver leads the designer and planner fallback chains with an OLLAMA
// CLOUD route before the metered OpenRouter one (mission-control.sh:1106 and :1215),
// and until 2026-09-21 the registry could not express that rung at all: its 4
// pi×ollama rows were every one a LOCAL-GPU qwen3.x, and its ollama-cloud rows were
// all on the motoko harness.
//
// M-MODEL-REGISTRY-SINGLE-SOURCE M8's park note filed this as a capability gap. It was
// a missing row: IsOllamaCloudRoute and UsesLocalGPU already classify a `:cloud` /
// `-cloud` row as non-GPU. This test pins the three properties that claim rests on.
//
// These rows are INERT by design — no role chain names them. The last assertion is the
// one that keeps Phase 0 honest: adding vocabulary must not move any routing.
func TestPiCloudRows_ExpressTheDriversFlatRateTier(t *testing.T) {
	if err := InitModelsConfig(); err != nil {
		t.Fatalf("InitModelsConfig: %v", err)
	}
	c := GlobalModelsConfig

	// The driver strings these rows exist to express, verbatim from mission-control.sh.
	want := map[string]string{
		"pi-cloud-kimi-k3":           "ollama/kimi-k3:cloud",
		"pi-cloud-deepseek-v4-flash": "ollama/deepseek-v4-flash:0731-cloud",
	}
	for name, agentModel := range want {
		m, err := c.GetModel(name)
		if err != nil {
			t.Errorf("GetModel(%q): %v", name, err)
			continue
		}
		if m.AgentModelName == nil || *m.AgentModelName != agentModel {
			got := "<nil>"
			if m.AgentModelName != nil {
				got = *m.AgentModelName
			}
			t.Errorf("%s: agent_model_name = %q, want %q — this row exists ONLY to express "+
				"the driver's rung, so the string has to match it exactly", name, got, agentModel)
		}
		if m.AgentCLI == nil || *m.AgentCLI != "pi" {
			t.Errorf("%s: agent_cli must be pi — a motoko-harness row already exists for "+
				"these weights and is not what the driver runs", name)
		}
		// The property the park note called a capability gap: a cloud row is NOT
		// GPU-bound, so it survives the LaneCloud filter and never takes the rig lock.
		if c.UsesLocalGPU(name) {
			t.Errorf("%s: UsesLocalGPU = true — an ollama CLOUD route shares nothing with "+
				"the rig; serializing it behind the GPU lock buys nothing and costs wall-clock",
				name)
		}
		if !IsOllamaCloudRoute(*m.AgentModelName) {
			t.Errorf("%s: IsOllamaCloudRoute(%q) = false — the `:cloud`/`-cloud` grammar is what "+
				"keeps this off the rig lock", name, *m.AgentModelName)
		}
	}

	// Phase 0 is INERT: vocabulary only, no routing change. If a later phase puts these
	// into a chain that is a dated departure and this assertion is what makes it
	// deliberate rather than accidental.
	for _, role := range []string{"designer", "planner", "executor", "evaluator"} {
		for _, lane := range []Lane{LaneLocal, LaneCloud} {
			chain, err := GlobalModelsConfig.ResolveRole(role, lane)
			if err != nil {
				t.Errorf("ResolveRole(%q, %s): %v", role, lane, err)
				continue
			}
			for _, e := range chain {
				if _, isNew := want[e.FriendlyName]; isNew {
					t.Errorf("role %q lane %s now routes through %q — Phase 0 adds vocabulary and "+
						"changes NO routing. Putting a pi-cloud row into a chain is a separate, "+
						"dated departure (precedent: 7423434b4)", role, lane, e.FriendlyName)
				}
			}
		}
	}
}
