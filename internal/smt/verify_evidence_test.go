package smt

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/verification"
)

func TestSolverEvidenceReasonsPreserveStatuses(t *testing.T) {
	for _, tc := range []struct {
		status SolverStatus
		reason verification.ReasonCode
		want   string
	}{
		{StatusVerified, "", "verified"}, {StatusCounterexample, "", "counterexample"},
		{StatusUnknown, verification.SolverUnknown, "unknown"}, {StatusUnknown, verification.SolverTimeout, "unknown"},
		{StatusError, verification.VerifierError, "error"},
	} {
		vr := solverVerifyResult("f", &SolverResult{Status: tc.status, ReasonCode: tc.reason, Error: "detail"})
		if vr.Status != tc.want {
			t.Fatalf("legacy status changed: %+v", vr)
		}
		if tc.reason != "" && vr.ReasonCode != tc.reason {
			t.Fatalf("reason dropped: %+v", vr)
		}
	}
}

func TestUnresolvedEvidenceIsUnsupportedEncoding(t *testing.T) {
	r := UnresolvedTypeVerifyResult("f", ErrUnresolvableTypes)
	if r.Status != "skipped" || r.ReasonCode != verification.UnsupportedEncoding {
		t.Fatalf("got %+v", r)
	}
}

func TestEvidenceNumericModelMatchesEncoder(t *testing.T) {
	e := verification.EncodedEvidence()
	for name, want := range map[string]string{"float": e.NumericModel.FloatSort, "int": e.NumericModel.IntSort} {
		got, err := mapTCon(name)
		if err != nil || got != want {
			t.Fatalf("model metadata diverged from encoder: %s %s %v", name, got, err)
		}
	}
}
