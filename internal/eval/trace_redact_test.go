package eval

import (
	"strings"
	"testing"
)

func str(s string) *StringValue { return &StringValue{Value: s} }

func header(name, value string) *RecordValue {
	return &RecordValue{Fields: map[string]Value{"name": str(name), "value": str(value)}}
}

// G7: every credential shape a program can hand to an effect is withheld in
// the trace rendering, and nothing else changes.
func TestShowTraceBounded_WithholdsCredentials(t *testing.T) {
	const tok = "tok-4b1c9e"
	cases := map[string]Value{
		"stream header record": &RecordValue{Fields: map[string]Value{"headers": &ListValue{Elements: []Value{
			header("Authorization", "Bearer "+tok), header("X-Trace", "keep"),
		}}}},
		"json header entry": &RecordValue{Fields: map[string]Value{"key": str("x-api-key"), "value": &TaggedValue{CtorName: "JString", Fields: []Value{str(tok)}}}},
		"pair":              &TupleValue{Elements: []Value{str("Cookie"), str("session=" + tok)}},
		"field":             &RecordValue{Fields: map[string]Value{"proxy_authorization": str(tok), "path": str("/keep")}},
		"bare bearer":       &ListValue{Elements: []Value{str("Bearer " + tok)}},
		"basic":             str("basic " + tok),
	}
	for name, v := range cases {
		out := ShowTraceBounded(v, 1024)
		if strings.Contains(out, tok) {
			t.Errorf("%s: trace rendering leaks the credential: %s", name, out)
		}
		if !strings.Contains(out, RedactedMarker) {
			t.Errorf("%s: no redaction marker: %s", name, out)
		}
		if !strings.Contains(v.String(), tok) {
			t.Errorf("%s: String() must be unchanged (redaction is trace-only)", name)
		}
	}
	keep := ShowTraceBounded(cases["stream header record"], 0)
	if !strings.Contains(keep, "keep") || !strings.Contains(keep, "Authorization") {
		t.Errorf("header names and other values must survive: %s", keep)
	}
}

// Exact names, not substrings: usage counters and ordinary words stay.
func TestShowTraceBounded_KeepsNonCredentials(t *testing.T) {
	v := &RecordValue{Fields: map[string]Value{
		"input_tokens": &IntValue{Value: 42},
		"keyboard":     str("qwerty"),
		"note":         str("Bearer"), // the bare word, no credential after it
		"pair":         &TupleValue{Elements: []Value{str("Accept"), str("text/plain")}},
	}}
	if got, want := ShowTraceBounded(v, 0), v.String(); got != want {
		t.Fatalf("non-credential value changed:\n got %s\nwant %s", got, want)
	}
	long := &ListValue{Elements: []Value{str(strings.Repeat("x", 100))}}
	if got, want := ShowTraceBounded(long, 10), ShowBounded(long, 10); got != want {
		t.Fatalf("bounding differs: %q vs %q", got, want)
	}
}
