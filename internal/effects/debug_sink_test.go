package effects

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// sinkFixture returns a context holding one log line and, when withFail is
// set, one failed assertion — the shape every host flushes.
func sinkFixture(msg string, withFail bool) *DebugContext {
	d := NewDebugContext()
	d.Log(msg, "prog.ail:7")
	if withFail {
		d.Check(false, "boom", "prog.ail:9")
	}
	return d
}

// capturingLogf collects decorated lines the way a host's log.Printf would,
// so the test can tell the two output paths apart.
func capturingLogf(buf *bytes.Buffer) func(string, ...any) {
	return func(format string, args ...any) {
		fmt.Fprintf(buf, "DECORATED:"+format+"\n", args...)
	}
}

func TestIsStructuredLine(t *testing.T) {
	cases := map[string]bool{
		`{"severity":"ERROR","message":"x"}`: true,
		`  {"severity":"ERROR"}`:             true,  // leading whitespace tolerated (Cloud Logging tolerates it)
		`{"severity":"ERROR"`:                false, // unterminated → text
		`{not json}`:                         false,
		`["a","b"]`:                          false, // an array is not a log object
		`plain text`:                         false,
		``:                                   false,
		`{}`:                                 true,
	}
	for in, want := range cases {
		if got := IsStructuredLine(in); got != want {
			t.Errorf("IsStructuredLine(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestSeverity(t *testing.T) {
	if got := Severity(`{"severity":"WARNING","message":"m"}`); got != "WARNING" {
		t.Errorf("Severity = %q, want WARNING", got)
	}
	if got := Severity(`plain`); got != "" {
		t.Errorf("Severity(plain) = %q, want empty", got)
	}
	if got := Severity(`{"message":"no severity"}`); got != "" {
		t.Errorf("Severity(no field) = %q, want empty", got)
	}
}

func TestSeverityLevel(t *testing.T) {
	cases := map[string]int{"DEBUG": 0, "TRACE": 0, "INFO": 1, "WARNING": 2, "ERROR": 3, "": 1, "BOGUS": 1}
	for in, want := range cases {
		if got := SeverityLevel(in); got != want {
			t.Errorf("SeverityLevel(%q) = %d, want %d", in, got, want)
		}
	}
}

// TestDebugSink_Flush is the exact-bytes table: every combination of
// structured/unstructured × label × MinLevel × Structured.
func TestDebugSink_Flush(t *testing.T) {
	const structured = `{"severity":"ERROR","message":"repro"}`
	const debugLine = `{"severity":"DEBUG","message":"low"}`
	const plain = `starting`
	const bracePlain = `{not json}`

	cases := []struct {
		name     string
		msg      string
		label    string
		minLevel int
		withFail bool
		strict   bool // Structured assertion output
		wantRaw  string
		wantDeco string
	}{
		{name: "structured verbatim", msg: structured, wantRaw: structured + "\n"},
		{name: "structured ignores label", msg: structured, label: "X", wantRaw: structured + "\n"},
		{name: "structured passes filter", msg: structured, minLevel: 2, wantRaw: structured + "\n"},
		{name: "structured below filter dropped", msg: debugLine, minLevel: 2},
		{name: "structured without severity passes any filter", msg: `{"message":"nolevel"}`, minLevel: 3, wantRaw: `{"message":"nolevel"}` + "\n"},
		{name: "unstructured decorated", msg: plain, wantDeco: "DECORATED:starting\n"},
		{name: "unstructured keeps label", msg: plain, label: "X", wantDeco: "DECORATED:[X] starting\n"},
		{name: "unstructured passes filter (no severity)", msg: plain, minLevel: 3, wantDeco: "DECORATED:starting\n"},
		{name: "brace-prefixed invalid JSON is decorated never dropped", msg: bracePlain, minLevel: 3, wantDeco: "DECORATED:{not json}\n"},
		{name: "assertion text mode", msg: plain, withFail: true,
			wantDeco: "DECORATED:starting\nDECORATED:[ASSERT FAIL] boom at prog.ail:9\n"},
		{name: "assertion text mode with label", msg: plain, label: "X", withFail: true,
			wantDeco: "DECORATED:[X] starting\nDECORATED:[X] [ASSERT FAIL] boom at prog.ail:9\n"},
		{name: "assertion structured mode", msg: structured, withFail: true, strict: true,
			wantRaw: structured + "\n" + `{"severity":"ERROR","message":"assertion failed: boom","location":"prog.ail:9","source":"Debug.check"}` + "\n"},
		{name: "assertion structured survives NONE level", msg: structured, minLevel: 4, withFail: true, strict: true,
			wantRaw: `{"severity":"ERROR","message":"assertion failed: boom","location":"prog.ail:9","source":"Debug.check"}` + "\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var raw, deco bytes.Buffer
			d := sinkFixture(tc.msg, tc.withFail)
			DebugSink{W: &raw, Logf: capturingLogf(&deco), MinLevel: tc.minLevel, Label: tc.label, Structured: tc.strict}.Flush(d)
			if raw.String() != tc.wantRaw {
				t.Errorf("raw:\n got %q\nwant %q", raw.String(), tc.wantRaw)
			}
			if deco.String() != tc.wantDeco {
				t.Errorf("decorated:\n got %q\nwant %q", deco.String(), tc.wantDeco)
			}
			if out := d.Collect(); len(out.Logs) != 0 || len(out.Assertions) != 0 {
				t.Errorf("Flush did not Reset the context: %+v", out)
			}
		})
	}
}

// The structured assertion line must be built with json.Marshal — a message
// containing quotes or newlines must still yield one valid JSON line.
func TestDebugSink_StructuredAssertionEscapes(t *testing.T) {
	d := NewDebugContext()
	d.Check(false, "he said \"no\"\nthen left", "a.ail:1")
	var raw bytes.Buffer
	DebugSink{W: &raw, Structured: true}.Flush(d)
	line := strings.TrimSuffix(raw.String(), "\n")
	if strings.Contains(line, "\n") || !json.Valid([]byte(line)) {
		t.Fatalf("assertion line is not one valid JSON line: %q", raw.String())
	}
	var got map[string]string
	if err := json.Unmarshal([]byte(line), &got); err != nil {
		t.Fatal(err)
	}
	if got["severity"] != "ERROR" || got["message"] != "assertion failed: he said \"no\"\nthen left" {
		t.Errorf("unexpected fields: %v", got)
	}
}

// Logf nil falls back to writing decorated lines to W; nil context is a no-op.
func TestDebugSink_Defaults(t *testing.T) {
	var raw bytes.Buffer
	DebugSink{W: &raw, Label: "L"}.Flush(sinkFixture("plain", false))
	if raw.String() != "[L] plain\n" {
		t.Errorf("nil Logf fallback: got %q", raw.String())
	}
	raw.Reset()
	DebugSink{W: &raw}.Flush(nil)
	if raw.Len() != 0 {
		t.Errorf("nil context wrote %q", raw.String())
	}
}
