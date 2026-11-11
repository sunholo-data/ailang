package main

import (
	"encoding/json"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// M-V1-MATCH-EXHAUSTIVENESS M2: a match that misses a case is a compile error,
// TC_MATCH_002 with the missing pattern and the match's file:line:col, on
// every path that compiles a program — `check` in its three output formats,
// `run` on the evaluator, `run --bytecode`, `run --strict-bytecode` and
// `compile --emit-go`. It is the type checker's verdict, so no path runs the
// program first: nothing prints RT002 (before the check, each fixture below
// reached RT002 at run time — or, on the VM, a Unit value).
//
// The run-time RT002 path itself is covered in process, with the check
// switched off (internal/vm TestMatchFailBothEngines).
func TestMatchNonExhaustiveRejectedOnEveryPath(t *testing.T) {
	t.Setenv("AILANG_CACHE_DIR", t.TempDir())
	bin := buildAilang(t)
	src := filepath.Join("cmd", "ailang", "testdata", "matchfail", "match_fail.ail")
	file := filepath.ToSlash(src)
	type site struct {
		line, col int
		missing   string
	}
	sites := []site{
		{12, 35, "Blue"},        // ADT
		{21, 36, "Has(_)"},      // a guarded arm does not count
		{28, 36, "[]"},          // list
		{36, 11, "Green"},       // value position (let)
		{46, 15, "Blue, Green"}, // inside a lambda
	}

	// check --json: every error of the module, each located and coded.
	stdout, stderr, code := runWithStdin(t, bin, "", "check", "--json", src)
	if code != 1 {
		t.Fatalf("check --json exit %d, want 1\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	var out checkJSONOutput
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("bad JSON %q: %v", stdout, err)
	}
	if out.Passed || out.ErrorCount != len(sites) || len(out.Errors) != len(sites) {
		t.Fatalf("want %d errors, got %+v", len(sites), out)
	}
	for i, s := range sites {
		e := out.Errors[i]
		if e.Code != "TC_MATCH_002" || filepath.ToSlash(e.File) != file || e.Line != s.line || e.Column != s.col ||
			!strings.HasSuffix(e.Message, "no arm matches: "+s.missing) || !strings.HasPrefix(e.Suggestion, "add the missing arm") {
			t.Errorf("error %d = %+v, want TC_MATCH_002 at %s:%d:%d naming %s", i, e, file, s.line, s.col, s.missing)
		}
	}

	// --format agent: one line per error.
	stdout, _, code = runWithStdin(t, bin, "", "check", "--format", "agent", src)
	lines := strings.Split(strings.TrimSpace(strings.ReplaceAll(stdout, "\r\n", "\n")), "\n")
	if code != 1 || len(lines) != len(sites) {
		t.Fatalf("agent exit %d, %d lines, want 1 and %d:\n%s", code, len(lines), len(sites), stdout)
	}
	for i, s := range sites {
		want := file + ":" + strconv.Itoa(s.line) + ":" + strconv.Itoa(s.col) + " TC_MATCH_002: "
		if l := filepath.ToSlash(lines[i]); !strings.HasPrefix(l, want) || !strings.Contains(l, "no arm matches: "+s.missing+" → ") {
			t.Errorf("agent line %d = %q, want prefix %q naming %s", i, l, want, s.missing)
		}
	}

	// Human output, and every engine and the Go emitter: the same verdict
	// before anything runs.
	paths := map[string][]string{
		"check":           {"check"},
		"run":             {"run", "--quiet", "--entry", "adt"},
		"run --bytecode":  {"run", "--quiet", "--bytecode", "--entry", "adt"},
		"run --strict":    {"run", "--quiet", "--strict-bytecode", "--entry", "adt"},
		"compile emit-go": {"compile", "--emit-go", "--out", t.TempDir()},
	}
	for name, args := range paths {
		stdout, stderr, code := runWithStdin(t, bin, "", append(args, src)...)
		all := filepath.ToSlash(stdout + stderr)
		if code == 0 {
			t.Errorf("%s: exit 0, want a compile error\n%s", name, all)
		}
		want := file + ":12:35: TC_MATCH_002: non-exhaustive match on Color — no arm matches: Blue"
		if !strings.Contains(all, want) {
			t.Errorf("%s: output lacks %q:\n%s", name, want, all)
		}
		if strings.Contains(all, "RT002") || strings.Contains(all, "panic:") {
			t.Errorf("%s: the program ran (or panicked) instead of failing to compile:\n%s", name, all)
		}
	}
}
