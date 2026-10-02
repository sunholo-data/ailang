package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// MOD015 (#1467, ruling 2026-10-02): a selectively imported name that the
// module also defines at module level is a compile error on every front door —
// check, run (evaluator and VM) and test — instead of the import silently
// winning (q.ail used to print 2).
func TestImportLocalCollisionMOD015(t *testing.T) {
	bin := buildAilang(t)
	src := filepath.Join("cmd", "ailang", "testdata", "importcollide", "q.ail")
	modes := [][]string{
		{"check", src},
		{"run", "--quiet", "--caps", "IO", "--entry", "main", src},
		{"run", "--quiet", "--bytecode", "--caps", "IO", "--entry", "main", src},
		{"test", src},
	}
	for _, mode := range modes {
		out, stderr, code := runWithStdin(t, bin, "", mode...)
		all := out + stderr
		if code == 0 {
			t.Errorf("%v: want a compile error, got exit 0\n%s", mode[:len(mode)-1], all)
			continue
		}
		// Both sites (import at 2:1, the local func at 6:6) and the
		// alias fix. Positions are matched without the directory so the test
		// holds for Windows path separators too.
		for _, want := range []string{"MOD015", "'tick'", "q.ail:2:1", "q.ail:6:6", "tick as aTick"} {
			if !strings.Contains(all, want) {
				t.Errorf("%v: output missing %q\n%s", mode[:len(mode)-1], want, all)
			}
		}
	}
}

// The suggested fix (alias the import) compiles, runs and tests.
func TestImportLocalCollisionFixedByAlias(t *testing.T) {
	bin := buildAilang(t)
	src := filepath.Join("cmd", "ailang", "testdata", "importcollide", "fixed.ail")
	out, stderr, code := runWithStdin(t, bin, "", "run", "--quiet", "--caps", "IO", "--entry", "main", src)
	if code != 0 || strings.TrimSpace(out) != "103" {
		t.Fatalf("run: exit=%d out=%q want 103\nstderr: %s", code, strings.TrimSpace(out), stderr)
	}
	for _, mode := range [][]string{{"check", src}, {"test", src}} {
		if out, stderr, code := runWithStdin(t, bin, "", mode...); code != 0 {
			t.Errorf("%v: exit=%d\n%s%s", mode[:len(mode)-1], code, out, stderr)
		}
	}
}
