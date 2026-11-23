package smt_test

import (
	"os"
	"path/filepath"
	"testing"
)

// M-VERIFY-SOUNDNESS-COVERAGE M2 (#1375, #215): literal-lambda map / filter /
// foldl / any / flatMap callers verify over any element sort, and a
// specialisation is a bounded unrolling — so a model is INCONCLUSIVE, never a
// VIOLATION that may rest on the uninterpreted frontier, and a false contract
// is never VERIFIED.
func TestHOFCallersVerifyOverAnyElementSort(t *testing.T) {
	path := filepath.Join("testdata", "hof", "hof_types.ail")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	report, _ := verifySource(t, path, string(src))
	want := map[string]string{
		// true and within the unroll depth (requires bounds the list)
		"shortSame": "verified", "goodSum": "verified", "strs": "verified", "floats": "verified",
		"intToStr": "verified", "lens": "verified", "applicableCount": "verified",
		"hasNeg": "verified", "dup": "verified", "twoMaps": "verified",
		// lambdas capturing an enclosing parameter or let; field access on the element
		"forSession": "verified", "allAbove": "verified", "shifted": "verified", "total": "verified",
		"allAboveBad": "inconclusive",
		// HOFs inside contracts: ensures (capturing result) and requires
		// named function arguments (eta-expanded)
		"doubled": "verified", "pos": "verified", "sumNat": "verified",
		"doubledBad": "inconclusive", "anyPosBad": "inconclusive",
		"applyParam": "skipped",
		"addOk":      "verified", "firstOk": "verified", "dropAll": "inconclusive", "firstOkBad": "inconclusive",
		// true for lists of ANY length: proved through the frontier's shape
		// (map's frontier is seq.map, so it preserves length by construction)
		"sameLen": "verified", "filterNoLonger": "verified", "filterSameLenBad": "inconclusive",
		// false: never verified
		"badLen": "inconclusive", "badFilter": "inconclusive", "badSum": "inconclusive",
		"strsBad": "inconclusive", "floatsBad": "inconclusive", "intToStrBad": "inconclusive",
		"lensBad": "inconclusive", "applicableBad": "inconclusive", "hasNegBad": "inconclusive",
		"dupBad": "inconclusive",
	}
	got := make(map[string]string)
	for _, r := range report.Results {
		got[r.Function] = r.Status
		if r.Status == "skipped" {
			t.Logf("%s skipped: %s", r.Function, r.Reason)
		}
		if r.Status == "verified" && r.BoundedDepth == 0 {
			t.Errorf("%s: a HOF-specialised proof must carry its bounded depth", r.Function)
		}
	}
	for fn, status := range want {
		if got[fn] != status {
			t.Errorf("%s: status %q, want %q", fn, got[fn], status)
		}
	}
}
