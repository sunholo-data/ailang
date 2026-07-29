package eval_analysis

import "fmt"

// DefaultHeadroomCeiling is the control-arm pass rate above which an A/B is
// unlikely to resolve anything. Advisory, not a hard limit.
const DefaultHeadroomCeiling = 0.90

// Headroom reports whether a comparison's control arm has room to move.
//
// # WHY THIS IS A RULE AND NOT A JUDGEMENT CALL
//
// An experiment can only detect an effect where the control arm can actually
// get worse or better. The fmt A/B was run against haiku, where both arms sat
// at ~96% — 30/30 vs 29/30, and 42/45 vs 43/45. Those numbers cannot
// distinguish "fmt does nothing" from "fmt helps a lot but there was nothing
// left to fix", because a benchmark set that is already passed cannot show
// improvement. The run consumed rig time and produced no information, and the
// mistake was invisible until someone looked at both arms side by side.
//
// The subject has to be chosen where the failure mode actually occurs. For a
// syntax-drift remedy that means a weak local model, not a frontier model at
// ceiling.
type Headroom struct {
	// Warn is true when the control arm is at or above the ceiling.
	Warn bool `json:"warn"`
	// Rate is the observed control-arm pass rate (0..1).
	Rate float64 `json:"rate"`
	// Ceiling is the threshold used.
	Ceiling float64 `json:"ceiling"`
	// Message explains the problem. Empty when Warn is false.
	Message string `json:"message,omitempty"`
}

// CheckHeadroom evaluates a control arm's pass rate against a ceiling.
//
// Returns Warn=false for an empty sample: no observations is not evidence of a
// ceiling, and a rule that fires on every brand-new experiment gets ignored.
//
// This NEVER blocks a run. A saturated arm is a reason to doubt a null result,
// not a reason to refuse to measure — a regression guard on a saturated arm is
// a legitimate thing to want.
func CheckHeadroom(controlPass, controlTotal int, ceiling float64) Headroom {
	if controlTotal <= 0 {
		return Headroom{Ceiling: ceiling}
	}

	rate := float64(controlPass) / float64(controlTotal)
	h := Headroom{Rate: rate, Ceiling: ceiling}
	if rate < ceiling {
		return h
	}

	h.Warn = true
	h.Message = fmt.Sprintf(
		"control arm is at %.1f%% (%d/%d), at or above the %.0f%% ceiling: there is too little headroom for this comparison to resolve a small effect. "+
			"A null result here means 'the benchmark set was already passed', not 'the treatment does nothing'. "+
			"Pick a subject where the failure mode actually occurs — this is the mistake that made the haiku fmt A/B (both arms ~96%%) uninformative.",
		100*rate, controlPass, controlTotal, 100*ceiling)
	return h
}
