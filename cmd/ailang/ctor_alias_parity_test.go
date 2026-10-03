package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// M-CTOR-PATTERN-ALIAS-AND-SCOPE (#1478): an aliased constructor import
// (`import std/option (None as Nada)`) binds in patterns AND expressions on
// every route, and a constructor pattern naming no known constructor is a
// compile error instead of a silently dead arm. Before the fix every alias row
// returned 0 (or failed "undefined variable"), the strict VM rejected the
// entries ("unknown tag Option.Nada"), and `unknown` compiled clean.

func TestCtorAliasParity1478(t *testing.T) {
	bin := buildAilang(t)
	dir := filepath.Join("cmd", "ailang", "testdata", "ctoralias")
	cases := []struct {
		file, entry, want string
	}{
		{"alias.ail", "aliasedNullary", "1"},
		{"alias.ail", "aliasedArgs", "42"},
		{"alias.ail", "aliasedExpr", "7"},
		{"alias.ail", "aliasedCall", "5"},
		{"alias.ail", "canonicalStillBound", "3"},
		{"transitive.ail", "transitiveNth", "20"},
		{"transitive.ail", "transitiveNone", "1"},
		{"clash.ail", "tripArrived", "1"},
		{"clash.ail", "tripDeparted", "0"},
		{"clash.ail", "localArrived", "2"},
		{"clash.ail", "aliasExpr", "3"},
	}
	for _, c := range cases {
		src := filepath.Join(dir, c.file)
		for _, mode := range [][]string{
			{"run", "--quiet", "--entry", c.entry, src},
			{"run", "--quiet", "--bytecode", "--strict-bytecode", "--entry", c.entry, src},
		} {
			out, stderr, code := runWithStdin(t, bin, "", mode...)
			if code != 0 || strings.TrimSpace(out) != c.want {
				t.Errorf("%s %v: exit=%d out=%q want %q\nstderr: %s", c.entry, mode[:len(mode)-1], code, strings.TrimSpace(out), c.want, stderr)
			}
		}
	}
}

func TestCtorPatternUnknownIsCompileError1478(t *testing.T) {
	bin := buildAilang(t)
	src := filepath.Join("cmd", "ailang", "testdata", "ctoralias", "unknown.ail")
	for _, args := range [][]string{
		{"check", src},
		{"run", "--quiet", "--entry", "unknown", src},
		{"run", "--quiet", "--bytecode", "--strict-bytecode", "--entry", "unknown", src},
	} {
		out, stderr, code := runWithStdin(t, bin, "", args...)
		all := out + stderr
		if code == 0 || !strings.Contains(all, "TC_MATCH_001") || !strings.Contains(all, "Bogus") {
			t.Errorf("%v: exit=%d, want non-zero with TC_MATCH_001 naming Bogus\noutput: %s", args, code, all)
		}
	}
}

// A constructor of an ADT the scrutinee cannot have, not in scope (only
// std/list imported): previously a silently dead arm on every route.
func TestCtorPatternOutOfScopeForeign1478(t *testing.T) {
	bin := buildAilang(t)
	src := filepath.Join("cmd", "ailang", "testdata", "ctoralias", "transitive_foreign.ail")
	out, stderr, code := runWithStdin(t, bin, "", "check", src)
	all := out + stderr
	if code == 0 || !strings.Contains(all, "'Err'") {
		t.Fatalf("exit=%d, want a compile error naming 'Err'\noutput: %s", code, all)
	}
}
