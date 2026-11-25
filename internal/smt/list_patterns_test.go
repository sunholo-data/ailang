package smt_test

import "testing"

// A match on list patterns ([], [a, b], [x, ...rest], x :: rest) is a
// length-and-element ite chain; recursion through one unrolls like any other
// helper. Before, every one was skipped. A pattern variable named like a
// record field binds under its own symbol: `Some(p) => p.p` and
// `[p, ...rest] => p.p` were Z3 errors. A false contract is never verified;
// this harness does not replay witnesses, so a refutation through the
// unrolling is inconclusive here.
func TestListPatterns(t *testing.T) {
	want := map[string]string{
		"emptyOrOne": "verified", "emptyOrOneFalse": "counterexample",
		"headAndTail": "verified",
		"pairOnly":    "verified", "pairOnlyFalse": "counterexample",
		"startsWithOne": "verified", "headIsOne": "verified",
		"someHead":      "verified",
		"countPosSmall": "verified", "countPosSmallFalse": "inconclusive",
		"noNegWeight": "verified",
		"guardedHead": "verified", "guardedHeadFalse": "counterexample",
		"twoTotal": "verified", "pOf": "verified", "keyOf": "verified",
	}
	report := verifyFixture(t, "list_patterns.ail")
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
