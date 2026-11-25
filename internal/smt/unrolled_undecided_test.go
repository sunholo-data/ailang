package smt

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/verification"
)

// An unknown reached through an unrolled recursive helper was a skip before
// the helper was encoded; it must not become a new failure (docparse
// referralParseEpoch: a nonlinear digit fold over referralDigits timed out).
func TestUnrolledUndecidedIsASkip(t *testing.T) {
	vr := unrolledUndecided(VerifyResult{Function: "f", Status: "unknown", Reason: "solver timeout"}, []string{"digits"}, false)
	if vr.Status != "skipped" || vr.ReasonCode != verification.UnsupportedEncoding || !strings.Contains(vr.Reason, "digits") || !strings.Contains(vr.Reason, "solver timeout") {
		t.Fatalf("got %+v", vr)
	}
}

// The same holds for a map/filter/foldl chain specialised to a bounded depth
// (docparse editApply_*: a foldl over [TableCell] timed out once its
// accumulator record had a sort). With no bounded unrolling the unknown stands.
func TestHOFUndecidedIsASkipAndPlainUnknownStands(t *testing.T) {
	vr := unrolledUndecided(VerifyResult{Function: "f", Status: "unknown", Reason: "solver timeout"}, nil, true)
	if vr.Status != "skipped" || !strings.Contains(vr.Reason, "map/filter/foldl chains") || !strings.Contains(vr.Reason, "solver timeout") {
		t.Fatalf("hof: got %+v", vr)
	}
	vr = unrolledUndecided(VerifyResult{Function: "f", Status: "unknown", Reason: "solver unknown"}, nil, false)
	if vr.Status != "unknown" || vr.Reason != "solver unknown" {
		t.Fatalf("no unrolling: got %+v", vr)
	}
}
