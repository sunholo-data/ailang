package smt_test

import "testing"

// Go-backed pure builtins (JSON text, SHA-256, XML escaping) are uninterpreted
// functions with the facts their implementations guarantee. A contract that
// needs no more than those facts is proved; one that needs more is
// inconclusive here, and refuted only by witness replay on the real builtin
// (the CLI replays; this harness does not). Before, every one was skipped.
func TestOpaqueBuiltinsProveWhatTheirFactsGive(t *testing.T) {
	want := map[string]string{
		"framed": "verified", "objectText": "verified", "emptyArray": "verified",
		"objectTextFalse": "inconclusive",
		"digest":          "verified", "digestFalse": "inconclusive",
		"escaped": "verified", "plainUnchanged": "verified",
		"parses": "verified", "parsesFalse": "inconclusive",
		"emptyHoisted": "verified",
	}
	report := verifyFixture(t, "opaque_builtins.ail")
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
	}
	for fn := range want {
		if !seen[fn] {
			t.Errorf("%s: no result", fn)
		}
	}
}

// A record alias with the same field names as an inline record of an ADT
// field (std/json's {key, value: Json}) no longer captures the inline
// record's literals: kv built a Pair holding a Json, and Z3 rejected it.
func TestRecordLiteralPicksBetweenSameFieldNames(t *testing.T) {
	report := verifyFixture(t, "record_field_collision.ail")
	want := map[string]string{"viaKv": "verified", "pair": "verified"}
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
