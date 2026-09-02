package coordinator

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
// (motoko) was pinned by M5. TestCloudAgents_RegistryMatchesTheDeletedRoutingTable
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
//     chain. The comment that used to sit here said retry chains were "a driver
//     concern" — true when the mission driver was the only dispatcher, and
//     false since the coordinator began dispatching unattended. The rest of the
//     chain is no longer discarded: see ResolveModelChain (M-COORDINATOR-
//     EXECUTION-TRUST M3);
//  3. a role the REGISTRY does not know is loud — "the registry has no entry for
//     this role" is a config gap, not a preference.
//
// No role and no model returns ("", nil): the coordinator has no opinion, and
// M6's executor-side check fails loudly at execution if nothing else supplies
// one. That split matters — the coordinator legitimately does not know the model
// for an agent whose task carries its own.
func ResolveModel(agent *AgentConfig) (string, error) {
	// One source of truth: the head of the chain ResolveModelChain returns.
	// Keeping a second resolution path here is how the two would drift, and a
	// drifted head means M3 silently re-routes every existing agent
	// (TestChainHeadMatchesResolveModel is the arm).
	chain, err := ResolveModelChain(agent)
	if err != nil {
		return "", err
	}
	if len(chain) == 0 {
		return "", nil
	}
	return chain[0], nil
}
