package smt_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/verification"
)

// M3 (M-VERIFY-CALL-SITE-REQUIRES, decision D1(a)): a call that may break its
// callee's requires is a counterexample with reason requires_violation. Before
// it, caller and viaLetBad were VERIFIED (S1): the callee's requires was never
// checked at the call.
func TestCallSiteRequiresAreChecked(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("testdata", "call_requires"))
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	cases := map[string]map[string]struct {
		status string
		code   verification.ReasonCode
		reason string
	}{
		"s1.ail": {
			"one":       {"verified", "", ""},
			"caller":    {"counterexample", verification.RequiresViolation, "call to one"},
			"guarded":   {"verified", "", ""}, // the branch implies the requires
			"viaLet":    {"verified", "", ""}, // the let-bound value meets it
			"viaLetBad": {"counterexample", verification.RequiresViolation, "call to one"},
		},
		"cr.ail": {
			// Contract-path callee (body not encodable): checked the same way.
			"useOk":  {"verified", "", ""},
			"useBad": {"counterexample", verification.RequiresViolation, "call to wordsAtLeast"},
			// A requires with no encoding: the call cannot be checked, so the
			// caller is skipped, never verified.
			"useOpaque": {"skipped", verification.UnsupportedEncoding, `requires of "opaque" cannot be encoded`},
			// describe's own ensures calls one(n) unguarded; that is describe's
			// contract, not useDescribe's call, so it is not the caller's to discharge.
			"useDescribe": {"verified", "", ""},
		},
	}
	for file, want := range cases {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		report, _ := verifySource(t, file, string(src))
		got := map[string]int{}
		for i, r := range report.Results {
			got[r.Function] = i
		}
		for fn, w := range want {
			i, ok := got[fn]
			if !ok {
				t.Errorf("%s %s: no result", file, fn)
				continue
			}
			r := report.Results[i]
			if r.Status != w.status || (w.code != "" && r.ReasonCode != w.code) || !strings.Contains(r.Reason, w.reason) {
				t.Errorf("%s %s: %s %s %q, want %s %s containing %q", file, fn, r.Status, r.ReasonCode, r.Reason, w.status, w.code, w.reason)
			}
		}
	}
}
