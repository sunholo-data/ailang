package smt

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/verification"
)

// checkCallObligations runs the call-site precondition queries of a verified
// result (M3, decision D1(a)). The postcondition was proved assuming each
// callee's requires at its calls; the result stays verified only if every
// such call is shown to meet it.
func checkCallObligations(vr VerifyResult, funcName string, enc *EncodeResult, cfg SolverConfig, boundedDepth int, verbose bool) VerifyResult {
	for _, ob := range enc.Obligations {
		vr.RequiresChecked = append(vr.RequiresChecked, ob.Callee)
	}
	for _, ob := range enc.Obligations {
		sr, err := Solve(ob.SMTLib, cfg)
		if err != nil {
			return VerifyResult{Function: funcName, Status: "error", Reason: fmt.Sprintf("solver error checking calls to %s: %v", ob.Callee, err), ReasonCode: verification.VerifierError}
		}
		r := solverVerifyResult(funcName, sr, boundedDepth, verbose)
		switch r.Status {
		case "verified":
			continue
		case "counterexample":
			if len(enc.Uninterpreted) > 0 {
				r = uninterpretedInconclusive(r, funcName, enc.Uninterpreted, sr, verbose)
				r.Reason = requiresReason(ob.Callee) + "; " + r.Reason
				return r
			}
			r.ReasonCode = verification.RequiresViolation
			r.Reason = requiresReason(ob.Callee)
		case "inconclusive":
			r.Reason = requiresReason(ob.Callee) + " only under bounded unrolling; " + r.Reason
		case "unknown":
			if len(enc.Uninterpreted) > 0 {
				r = uninterpretedUndecided(r, enc.Uninterpreted)
			}
			r.Reason = fmt.Sprintf("could not decide whether every call to %s meets its requires: %s", ob.Callee, r.Reason)
		}
		r.Duration += vr.Duration
		r.RequiresChecked = vr.RequiresChecked
		return r
	}
	return vr
}

func requiresReason(callee string) string {
	return fmt.Sprintf("a call to %s may break its requires", callee)
}
