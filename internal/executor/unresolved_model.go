package executor

import (
	"fmt"
	"strings"
)

// M-MODEL-REGISTRY-SINGLE-SOURCE M6 (decision D2(a), ratified by Mark 2026-08-27).
//
// What replaced the ten hardcoded model defaults.
//
// Each executor used to substitute a literal when nothing set a model — four of
// them claude-haiku-4-5. The fleet migrated off Anthropic on 2026-08-27 because
// Claude-CLI OAuth for headless agents is being retired, so every one of those
// literals was a silent path back onto a provider we are leaving, invisible
// until an invoice or an outage. CLAUDE.md Critical Principle 2: a fallback that
// changes billing or availability is not a convenience, it is a defect.
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
		"put an unpinned agent back on a provider the fleet has migrated off")
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
