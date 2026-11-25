package smt_test

import (
	"strings"
	"testing"
)

// A skip reached through helpers names what blocks it at the end of the
// chain. Before, every one of these stopped at the first
// `calls user function "X" that is not SMT-encodable in this context`.
func TestSkipThroughHelpersNamesTheBlocker(t *testing.T) {
	// Each fragment must appear, in this order, in the function's reason.
	want := map[string][]string{
		// top → mid → leaf, whose list-pattern match does not encode.
		"top": {`calls user function "mid"`, `calls user function "leaf"`, "match on list patterns"},
		// ping → pong → ping: a cycle through the verified function.
		"ping": {`calls user function "pong"`, "the helpers are mutually recursive"},
		// getString → get → findInList, a private std/json helper.
		"named": {`calls user function "findInList"`, "was not collected as a callee"},
	}
	report := verifyFixture(t, "callee_chain.ail")
	seen := map[string]bool{}
	for _, r := range report.Results {
		frags, ok := want[r.Function]
		if !ok {
			continue
		}
		seen[r.Function] = true
		at := 0
		for _, f := range frags {
			i := strings.Index(r.Reason[at:], f)
			if r.Status != "skipped" || i < 0 {
				t.Errorf("%s: %s %q\nwant skipped, with %q in order", r.Function, r.Status, r.Reason, frags)
				break
			}
			at += i + len(f)
		}
	}
	for fn := range want {
		if !seen[fn] {
			t.Errorf("%s: no result", fn)
		}
	}
}
