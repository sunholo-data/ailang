package smt_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/smt"
	"github.com/sunholo-data/ailang/internal/verification"
)

// Obligations inside definitions (codegen_def_obligations.go): a helper's
// body, a map lambda and the function's own recursion carry their callee
// requires and numeric obligations to each call. Before it, useInv, useTwice
// and useNeedsBad were VERIFIED, useSquare had no overflow note, and recBad
// was VERIFIED although recBad(1) calls recBad(-1).
func TestObligationsInsideDefinitions(t *testing.T) {
	path := filepath.Join("testdata", "def_obligations", "def_obs.ail")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	type want struct {
		status  string
		code    verification.ReasonCode
		caveats []string
		reason  string
	}
	floatNote := []string{smt.CaveatFloatInputs}
	cases := map[string]want{
		"useSquare":      {"verified", "", []string{smt.CaveatIntOverflow}, ""},
		"useSquareSmall": {"verified", "", nil, ""},
		"useInv":         {"counterexample", verification.FloatNaNOrInf, nil, ""},
		"useInvGuarded":  {"verified", "", floatNote, ""},
		"useInvSafe":     {"verified", "", floatNote, ""},
		"useTwice":       {"counterexample", verification.FloatNaNOrInf, nil, ""},
		"useNeedsBad":    {"counterexample", verification.RequiresViolation, nil, "call to needsPos"},
		"useNeedsOk":     {"verified", "", nil, ""},
		"useLambda":      {"verified", "", []string{smt.CaveatIntOverflow, smt.CaveatObligationsBounded}, ""},
		"recOk":          {"verified", "", nil, ""},
		// Every model of a bounded unrolling is inconclusive until replay
		// decides it; the reason names the broken requires.
		"recBad": {"inconclusive", verification.BoundedInconclusive, nil, "call to recBad may break its requires"},
	}
	report, _ := verifySource(t, path, string(src))
	seen := map[string]bool{}
	for _, r := range report.Results {
		w, ok := cases[r.Function]
		if !ok {
			continue
		}
		seen[r.Function] = true
		if r.Status != w.status || r.ReasonCode != w.code {
			t.Errorf("%s: %s/%s, want %s/%s (%s)", r.Function, r.Status, r.ReasonCode, w.status, w.code, r.Reason)
		}
		var kinds []string
		for _, c := range r.Caveats {
			kinds = append(kinds, c.Kind)
		}
		if !slices.Equal(kinds, w.caveats) {
			t.Errorf("%s: caveats %v, want %v", r.Function, kinds, w.caveats)
		}
		if w.reason != "" && !strings.Contains(r.Reason, w.reason) {
			t.Errorf("%s: reason %q should mention %q", r.Function, r.Reason, w.reason)
		}
	}
	for fn := range cases {
		if !seen[fn] {
			t.Errorf("%s: no result", fn)
		}
	}
}
