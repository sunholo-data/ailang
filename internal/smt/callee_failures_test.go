package smt_test

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/smt"
)

// A helper whose body does not encode is called opaquely, and the VERIFIED
// result's caveat names what blocks it at the end of the chain (these were
// skips, each naming the same chain, before opaque helpers).
func TestOpaqueHelperCaveatNamesTheBlocker(t *testing.T) {
	// Each fragment must appear, in this order, in the helpers_not_checked caveat.
	want := map[string][]string{
		// top → mid → leaf, whose match on a tuple does not encode.
		"top": {"leaf (", "tuple expressions cannot be encoded"},
		// A helper's unencodable builtin is named, not "uses unencodable types".
		"viaParse": {"parsed (", `Callee "parsed" uses an unencodable builtin: std/string.stringToFloat`},
	}
	report := verifyFixture(t, "callee_chain.ail")
	status := map[string]string{}
	for _, r := range report.Results {
		status[r.Function] = r.Status
		frags, ok := want[r.Function]
		if !ok {
			continue
		}
		detail := ""
		for _, c := range r.Caveats {
			if c.Kind == smt.CaveatOpaqueHelpers {
				detail = c.Detail
			}
		}
		at := 0
		for _, f := range frags {
			i := strings.Index(detail[at:], f)
			if r.Status != "verified" || i < 0 {
				t.Errorf("%s: %s %q\nwant verified, with %q in order in caveat %q", r.Function, r.Status, r.Reason, frags, detail)
				break
			}
			at += i + len(f)
		}
	}
	// ping → pong → ping verifies through ping's own contract; topFalse
	// depends on leaf's value, so its model is inconclusive.
	for fn, w := range map[string]string{"top": "verified", "viaParse": "verified", "ping": "verified", "topFalse": "inconclusive"} {
		if status[fn] != w {
			t.Errorf("%s: status %q, want %q", fn, status[fn], w)
		}
	}
}

// Records built in helper bodies and in accumulator literals have sorts: std/json
// kv's {key, value} (declared by Json's JObject field) and a named alias seeded
// from a literal. Each verified contract has a false twin that must be refuted.
// Before, every one was skipped with "no declared record type has fields".
func TestRecordsBuiltInHelpersAndLiterals(t *testing.T) {
	want := map[string]string{
		"viaKv": "verified", "viaKvFalse": "counterexample",
		"initial": "verified", "initialFalse": "counterexample",
		"viaFold":  "skipped",
		"choiceOf": "verified", "choiceOfFalse": "counterexample",
	}
	report := verifyFixture(t, "records_in_helpers.ail")
	seen := map[string]bool{}
	for _, r := range report.Results {
		w, ok := want[r.Function]
		if !ok {
			continue
		}
		seen[r.Function] = true
		if r.Status != w {
			t.Errorf("%s: %s (%s), want %s", r.Function, r.Status, r.Reason, w)
		}
		if strings.Contains(r.Reason, "no declared record type") {
			t.Errorf("%s: still blocked on the record sort: %s", r.Function, r.Reason)
		}
		if r.Function == "viaFold" && !strings.Contains(r.Reason, "map/filter/foldl chains") {
			t.Errorf("viaFold: reason %q, want the bounded-unrolling skip", r.Reason)
		}
	}
	for fn := range want {
		if !seen[fn] {
			t.Errorf("%s: no result", fn)
		}
	}
}
