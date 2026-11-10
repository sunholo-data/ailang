package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// M-V1-MATCH-EXHAUSTIVENESS M0: a match none of whose arms matches is RT002
// with the match's file:line:col and the scrutinee's shape, on the evaluator
// and on the strict bytecode VM alike, with identical text.
//
// Before the fix the evaluator said "no pattern matched in match expression"
// with no position, and the VM had no trap at all: the match fell off its end
// and yielded Unit. Measured on the base binary: adt, guardFails, listEmpty
// and lambdaMatch exited 0 with no output; adtArith and letBound died later in
// `arith ADD: type mismatch Unit vs Int` at the caller's line.
func TestMatchFailParity(t *testing.T) {
	bin := buildAilang(t)
	src := filepath.Join("cmd", "ailang", "testdata", "matchfail", "match_fail.ail")
	file := filepath.ToSlash(src)
	cases := []struct {
		entry, want string
	}{
		// SwitchStmt path: Default is the trap.
		{"adt", "at " + file + ":12:35 (scrutinee constructor: Blue)"},
		{"adtArith", "at " + file + ":12:35 (scrutinee constructor: Blue)"},
		// SwitchStmt path: a failed guard falls back to the trap.
		{"guardFails", "at " + file + ":21:36 (scrutinee constructor: Has)"},
		// if-chain path; shape only, never contents.
		{"listEmpty", "at " + file + ":28:36 (scrutinee: empty list)"},
		// value position (flattenValue rewrites returns, not the trap).
		{"letBound", "at " + file + ":36:11 (scrutinee constructor: Green)"},
		// inside a lambda: the prototype has no file, the trap carries one.
		{"lambdaMatch", "at " + file + ":46:15 (scrutinee constructor: Blue)"},
	}
	engines := map[string][]string{
		"evaluator": {"run", "--quiet"},
		"vm":        {"run", "--quiet", "--bytecode", "--strict-bytecode"},
	}
	for _, c := range cases {
		for engine, base := range engines {
			args := append(append([]string{}, base...), "--entry", c.entry, src)
			stdout, stderr, code := runWithStdin(t, bin, "", args...)
			stderr = filepath.ToSlash(stderr)
			if code != 1 {
				t.Errorf("%s/%s: exit %d, want 1\nstdout: %s\nstderr: %s", engine, c.entry, code, stdout, stderr)
			}
			if strings.Contains(stderr, "panic:") || strings.Contains(stderr, "goroutine ") {
				t.Errorf("%s/%s: Go panic leaked:\n%s", engine, c.entry, stderr)
			}
			if strings.Contains(stderr, "Unit") {
				t.Errorf("%s/%s: a Unit value escaped the match:\n%s", engine, c.entry, stderr)
			}
			want := "RT002: no arm matched in match " + c.want
			if !strings.Contains(stderr, want) {
				t.Errorf("%s/%s: stderr %q\nwant it to contain %q", engine, c.entry, stderr, want)
			}
		}
	}
}
