package pipeline

import (
	"strings"
	"testing"
)

// TestConcatStringErrorMessage verifies that using `++` on strings produces
// a helpful error pointing to interpolation and the concat()/join() stdlib.
//
// This is the user-facing contract of M-CONCAT-DISAMBIG Phase 2 (v0.13.0):
// `++` is for lists only; strings must use `"${...}"`, `concat([parts])`, or
// `join(sep, parts)`.
func TestConcatStringErrorMessage(t *testing.T) {
	code := `
module test/concat_error

export func broken() -> string {
  "a" ++ "b"
}
`

	err := typeCheckCode(t, code)
	if err == nil {
		t.Fatal("string ++ string should fail after M-CONCAT-DISAMBIG Phase 2")
	}

	msg := err.Error()

	wantFragments := []string{
		"++",
		"list",
		"interpolation",
	}
	for _, frag := range wantFragments {
		if !strings.Contains(msg, frag) {
			t.Errorf("error message must mention %q for discoverability, got: %s", frag, msg)
		}
	}
}

// TestConcatListPolymorphic verifies that `++` still works when both operands
// are polymorphic `[a]` (recursive/HOF contexts) — the bug that M-CONCAT-DISAMBIG
// Phase 2 must NOT regress.
func TestConcatListPolymorphic(t *testing.T) {
	code := `
module test/concat_poly

export func flatten[a](xss: [[a]]) -> [a] {
  match xss {
    [] => [],
    xs :: rest => xs ++ flatten(rest)
  }
}
`

	err := typeCheckCode(t, code)
	if err != nil {
		t.Errorf("polymorphic list ++ must type-check, got: %v", err)
	}
}
