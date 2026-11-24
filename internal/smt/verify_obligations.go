package smt

import (
	"fmt"
	"regexp"

	"github.com/sunholo-data/ailang/internal/verification"
)

// Caveat qualifies a result without changing its status: something the proof
// does not cover, with an input that shows it when one exists.
type Caveat struct {
	Kind   string         `json:"kind"`
	Detail string         `json:"detail"`
	Model  []ModelBinding `json:"model,omitempty"`
}

// Caveat kinds.
const (
	CaveatIntOverflow          = "int_overflow"
	CaveatIntOverflowUndecided = "int_overflow_undecided"
	CaveatFloatInputs          = "float_inputs_assumed_non_nan"
)

const intOverflowDetail = "an int + - * may leave the int64 range for some inputs (runtime int wraps at 64 bits); bound the inputs in requires, or run with --int-model=int64 to make this a violation"

// checkCallObligations runs the obligation queries of a verified result: callee
// requires at each call (M3, D1 a) and the numeric operations (M4, D2 a′ and
// D4 b′). The postcondition was proved assuming every obligation; the result
// stays verified only if each is shown to hold — except int64 overflow, which
// by default is a caveat on the verified result.
func checkCallObligations(vr VerifyResult, funcName string, enc *EncodeResult, cfg SolverConfig, boundedDepth int, opts VerifyOptions) VerifyResult {
	for _, ob := range enc.Obligations {
		if !isNumericObligation(ob.Callee) {
			vr.RequiresChecked = append(vr.RequiresChecked, ob.Callee)
		}
	}
	intChecked := opts.IntModel == verification.IntModelInt64
	for _, ob := range enc.Obligations {
		sr, err := Solve(ob.SMTLib, cfg)
		if err != nil {
			return VerifyResult{Function: funcName, Status: "error", Reason: fmt.Sprintf("solver error checking %s: %v", obligationLabel(ob.Callee), err), ReasonCode: verification.VerifierError}
		}
		r := solverVerifyResult(funcName, sr, boundedDepth, opts.Verbose)
		if r.Status == "verified" {
			continue
		}
		if ob.Callee == obligationInt64 && !intChecked {
			// D2 a′: reported on the verified result, never a verdict change.
			c := Caveat{Kind: CaveatIntOverflow, Detail: intOverflowDetail}
			if r.Status == "counterexample" || r.Status == "inconclusive" {
				c.Model = sr.Model
			} else {
				c.Kind = CaveatIntOverflowUndecided
				c.Detail = "could not decide whether every int + - * stays in the int64 range: " + r.Reason
			}
			vr.Caveats = append(vr.Caveats, c)
			continue
		}
		switch r.Status {
		case "counterexample":
			if ob.DependsOnUninterpreted {
				r = uninterpretedInconclusive(r, funcName, enc.Uninterpreted, sr, opts.Verbose)
				r.Reason = obligationReason(ob.Callee) + "; " + r.Reason
				break
			}
			r.ReasonCode = obligationCode(ob.Callee)
			r.Reason = obligationReason(ob.Callee)
		case "inconclusive":
			r.Reason = obligationReason(ob.Callee) + " only under bounded unrolling; " + r.Reason
		case "unknown":
			if ob.DependsOnUninterpreted {
				r = uninterpretedUndecided(r, enc.Uninterpreted)
			}
			r.Reason = fmt.Sprintf("could not decide %s: %s", obligationLabel(ob.Callee), r.Reason)
		}
		r.Duration += vr.Duration
		r.RequiresChecked = vr.RequiresChecked
		return r
	}
	return vr
}

func obligationLabel(name string) string {
	switch name {
	case obligationInt64:
		return "whether every int + - * stays in the int64 range"
	case obligationNaN:
		return "whether a float operation can produce NaN or ±Infinity"
	}
	return fmt.Sprintf("whether every call to %s meets its requires", name)
}

func obligationReason(name string) string {
	switch name {
	case obligationInt64:
		return "an int + - * may leave the int64 range (--int-model=int64)"
	case obligationNaN:
		return "a float operation may produce NaN or ±Infinity (division or remainder by zero, sqrt of a negative, log of a non-positive)"
	}
	return requiresReason(name)
}

func obligationCode(name string) verification.ReasonCode {
	switch name {
	case obligationInt64:
		return verification.IntOverflow
	case obligationNaN:
		return verification.FloatNaNOrInf
	}
	return verification.RequiresViolation
}

func requiresReason(callee string) string {
	return fmt.Sprintf("a call to %s may break its requires", callee)
}

var realSort = regexp.MustCompile(`\bReal\b`)

// floatCaveat notes, on a verified result whose encoding uses floats, that its
// float inputs are assumed to be ordinary numbers (D4 b′).
func floatCaveat(vr VerifyResult, smtlib string) VerifyResult {
	if vr.Status == "verified" && realSort.MatchString(smtlib) {
		vr.Caveats = append(vr.Caveats, Caveat{Kind: CaveatFloatInputs, Detail: "float inputs are assumed to be ordinary numbers (not NaN or ±Infinity); float operations that could produce them are checked"})
	}
	return vr
}
