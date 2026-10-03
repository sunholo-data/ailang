package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// M-ELABORATOR-LEXICAL-SCOPE (#1467): a local binder (lambda / function
// parameter, let, letrec, block statement-let, match binder) shadows an
// imported function, a builtin, or a constructor of the same name inside its
// scope, on both the evaluator and the strict VM. Before the fix the
// elaborator consulted the constructor table and globalEnv first, so the whole
// module failed to type-check ("cannot unify type constructor int with
// *types.TFunc2") or, in miscompile.ail, silently ran the import.

func TestLexicalScopeShadowing1467(t *testing.T) {
	bin := buildAilang(t)
	src := filepath.Join("cmd", "ailang", "testdata", "lexscope", "d.ail")
	want := map[string]string{
		"lambdaRecord":          "4",
		"lambdaArith":           "8",
		"letBinding":            "5",
		"funcParam":             "42",
		"matchBinder":           "9",
		"matchGuard":            "4",
		"tupleBinder":           "3",
		"listBinder":            "5",
		"builtinLet":            "3",
		"builtinParam":          "2",
		"letrecBinding":         "6",
		"blockLet":              "6",
		"blockBeforeLet":        "12",
		"ctorParam":             "4",
		"ctorCallParam":         "4",
		"moduleAliasLocal":      "42",
		"importStillVisible":    "2",
		"ctorStillVisible":      "2",
		"qualifiedStillVisible": "3",
	}
	// The strict VM cannot compile these shapes for ANY binder name (a bare
	// variable-pattern arm: "unknown ADT \"\" in switch", see
	// m-vm-var-pattern-default-arm; an expression-form letrec: "call to
	// unbound name"), so they run on the evaluator only.
	evaluatorOnly := map[string]bool{"matchBinder": true, "matchGuard": true, "letrecBinding": true}
	for entry, w := range want {
		modes := [][]string{{"run", "--quiet", "--entry", entry, src}}
		if !evaluatorOnly[entry] {
			modes = append(modes, []string{"run", "--quiet", "--bytecode", "--strict-bytecode", "--entry", entry, src})
		}
		for _, mode := range modes {
			out, stderr, code := runWithStdin(t, bin, "", mode...)
			if code != 0 || strings.TrimSpace(out) != w {
				t.Errorf("%s %v: exit=%d out=%q want %q\nstderr: %s", entry, mode[:len(mode)-1], code, strings.TrimSpace(out), w, stderr)
			}
		}
	}
}

// Row 8 of the design matrix: `mk(\tick. tick)(5)` with mk expecting a
// function-returning lambda used to type-check by resolving the body's `tick`
// to the import, and returned 6. The parameter is an int, so it is a type
// error.
func TestLexicalScopeNoSilentImportCapture1467(t *testing.T) {
	bin := buildAilang(t)
	src := filepath.Join("cmd", "ailang", "testdata", "lexscope", "miscompile.ail")
	out, stderr, code := runWithStdin(t, bin, "", "run", "--quiet", "--entry", "main", src)
	if code == 0 {
		t.Fatalf("want a type error, got exit 0 with output %q (stderr %s)", strings.TrimSpace(out), stderr)
	}
}
