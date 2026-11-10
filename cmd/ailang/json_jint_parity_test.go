package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// M-V1-STDLIB-SURFACE-POLISH B1–B4: JInt, getInt, decode trailing data and
// empty keys, on the evaluator and the strict bytecode VM. Both engines must
// print the same transcript, and it must be the expected one.
func TestJSONJIntParity(t *testing.T) {
	bin := buildAilang(t)
	src := filepath.Join("cmd", "ailang", "testdata", "jsonjint", "json_jint.ail")
	want, err := os.ReadFile(filepath.Join("testdata", "jsonjint", "json_jint.expected"))
	if err != nil {
		t.Fatal(err)
	}
	engines := map[string][]string{
		"evaluator": {"run", "--quiet", "--caps", "IO", "--entry", "main", src},
		"vm":        {"run", "--quiet", "--bytecode", "--strict-bytecode", "--caps", "IO", "--entry", "main", src},
	}
	// A Windows checkout may give the expected file CRLF line endings, and
	// println may write them; compare lines, not line terminators.
	lf := func(s string) string { return strings.ReplaceAll(s, "\r\n", "\n") }
	outs := map[string]string{}
	for engine, args := range engines {
		stdout, stderr, code := runWithStdin(t, bin, "", args...)
		if code != 0 {
			t.Fatalf("%s: exit %d\nstderr: %s", engine, code, stderr)
		}
		stdout = lf(stdout)
		outs[engine] = stdout
		if stdout != lf(string(want)) {
			t.Errorf("%s transcript differs from json_jint.expected:\n%s", engine, lineDiff(lf(string(want)), stdout))
		}
	}
	if outs["evaluator"] != outs["vm"] {
		t.Errorf("engines disagree:\n%s", lineDiff(outs["evaluator"], outs["vm"]))
	}
}

// lineDiff names the first differing line.
func lineDiff(a, b string) string {
	al, bl := strings.Split(a, "\n"), strings.Split(b, "\n")
	for i := 0; i < len(al) || i < len(bl); i++ {
		var x, y string
		if i < len(al) {
			x = al[i]
		}
		if i < len(bl) {
			y = bl[i]
		}
		if x != y {
			return "line " + strconv.Itoa(i+1) + ":\n  want: " + x + "\n  got:  " + y
		}
	}
	return "(no line differs)"
}
