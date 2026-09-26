package modelreg

import (
	"fmt"
	"sort"
	"strings"
)

// M-MODEL-REGISTRY-SINGLE-SOURCE M3 (decisions D1(a)/D4(a), ratified 2026-08-27).
//
// models.yml has always been the registry of model FACTS. This adds model
// POLICY: which model plays which role. It supersedes the `model_routing` table
// in config.cloud.yaml (M-PIPELINE-RECONCILIATION M5), which answered the same
// question in a second place.
//
// Chains name FRIENDLY NAMES, not wire strings. That is deliberate and it is
// what makes the table unambiguous. The live model_routing entries are raw
// per-harness strings, and mapping one back to the registry is many-to-one:
// "openrouter/deepseek/deepseek-v4-flash-0731" is both motoko-or-deepseek-v4-flash
// and opencode-or-deepseek-v4-flash. A friendly name picks the row, and the row
// carries both the harness (agent_cli) and the exact wire string.

// Lane says where a resolution will run. It exists because ollama rows are the
// single shared GPU rig: offering one to a Cloud Run job resolves to a model
// that host cannot reach.
type Lane string

const (
	// LaneLocal is the GPU rig — every row is eligible, ollama included.
	LaneLocal Lane = "local"
	// LaneCloud is Cloud Run and friends — local-GPU rows are filtered out.
	LaneCloud Lane = "cloud"
)

// RoleEntry is one link in a role's fallback chain, resolved against the registry.
type RoleEntry struct {
	// FriendlyName is the models.yml key — the auditable identity.
	FriendlyName string
	// Executor is the harness that runs it (agent_cli): codex, opencode, pi, motoko.
	Executor string
	// ModelName is the exact string handed to that harness. It is
	// agent_model_name when the row has one and api_name otherwise — the rule
	// GetExecutorForModel already implements, not a second one.
	ModelName string
}

// ListRoles returns the declared role names, sorted.
func (c *ModelsConfig) ListRoles() []string {
	roles := make([]string, 0, len(c.Roles))
	for r := range c.Roles {
		roles = append(roles, r)
	}
	sort.Strings(roles)
	return roles
}

// ResolveRole returns a role's ordered fallback chain for a lane.
//
// It is loud in three places, because each silent answer would be worse than an
// error: an unknown role is a config gap; an unresolvable friendly name means
// the chain names a model the registry does not have; and a chain emptied by
// lane filtering means the role is declared but unreachable from where the
// caller is standing.
func (c *ModelsConfig) ResolveRole(role string, lane Lane) ([]RoleEntry, error) {
	chain, ok := c.Roles[role]
	if !ok || len(chain) == 0 {
		return nil, fmt.Errorf("registry has no chain for role %q (roles declared: %s)",
			role, strings.Join(c.ListRoles(), ", "))
	}

	var out []RoleEntry
	var skippedLocal []string
	for _, friendly := range chain {
		executor, modelName, err := c.GetExecutorForModel(friendly)
		if err != nil {
			return nil, fmt.Errorf("role %q names %q, which the registry cannot resolve: %w",
				role, friendly, err)
		}
		if lane == LaneCloud && c.UsesLocalGPU(friendly) {
			skippedLocal = append(skippedLocal, friendly)
			continue
		}
		out = append(out, RoleEntry{
			FriendlyName: friendly,
			Executor:     executor,
			ModelName:    modelName,
		})
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("role %q has no entry reachable from lane %q; "+
			"every model in its chain runs on the local GPU rig (%s)",
			role, lane, strings.Join(skippedLocal, ", "))
	}
	return out, nil
}
