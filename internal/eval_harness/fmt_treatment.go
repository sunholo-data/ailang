package eval_harness

import (
	"fmt"
	"strings"
)

// fmtExtensionID is the extension id motoko reports when motoko_ext_fmt is
// loaded (see the step-0 runtime_config_resolved broadcast).
const fmtExtensionID = "fmt"

// ResolveFmtArm reports which arm of the fmt experiment a run belongs to.
//
// # WHY THIS IS NOT JUST THE FLAG
//
// There are TWO fmt mechanisms, and they are easy to conflate:
//
//   - the Claude Code PostToolUse hook, selected by `-fmt-hook on|off`
//   - motoko_ext_fmt, selected by loading the extension via a motoko profile
//
// FmtHookState was resolved from the flag alone. A motoko-profile arm never
// sets that flag, so the ollama_fmt TREATMENT arm banked "off" — the same label
// as its own control. The comparison could not tell the arms apart, and M6's
// verification gate had nothing to assert against.
//
// The arm is therefore derived from what was actually LOADED (the subject's own
// step-0 broadcast), falling back to the flag for the Claude path.
func ResolveFmtArm(flagMode FmtHookMode, resolvedExtensions string) string {
	if flagMode == FmtHookModeOn {
		return "on"
	}
	for _, id := range strings.Split(resolvedExtensions, ",") {
		// Exact match, not substring: "fmtx" and "not_fmt_really" are different
		// extensions that merely contain the letters.
		if strings.TrimSpace(id) == fmtExtensionID {
			return "on"
		}
	}
	return "off"
}

// AssertFmtTreatmentIntegrity checks that the fmt treatment actually applied,
// returning an invalid marker when it did not. Returns nil when the arm is
// consistent with its observed events.
//
// This is M6's verification gate ("a single fmt_on run banks fmt_hook_events > 0
// ... a fmt_off run banks 0") and M5's void clause, made executable. Both
// existed only as prose, which is why the fmt A/B could have reported a
// confident null while the treatment never fired.
//
// The control arm is checked too: events on an OFF arm mean contamination, and
// a contaminated control invalidates the comparison just as surely as an inert
// treatment.
func AssertFmtTreatmentIntegrity(arm string, events []FmtHookEvent) *Validity {
	switch arm {
	case "on":
		if len(events) == 0 {
			v := MarkInvalid(ReasonTreatmentUnproven)
			v.Detail = "fmt ON arm banked zero fmt_hook_events: the treatment cannot be shown to have applied, so a null result here would be meaningless (M-EVAL-FMT-WEAKMODEL-AB M6 verification gate)"
			return v
		}
	case "off":
		if len(events) > 0 {
			v := MarkInvalid(ReasonTreatmentUnproven)
			v.Detail = fmt.Sprintf("fmt OFF arm banked %d fmt_hook_event(s): the control was contaminated by the treatment, so the comparison is invalid", len(events))
			return v
		}
	}
	return nil
}
