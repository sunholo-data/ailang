package executor

import (
	"fmt"
	"strings"
)

// M-MODEL-REGISTRY-SINGLE-SOURCE M6 (decision D2(a), ratified by Mark 2026-08-27).
//
// What replaced the ten hardcoded model defaults.
//
// WHY THE DEFAULTS WERE A DEFECT — stated accurately, because the first version
// of this comment was not.
//
// Four of the ten literals were claude-haiku-4-5. An agent whose pin was dropped
// silently ran a model nobody chose, on an account nobody was watching. That is
// the defect on its own terms: a fallback that changes which model runs, and
// therefore what it costs, is a data-integrity fallback (CLAUDE.md Critical
// Principle 2), whatever the provider.
//
// WHAT THIS COMMENT USED TO CLAIM, AND WHY IT WAS WRONG. It said the fleet moved
// off Anthropic "because Claude-CLI OAuth for headless agents is being retired".
// Verified against Anthropic's own help centre 2026-08-27: that change was
// ANNOUNCED for 2026-06-15 and PAUSED on the day. `claude -p` still draws on
// subscription limits; the separate API-rate credit was never issued. Anthropic
// say they will give notice before anything takes effect.
//
// And Cloud Run jobs DO run Claude on subscription OAuth: the `agent_executor`
// job template takes CLAUDE_CODE_OAUTH_TOKEN from Secret Manager, deliberately
// kept apart from the `agent_executor_apikey` template so the two auth modes
// cannot mix. So neither "OAuth is going away" nor "containers cannot do OAuth"
// is true.
//
// The 2026-08-27 fleet migration stands on its own reasons — cost visibility and
// single-lane metering — and as a hedge against a PAUSED change. It is not a
// response to a live one.
//
// The error names the agent AND the roles the registry knows, because the two
// questions an operator has next are "which agent broke?" and "what may I pin it
// to?". An error that answers neither just relocates the search.

// UnresolvedModelError reports that an executor was constructed with no model
// and has no default to fall back on.
type UnresolvedModelError struct {
	Executor    string   // harness that needed a model: opencode, pi, motoko, claude, codex
	ConfigField string   // the Config field that would have supplied it
	KnownRoles  []string // roles the registry can resolve, if it has been loaded
}

func (e *UnresolvedModelError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "executor %q was given no model and has no default: "+
		"set the agent's `model:` pin, or give it a `role:` the registry can resolve",
		e.Executor)
	if e.ConfigField != "" {
		fmt.Fprintf(&b, " (config field %s)", e.ConfigField)
	}
	if len(e.KnownRoles) > 0 {
		fmt.Fprintf(&b, ". Roles the registry knows: %s", strings.Join(e.KnownRoles, ", "))
	} else {
		b.WriteString(". The registry declares no roles in this process, " +
			"so an explicit model pin is the only option here")
	}
	b.WriteString(". Hardcoded provider defaults were removed deliberately " +
		"(M-MODEL-REGISTRY-SINGLE-SOURCE D2(a)): a silent fallback here would " +
		"put an unpinned agent on a model nobody chose")
	return b.String()
}

// RoleLister lets the error name known roles without executor importing the
// registry. The registry is a leaf that executor MAY import (D4(a)), but keeping
// this an injected function keeps the error message a formatting concern rather
// than a coupling.
var RoleLister func() []string

// ErrUnresolvedModel builds the typed error for one executor.
func ErrUnresolvedModel(executorName, configField string) error {
	var roles []string
	if RoleLister != nil {
		roles = RoleLister()
	}
	return &UnresolvedModelError{Executor: executorName, ConfigField: configField, KnownRoles: roles}
}
