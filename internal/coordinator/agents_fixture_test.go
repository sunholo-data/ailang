package coordinator

import (
	"encoding/json"
	"os"
	"testing"
)

// M-MODEL-REGISTRY-SINGLE-SOURCE M5 (decision D2(a) precondition, ratified 2026-08-27).
//
// D2(a) — fail loudly instead of falling back to a hardcoded default — was
// ratified with one sequencing condition: prove no live agent depends on a
// default BEFORE the defaults are removed. This fixture is that proof, and it
// gates M6.
//
// It also pins the M7 claim that deleting `model_routing` changes nothing: 33 of
// the 34 agents carry an explicit `model:`, and ResolveModel takes the pin
// first, so the routing table never fires for them. That makes the deletion
// inert BY CONSTRUCTION — this test is what turns "by construction" into a
// measurement.

type fixtureAgent struct {
	ID    string `json:"id"`
	Role  string `json:"role"`
	Model string `json:"model"`
}

type cloudFixture struct {
	Agents       []fixtureAgent      `json:"agents"`
	ModelRouting map[string][]string `json:"model_routing"`
}

func loadCloudFixture(t *testing.T) cloudFixture {
	t.Helper()
	raw, err := os.ReadFile("testdata/cloud_agents_20260827.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var f cloudFixture
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	if len(f.Agents) != 34 {
		t.Fatalf("fixture has %d agents, expected the 34 measured 2026-08-27; "+
			"if the fleet genuinely changed, re-snapshot and say so in the commit", len(f.Agents))
	}
	return f
}

// The load-bearing one: deleting model_routing must not move any agent.
func TestCloudAgents_ResolveIdenticallyWithoutModelRouting(t *testing.T) {
	f := loadCloudFixture(t)

	// Control: the fixture must actually CONTAIN a routing table, or "resolution
	// is unchanged when I remove it" is a statement about nothing.
	if len(f.ModelRouting) == 0 {
		t.Fatal("instrument check failed: fixture carries no model_routing, so removing " +
			"it below proves nothing")
	}

	var moved []string
	for _, a := range f.Agents {
		agent := &AgentConfig{ID: a.ID, Role: a.Role, Model: a.Model}

		before, errBefore := ResolveModel(agent, f.ModelRouting)
		after, errAfter := ResolveModel(agent, nil) // post-M7: no table at all

		if (errBefore == nil) != (errAfter == nil) || before != after {
			moved = append(moved, a.ID)
			t.Errorf("agent %q (role %q, pin %q) MOVES when model_routing is deleted: "+
				"%q (err %v) -> %q (err %v)", a.ID, a.Role, a.Model, before, errBefore, after, errAfter)
		}
	}
	if len(moved) > 0 {
		t.Logf("%d of %d agents move; M7 is NOT inert and needs a migration, not a deletion",
			len(moved), len(f.Agents))
	}
}

// The D2(a) blast radius: exactly which agents fall through to a provider
// default today. Each one hard-fails the moment M6 removes the defaults, so
// each needs a deliberate pin or role first.
func TestCloudAgents_NoAgentReliesOnAProviderDefault(t *testing.T) {
	f := loadCloudFixture(t)

	var unpinned []string
	for _, a := range f.Agents {
		agent := &AgentConfig{ID: a.ID, Role: a.Role, Model: a.Model}
		model, err := ResolveModel(agent, f.ModelRouting)
		if err == nil && model == "" {
			// ("", nil) is ResolveModel saying "no opinion" — the provider
			// default applies, which is precisely the hardcoded literal M6 deletes.
			unpinned = append(unpinned, a.ID)
		}
	}

	if len(unpinned) > 0 {
		t.Errorf("these agents resolve to no model and would hard-fail under D2(a): %v\n"+
			"Give each a deliberate `model:` pin or a `role:` BEFORE M6 removes the "+
			"defaults in internal/executor. This is the D2 sequencing condition.", unpinned)
	}
}
