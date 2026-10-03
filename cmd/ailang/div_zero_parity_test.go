package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// M-INT-DIV-ZERO-ERROR (#1449): `ailang run` reports integer / and % by zero
// as RT001 with the dividing line, on the evaluator and on the strict bytecode
// VM alike — exit 1, no Go panic. Before the fix the evaluator printed
// `panic: division by zero [recovered, repanicked]` and a Go stack for `/`.
func TestIntDivZeroParity(t *testing.T) {
	bin := buildAilang(t)
	src := filepath.Join("internal", "embed", "testdata", "divzero.ail")
	cases := []struct {
		entry, arg, op, loc string
	}{
		{"divBy", "0", "division", "divzero.ail:7"},
		{"modBy", "0", "modulo", "divzero.ail:9"},
		{"outer", "1", "division", "divzero.ail:7"},
		{"literal", "", "division", "divzero.ail:13"},
	}
	engines := map[string][]string{
		"evaluator": {"run", "--quiet"},
		"vm":        {"run", "--quiet", "--bytecode", "--strict-bytecode"},
	}
	for _, c := range cases {
		for engine, base := range engines {
			args := append(append([]string{}, base...), "--entry", c.entry)
			if c.arg != "" {
				args = append(args, "--args-json", c.arg)
			}
			args = append(args, src)
			stdout, stderr, code := runWithStdin(t, bin, "", args...)
			stderr = filepath.ToSlash(stderr)
			if code != 1 {
				t.Errorf("%s/%s: exit %d, want 1\nstdout: %s\nstderr: %s", engine, c.entry, code, stdout, stderr)
			}
			if strings.Contains(stderr, "panic:") || strings.Contains(stderr, "goroutine ") {
				t.Errorf("%s/%s: Go panic leaked:\n%s", engine, c.entry, stderr)
			}
			if !strings.Contains(stderr, "RT001: integer "+c.op+" by zero") || !strings.Contains(stderr, c.loc) {
				t.Errorf("%s/%s: stderr %q, want RT001 integer %s by zero at %s", engine, c.entry, stderr, c.op, c.loc)
			}
		}
	}
}

// A zero divisor inside a runtime-checked `ensures` is RT001 too, not a crash.
func TestIntDivZeroContract(t *testing.T) {
	bin := buildAilang(t)
	src := filepath.Join("internal", "embed", "testdata", "divzero.ail")
	_, stderr, code := runWithStdin(t, bin, "", "run", "--quiet", "--verify-contracts", "--entry", "ensuresDivides", "--args-json", "0", src)
	stderr = filepath.ToSlash(stderr)
	if code != 1 || strings.Contains(stderr, "panic:") || !strings.Contains(stderr, "RT001: integer division by zero at") || !strings.Contains(stderr, "divzero.ail:18") {
		t.Fatalf("exit=%d stderr=%s, want exit 1 and RT001 at divzero.ail:18", code, stderr)
	}
}
