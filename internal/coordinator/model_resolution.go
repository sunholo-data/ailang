package coordinator

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/modelreg"
)

// M-MODEL-REGISTRY-SINGLE-SOURCE M7 (decisions D1(a)/D4(a), ratified 2026-08-27).
//
// This file was model_routing.go, which held a `ModelRouting` table loaded from
// config.cloud.yaml (M-PIPELINE-RECONCILIATION M5). That table answered "which
// model runs this role?" in a SECOND place — a second place that needed its own
// deploy, and that the mission driver never actually read despite a comment
// saying it did.
//
// The registry answers it now. Deleting the table was proven inert before it was
// deleted: 33 of the 34 cloud agents carry an explicit `model:` pin and
// ResolveModel takes the pin first, so the table never fired for them; the 34th
// (motoko) was pinned by M5. TestCloudAgents_ResolveIdenticallyWithoutModelRouting
// measures that rather than assuming it.
//
// This is the coordinator's FIRST dependency on the registry package. V7 flagged
// the cycle risk; D4(a) removed it by making modelreg a leaf, so the import is
// one-way and safe.

// ResolveModel picks the model for an agent:
//
//  1. an explicit per-agent `model:` wins — it is a deliberate pin, and an
//     operator naming a model outranks any table;
//  2. else the agent's `role:` resolves through the registry, first entry of the
//     chain (the coordinator's executor takes one model; retry chains are a
//     driver concern);
//  3. a role the REGISTRY does not know is loud — "the registry has no entry for
//     this role" is a config gap, not a preference.
//
// No role and no model returns ("", nil): the coordinator has no opinion, and
// M6's executor-side check fails loudly at execution if nothing else supplies
// one. That split matters — the coordinator legitimately does not know the model
// for an agent whose task carries its own.
func ResolveModel(agent *AgentConfig) (string, error) {
	if agent == nil {
		return "", nil
	}
	if agent.Model != "" {
		return agent.Model, nil
	}
	if agent.Role == "" {
		return "", nil
	}

	if modelreg.GlobalModelsConfig == nil {
		if err := modelreg.InitModelsConfig(); err != nil {
			return "", fmt.Errorf("agent %q has role %q but the model registry could not be loaded: %w",
				agent.ID, agent.Role, err)
		}
	}

	// Cloud lane: the coordinator dispatches to Cloud Run, which cannot reach the
	// local GPU rig. Offering an ollama row here would resolve to a model that
	// host cannot run.
	chain, err := modelreg.GlobalModelsConfig.ResolveRole(agent.Role, modelreg.LaneCloud)
	if err != nil {
		return "", fmt.Errorf("agent %q: %w (roles the registry knows: %s)",
			agent.ID, err, strings.Join(modelreg.GlobalModelsConfig.ListRoles(), ", "))
	}
	return chain[0].ModelName, nil
}
