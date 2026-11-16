package pipeline

import (
	"fmt"
	"strings"
	"testing"
)

// #2363 review, round 2: every kind of diagnostic that renders a type must
// render it bounded. A doubling alias chain of 28 (P0[int] expands to a tuple
// tree of 2^28 leaves under Option) is met by each kind below; before the
// fix several wrote the expansion out in full (field access 1.1 GB at n=24,
// killed at 3 GB at n=28). Each must finish under the time and allocation
// budgets with a message of bounded length.
func TestCheckBlowup_EveryDiagnosticRendersTypesBounded(t *testing.T) {
	kinds := map[string]string{
		"rigid signature":     "pure func f(x: P0[int]) -> P0[b] = x",
		"application arg":     "pure func g(y: int) -> int = y\npure func f(x: P0[int]) -> int = g(x)",
		"if branches":         "pure func f(x: P0[int]) -> int = if true then x else 1",
		"field access":        "pure func f(x: P0[int]) -> int = x.foo",
		"equality on fn":      "pure func f(x: P0[int -> int]) -> bool = x == x",
		"show of fn":          "pure func f(x: P0[int -> int]) -> string = show(x)",
		"match literal":       "pure func f(x: P0[int]) -> int = match x { 1 => 1, _ => 2 }",
		"record mismatch":     "pure func f(x: P0[int]) -> {w: int} = x",
		"arithmetic":          "pure func f(x: P0[int]) -> int = x + 1",
		"cons":                "pure func f(x: P0[int]) -> [int] = x :: [1]",
		"record update":       "pure func f(x: P0[int]) -> int { let y = {x | a: 1}; 0 }",
		"ordering on fn":      "pure func f(x: P0[int -> int]) -> bool = x < x",
		"string concat":       "pure func f(x: P0[int]) -> string = \"a\" ++ x",
		"interpolation of fn": "pure func f(x: P0[int -> int]) -> string = \"${x}\"",
		"tuple pattern":       "pure func f(x: P0[int]) -> int = match x { (a, b, c) => 1 }",
		"return mismatch":     "pure func f(x: P0[int]) -> int = x",
	}
	for name, body := range kinds {
		t.Run(name, func(t *testing.T) {
			src := blowupAliasChain(28, body)
			err := checkBounded(t, src, blowupLimit, blowupAlloc)
			if err == nil {
				return // some kinds are accepted (the type is never mismatched); bounded either way
			}
			if len(err.Error()) > 64<<10 {
				t.Fatalf("diagnostic is %d bytes: a type was written out in full", len(err.Error()))
			}
		})
	}
}

// Ordinary diagnostics are byte-identical: a small type renders exactly as
// before, with no "shown in part".
func TestCheckBlowup_SmallTypeDiagnosticsUnchanged(t *testing.T) {
	src := blowupAliasChain(3, "pure func f(x: P0[int]) -> int = x")
	err := checkBounded(t, src, blowupLimit, blowupAlloc)
	want := fmt.Sprintf("Option[%s]", "(((int, int), (int, int)), ((int, int), (int, int)))")
	if err == nil || !strings.Contains(err.Error(), want) || strings.Contains(err.Error(), "shown in part") {
		t.Fatalf("want the full type %s in the message, got %v", want, err)
	}
}
