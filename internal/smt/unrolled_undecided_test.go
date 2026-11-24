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
	vr := unrolledUndecided(VerifyResult{Function: "f", Status: "unknown", Reason: "solver timeout"}, []string{"digits"})
	if vr.Status != "skipped" || vr.ReasonCode != verification.UnsupportedEncoding || !strings.Contains(vr.Reason, "digits") || !strings.Contains(vr.Reason, "solver timeout") {
		t.Fatalf("got %+v", vr)
	}
}
