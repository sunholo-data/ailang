package smt_test

import "testing"

// findIndex with a literal (or named) predicate is specialised like any: an
// index chain, the Option rebuilt at the call site, and beyond the unrolling
// depth an unknown index that is still -1 or a position. std/json's get,
// getString and friends reach it through the private findInList, which the
// callee collector now finds in std/json rather than in the caller.
// Before, every one was skipped. A false contract is never verified; this
// harness does not replay witnesses, so a refutation through the unrolling is
// inconclusive here (the CLI's replay confirms the parameterised ones).
func TestFindIndexAndJSONAccessors(t *testing.T) {
	want := map[string]string{
		"firstMatch": "verified", "firstMatchFalse": "inconclusive",
		"notFound": "verified", "inRange": "verified",
		"alwaysFoundFalse": "inconclusive",
		"readBack":         "verified", "missingKey": "verified",
		"missingKeyFalse": "inconclusive",
		"sameSearch":      "verified", // only findIndex seeds Option here
	}
	report := verifyFixture(t, "find_index.ail")
	for _, r := range report.Results {
		if w, ok := want[r.Function]; ok {
			delete(want, r.Function)
			if r.Status != w {
				t.Errorf("%s: %s (%s), want %s", r.Function, r.Status, r.Reason, w)
			}
		}
	}
	for fn := range want {
		t.Errorf("%s: no result", fn)
	}
}
