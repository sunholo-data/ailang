package main

import (
	"slices"
	"testing"
)

// Reasons as `ailang verify` reports them on the downstream corpora.
func TestBlockersStripTheEncoderChain(t *testing.T) {
	for _, c := range []struct {
		reason string
		want   []string
	}{
		{`SMT declaration closure could not resolve a required sort (cannot encode function body: let value: unresolvable SMT declaration types: calls user function "eventJson" that is not SMT-encodable in this context: calls user function "kv" that is not SMT-encodable in this context: record construction: no declared record type has fields [key value] (a field type, e.g. a tuple, may not be encodable))`,
			[]string{anonymousRecord}},
		{`unsupported construct in SMT encoding: resolving cross-function calls: unsupported construct in SMT encoding: callee "dist": the specialised map chain does not encode: encoding unrolled body at depth 1: let body: let body: unsupported construct in SMT encoding: match on list patterns (x :: rest) is outside the decodable fragment`,
			[]string{"match on list patterns (x :: rest) is outside the decodable fragment"}},
		{`Function "f" has effects; Function "f" calls "isErr" whose signature uses an unencodable type "Result[a, e]"`,
			[]string{`Function "_" has effects`, "callee signature uses an unencodable type: Result"}},
		{`SMT declaration closure could not resolve a required sort (cannot encode function body: unresolvable SMT declaration types: calls user function "weighted" that is not SMT-encodable in this context: Callee "weighted" is recursive; Callee "weighted" uses unencodable types)`,
			[]string{`Callee "_" is recursive`, `Callee "_" uses unencodable types`}},
		{`SMT declaration closure could not resolve a required sort (cannot encode function body: unresolvable SMT declaration types: calls user function "sha256Hex" that is not SMT-encodable in this context: Callee "sha256Hex" uses an unencodable builtin: _crypto_sha256hex)`,
			[]string{"unencodable builtin: _crypto_sha256hex"}},
		{`Function "f" uses an unencodable builtin: std/string.stringToInt; Function "f" has effects`,
			[]string{"unencodable builtin: std/string.stringToInt", `Function "_" has effects`}},
		{`unsupported construct in SMT encoding: resolving cross-function calls: unsupported construct in SMT encoding: circular function call detected: a → b → a`,
			[]string{mutualRecursion}},
		{`SMT declaration closure could not resolve a required sort (cannot encode function body: let body: unresolvable SMT declaration types: calls user function "pong" that is not SMT-encodable in this context: calls user function "ping" that is not SMT-encodable in this context: it is defined after its caller in dependency order (the helpers are mutually recursive))`,
			[]string{mutualRecursion}},
		{`cannot encode function body: constructor arg 0: unresolvable SMT declaration types: calls user function "kv" that is not SMT-encodable in this context: record construction: no declared record type has fields [key value]`,
			[]string{anonymousRecord}},
	} {
		if got := blockers(c.reason); !slices.Equal(got, c.want) {
			t.Errorf("blockers(%.60q…)\n got %q\nwant %q", c.reason, got, c.want)
		}
	}
}

func TestBlockerCountsSeparateSoleFromShared(t *testing.T) {
	rows := []row{
		{Status: "skipped", Blockers: []string{"effects"}},
		{Status: "skipped", Blockers: []string{"effects", "tuple"}},
		{Status: "skipped", Reason: `Function "g" has effects`}, // banked before blockers were recorded
		{Status: "verified", Blockers: []string{"effects"}},
	}
	anyOf, sole := blockerCounts(rows)
	if anyOf["effects"] != 2 || sole["effects"] != 1 || anyOf["tuple"] != 1 || sole["tuple"] != 0 || sole[`Function "_" has effects`] != 1 {
		t.Fatalf("any %v, sole %v", anyOf, sole)
	}
}
