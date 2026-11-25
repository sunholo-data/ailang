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
		// A helper's unencodable builtin is named, not "uses unencodable types".
		"viaParse": {`calls user function "parsed"`, `Callee "parsed" uses an unencodable builtin: std/string.stringToInt`},
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

// Records built in helper bodies and in accumulator literals have sorts: std/json
// kv's {key, value} (declared by Json's JObject field) and a named alias seeded
// from a literal. Each verified contract has a false twin that must be refuted.
// Before, every one was skipped with "no declared record type has fields".
func TestRecordsBuiltInHelpersAndLiterals(t *testing.T) {
	want := map[string]string{
		"viaKv": "verified", "viaKvFalse": "counterexample",
		"initial": "verified", "initialFalse": "counterexample",
		"viaFold": "skipped",
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
