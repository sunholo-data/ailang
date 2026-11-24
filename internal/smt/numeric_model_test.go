package smt_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/sunholo-data/ailang/internal/smt"
	"github.com/sunholo-data/ailang/internal/verification"
)

// M4 (M-VERIFY-NUMERIC-MODEL, decisions D2 a′ and D4 b′). Before it, inc was a
// bare VERIFIED although inc(2^63-1) wraps, and ratioBad / rootBad were
// VERIFIED although they divide by zero / take sqrt of a negative.
func TestNumericModel(t *testing.T) {
	path := filepath.Join("testdata", "numeric", "num_model.ail")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	type want struct {
		status  string
		code    verification.ReasonCode
		caveats []string
	}
	floatNote := []string{smt.CaveatFloatInputs}
	modes := map[string]map[string]want{
		verification.IntModelUnbounded: {
			"inc":        {"verified", "", []string{smt.CaveatIntOverflow}},
			"incBounded": {"verified", "", nil},
			// The field operand is an int64 value: the guard (in64 b.n) means
			// only the operation itself is checked.
			"fieldCopy": {"verified", "", nil},
			"ratio":     {"verified", "", floatNote},
			"ratioBad":  {"counterexample", verification.FloatNaNOrInf, nil},
			"root":      {"verified", "", floatNote},
			// sqrt is uninterpreted, but the obligation x >= 0 does not mention
			// it, so the model is a genuine violation, not inconclusive.
			"rootBad": {"counterexample", verification.FloatNaNOrInf, nil},
		},
		verification.IntModelInt64: {
			"inc":        {"counterexample", verification.IntOverflow, nil},
			"incBounded": {"verified", "", nil},
			"fieldCopy":  {"verified", "", nil},
		},
	}
	for mode, cases := range modes {
		report, _ := verifySourceWith(t, path, string(src), smt.VerifyOptions{IntModel: mode})
		if got := report.Evidence.NumericModel.IntOverflow; got != map[string]string{verification.IntModelUnbounded: "reported", verification.IntModelInt64: "checked"}[mode] {
			t.Errorf("%s: evidence int_overflow = %q", mode, got)
		}
		results := map[string]smt.VerifyResult{}
		for _, r := range report.Results {
			results[r.Function] = r
		}
		for fn, w := range cases {
			r := results[fn]
			var kinds []string
			for _, c := range r.Caveats {
				kinds = append(kinds, c.Kind)
			}
			if r.Status != w.status || r.ReasonCode != w.code || !slices.Equal(kinds, w.caveats) {
				t.Errorf("%s %s: %s %q caveats %v (reason %q); want %s %q caveats %v", mode, fn, r.Status, r.ReasonCode, kinds, r.Reason, w.status, w.code, w.caveats)
			}
		}
	}
}
