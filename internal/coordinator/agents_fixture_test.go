package coordinator

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/sunholo-data/ailang/internal/modelreg"
)

// M-MODEL-REGISTRY-SINGLE-SOURCE M5 + M7.
//
// The fixture snapshots ailang-multivac/config/config.cloud.yaml — the 34 cloud
// agents and the `model_routing` table as it stood on 2026-08-27, BEFORE M7
// deleted that table.
//
// It carries two guarantees:
//
//  1. M7's deletion is INERT. The registry must resolve every agent exactly as
//     the config-side table did. 33 of 34 carry an explicit pin (which always
//     won anyway) and the 4 role-bearing agents must now resolve to the same
//     model the table named. This is what turns "inert by construction" into a
//     measurement.
//  2. D2(a)'s precondition holds: no agent falls through to a provider default,
//     which is what M6 removed. That gate is why M5 had to land before M6.

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

// The load-bearing one: the registry must give the same answer the deleted table did.
func TestCloudAgents_RegistryMatchesTheDeletedRoutingTable(t *testing.T) {
	if err := modelreg.InitModelsConfig(); err != nil {
		t.Fatalf("init registry: %v", err)
	}
	f := loadCloudFixture(t)

	// Control: the fixture must actually carry the old table, or "the registry
	// agrees with it" is a statement about nothing.
	if len(f.ModelRouting) == 0 {
		t.Fatal("instrument check failed: fixture carries no model_routing to compare against")
	}

	checkedRoles := 0
	for _, a := range f.Agents {
		got, err := ResolveModel(&AgentConfig{ID: a.ID, Role: a.Role, Model: a.Model})
		if err != nil {
			t.Errorf("agent %q (role %q, pin %q): %v", a.ID, a.Role, a.Model, err)
			continue
		}

		// An explicit pin always won, before and after. It must still.
		if a.Model != "" {
			if got != a.Model {
				t.Errorf("agent %q pin %q now resolves to %q — pins must be untouchable",
					a.ID, a.Model, got)
			}
			continue
		}
		if a.Role == "" {
			if got != "" {
				t.Errorf("agent %q has neither pin nor role but resolved to %q", a.ID, got)
			}
			continue
		}
		// Role-only: the registry must name what the old table named.
		want := f.ModelRouting[a.Role]
		if len(want) == 0 {
			t.Errorf("agent %q has role %q, absent from the recorded table", a.ID, a.Role)
			continue
		}
		checkedRoles++
		if got != want[0] {
			t.Errorf("agent %q role %q: registry says %q, the deleted table said %q — "+
				"M7 was supposed to be inert", a.ID, a.Role, got, want[0])
		}
	}
	// Every one of the 34 carries a pin, so the loop above never exercises the
	// ROLE path — which would make "the registry matches the old table" a claim
	// about code no real agent runs. Test it directly: for each role-bearing
	// agent, drop the pin and require the registry to name what the table named.
	// This is the path that fires the moment somebody removes a pin.
	if checkedRoles == 0 {
		for _, a := range f.Agents {
			if a.Role == "" {
				continue
			}
			want := f.ModelRouting[a.Role]
			if len(want) == 0 {
				t.Errorf("agent %q has role %q, absent from the recorded table", a.ID, a.Role)
				continue
			}
			got, err := ResolveModel(&AgentConfig{ID: a.ID, Role: a.Role}) // pin deliberately dropped
			if err != nil {
				t.Errorf("agent %q role %q unpinned: %v", a.ID, a.Role, err)
				continue
			}
			checkedRoles++
			if got != want[0] {
				t.Errorf("agent %q role %q WITHOUT its pin: registry says %q, the deleted "+
					"table said %q — M7 is not inert on the role path", a.ID, a.Role, got, want[0])
			}
		}
		if checkedRoles == 0 {
			t.Error("no agent carries a role, so the role path went untested entirely")
		}
	}
	t.Logf("34 agents checked; %d role resolutions verified against the deleted table", checkedRoles)
}

// D2(a)'s precondition, and the gate that had to pass before M6 removed the
// hardcoded defaults.
func TestCloudAgents_NoAgentReliesOnAProviderDefault(t *testing.T) {
	if err := modelreg.InitModelsConfig(); err != nil {
		t.Fatalf("init registry: %v", err)
	}
	f := loadCloudFixture(t)

	var unpinned []string
	for _, a := range f.Agents {
		model, err := ResolveModel(&AgentConfig{ID: a.ID, Role: a.Role, Model: a.Model})
		if err == nil && model == "" {
			// ("", nil) is the coordinator saying "no opinion" — before M6 the
			// provider default applied, which is precisely the literal M6 deleted.
			unpinned = append(unpinned, a.ID)
		}
	}

	if len(unpinned) > 0 {
		t.Errorf("these agents resolve to no model and hard-fail at execution under D2(a): %v\n"+
			"Give each a deliberate `model:` pin or a `role:` the registry knows.", unpinned)
	}
}
