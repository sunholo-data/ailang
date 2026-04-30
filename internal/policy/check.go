package policy

import (
	"sort"

	"github.com/sunholo-data/ailang/internal/types"
)

// ErrorKind enumerates the structured admission failure modes.
// Stable JSON: do not rename without bumping the schema version in messages.
type ErrorKind string

const (
	KindOK              ErrorKind = ""
	KindPolicyViolation ErrorKind = "policy_violation"
	KindParametricEntry ErrorKind = "parametric_entry_unsupported"
	KindMissingEntry    ErrorKind = "missing_entry"
	KindNotAFunction    ErrorKind = "entry_not_a_function"
)

// Decision is the result of a policy admission check.
//
// JSON shape is part of the runner contract — do not change without
// updating the message-schema version and the design doc at
// design_docs/planned/v0_16_0/m-agent-safe-runner.md.
type Decision struct {
	OK                bool      `json:"ok"`
	ErrorKind         ErrorKind `json:"error_kind,omitempty"`
	Message           string    `json:"message,omitempty"`
	Function          string    `json:"function,omitempty"`
	DeclaredEffects   []string  `json:"declared_effects"`
	AllowedCaps       []string  `json:"allowed_caps"`
	MissingFromPolicy []string  `json:"missing_from_policy,omitempty"`
}

// Check performs admission control: is the entry function's declared effect
// row permitted by the policy?
//
// Rules (v1 / spike):
//  1. If the row is nil (pure) or has no labels → admitted.
//  2. If the row has a non-nil Tail (parametric / open row) → REJECT with
//     KindParametricEntry. Rationale: an open row could instantiate to
//     anything; admitting it would defeat the purpose of the policy.
//     `main` is conventionally monomorphic, so this is rarely a real
//     limitation. Documented as a v1 restriction.
//  3. Otherwise: admit iff every label in the declared row is in
//     policy.AllowedCaps. Lists missing labels in MissingFromPolicy.
//
// Budgets and FS sandbox are enforced at runtime by the existing effect
// machinery; this function only does the static cap-subset check.
func Check(p *Policy, entry string, row *types.Row) Decision {
	allowed := sortedAllowed(p)

	// Pure or absent effect row: trivially admitted.
	if row == nil || len(row.Labels) == 0 {
		return Decision{
			OK:              true,
			Function:        entry,
			DeclaredEffects: []string{},
			AllowedCaps:     allowed,
		}
	}

	declared := sortedLabels(row)

	// Parametric entry: reject. We cannot soundly admit an open row.
	if row.Tail != nil {
		return Decision{
			OK:              false,
			ErrorKind:       KindParametricEntry,
			Message:         "entry function has parametric effect row; policy admission requires a monomorphic entry. Instantiate the entry's effects explicitly.",
			Function:        entry,
			DeclaredEffects: declared,
			AllowedCaps:     allowed,
		}
	}

	// Monomorphic subset check.
	allowedSet := p.AllowedSet()
	var missing []string
	for label := range row.Labels {
		if !allowedSet[label] {
			missing = append(missing, label)
		}
	}
	if len(missing) == 0 {
		return Decision{
			OK:              true,
			Function:        entry,
			DeclaredEffects: declared,
			AllowedCaps:     allowed,
		}
	}

	sort.Strings(missing)
	return Decision{
		OK:                false,
		ErrorKind:         KindPolicyViolation,
		Message:           "declared effects exceed policy",
		Function:          entry,
		DeclaredEffects:   declared,
		AllowedCaps:       allowed,
		MissingFromPolicy: missing,
	}
}

// CheckScheme adapts Check to a function's type Scheme. It pulls the effect
// row off a *TFunc2 underlying the scheme. Returns a structured Decision
// rather than an error so the caller can render it as JSON.
func CheckScheme(p *Policy, entry string, scheme *types.Scheme) Decision {
	allowed := sortedAllowed(p)

	if scheme == nil || scheme.Type == nil {
		return Decision{
			OK:          false,
			ErrorKind:   KindMissingEntry,
			Message:     "entry function has no type information",
			Function:    entry,
			AllowedCaps: allowed,
		}
	}

	fn, ok := scheme.Type.(*types.TFunc2)
	if !ok {
		return Decision{
			OK:          false,
			ErrorKind:   KindNotAFunction,
			Message:     "entry symbol is not a function",
			Function:    entry,
			AllowedCaps: allowed,
		}
	}

	return Check(p, entry, fn.EffectRow)
}

func sortedAllowed(p *Policy) []string {
	out := make([]string, len(p.AllowedCaps))
	copy(out, p.AllowedCaps)
	sort.Strings(out)
	return out
}

func sortedLabels(row *types.Row) []string {
	out := make([]string, 0, len(row.Labels))
	for k := range row.Labels {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
