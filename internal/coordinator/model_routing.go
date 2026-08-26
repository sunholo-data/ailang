package coordinator

import (
	"fmt"
	"sort"
	"strings"
)

// M-PIPELINE-RECONCILIATION M5 (decision D3, ratified 2026-08-26): one model
// routing table, read by both lanes.
//
// Before this, Lane B pinned models statically per agent in the cloud config
// (`model: opus` on the designer and executor, set once and never revisited),
// while Lane A's mission loop evolved cross-provider rotation chains in shell.
// The two answered "which model runs this role?" differently, and the static
// pins burned opus by default with none of the fallback logic.
//
// The table maps a ROLE to an ordered model chain. Lane B resolves an agent's
// model through it (first entry — the coordinator's executor takes one model;
// retry chains are a driver concern). Lane A's driver reads the same table via
// `ailang coordinator routing <role>`, which prints the full chain.

// ModelRouting maps role name → ordered model chain (primary first).
type ModelRouting map[string][]string

// ResolveModel picks the model for an agent:
//
//  1. an explicit per-agent `model:` wins (it is a deliberate pin);
//  2. else the agent's `role:` looks up the routing table, first entry;
//  3. a role that is MISSING from the table is loud — it returns an error
//     rather than silently falling back, because "the routing table does not
//     know this role" is a config gap, not a preference.
//
// No role and no model returns ("", nil): the provider default applies, which
// is the pre-M5 behavior for every agent that never opted into routing.
func ResolveModel(agent *AgentConfig, routing ModelRouting) (string, error) {
	if agent == nil {
		return "", nil
	}
	if agent.Model != "" {
		return agent.Model, nil
	}
	if agent.Role == "" {
		return "", nil
	}
	chain, ok := routing[agent.Role]
	if !ok || len(chain) == 0 {
		return "", fmt.Errorf("agent %q has role %q but the model_routing table has no entry for it (roles present: %s)",
			agent.ID, agent.Role, strings.Join(routingRoles(routing), ", "))
	}
	return chain[0], nil
}

func routingRoles(r ModelRouting) []string {
	roles := make([]string, 0, len(r))
	for k := range r {
		roles = append(roles, k)
	}
	sort.Strings(roles)
	return roles
}

// coordConfigRouting returns the daemon's loaded routing table (nil-safe).
func (d *Daemon) coordConfigRouting() ModelRouting {
	if d.coordConfig == nil {
		return nil
	}
	return d.coordConfig.ModelRouting
}
