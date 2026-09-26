package coordinator

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/modelreg"
)

// M-MODEL-REGISTRY-SINGLE-SOURCE M7.
//
// The coordinator resolves an agent's model through the REGISTRY, superseding
// the `model_routing` table in config.cloud.yaml (M-PIPELINE-RECONCILIATION M5).
// This is the coordinator's first dependency on the registry — safe only because
// M1 made modelreg a leaf (V7 flagged the cycle risk; D4(a) removed it).

func TestResolveModel_ExplicitPinWins(t *testing.T) {
	if err := modelreg.InitModelsConfig(); err != nil {
		t.Fatalf("init registry: %v", err)
	}
	// A deliberate pin must beat the registry: it is an operator saying "this one".
	got, err := ResolveModel(&AgentConfig{ID: "a", Model: "some/explicit-model", Role: "executor"})
	if err != nil {
		t.Fatalf("ResolveModel: %v", err)
	}
	if got != "some/explicit-model" {
		t.Errorf("got %q, want the explicit pin", got)
	}
}

func TestResolveModel_RoleResolvesThroughRegistry(t *testing.T) {
	if err := modelreg.InitModelsConfig(); err != nil {
		t.Fatalf("init registry: %v", err)
	}
	// Control: the registry must actually declare this role, or the assertion
	// below is about nothing.
	chain, err := modelreg.GlobalModelsConfig.ResolveRole("executor", modelreg.LaneCloud)
	if err != nil || len(chain) == 0 {
		t.Fatalf("instrument check failed: registry has no executor chain (%v)", err)
	}

	got, err := ResolveModel(&AgentConfig{ID: "b", Role: "executor"})
	if err != nil {
		t.Fatalf("ResolveModel: %v", err)
	}
	if got != chain[0].ModelName {
		t.Errorf("got %q, want the registry's first entry %q", got, chain[0].ModelName)
	}
}

func TestResolveModel_NoRoleAndNoPinIsSilent(t *testing.T) {
	// ("", nil) means "no opinion" — the executor's own fail-loud (M6) handles
	// it at execution. The coordinator must not invent a model here.
	got, err := ResolveModel(&AgentConfig{ID: "c"})
	if err != nil {
		t.Fatalf("an agent with neither pin nor role is not an error here: %v", err)
	}
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestResolveModel_UnknownRoleIsLoudAndNamesKnownRoles(t *testing.T) {
	if err := modelreg.InitModelsConfig(); err != nil {
		t.Fatalf("init registry: %v", err)
	}
	_, err := ResolveModel(&AgentConfig{ID: "d", Role: "no-such-role"})
	if err == nil {
		t.Fatal("a role the registry does not know is a config gap and must be loud")
	}
	if !strings.Contains(err.Error(), "d") {
		t.Errorf("error must name the agent so an operator knows what broke; got: %v", err)
	}
	for _, known := range []string{"designer", "executor"} {
		if !strings.Contains(err.Error(), known) {
			t.Errorf("error should name known role %q; got: %v", known, err)
		}
	}
}
