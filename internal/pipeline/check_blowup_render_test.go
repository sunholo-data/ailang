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
		"rigid signature":      "pure func f(x: P0[int]) -> P0[b] = x",
		"application arg":      "pure func g(y: int) -> int = y\npure func f(x: P0[int]) -> int = g(x)",
		"if branches":          "pure func f(x: P0[int]) -> int = if true then x else 1",
		"field access":         "pure func f(x: P0[int]) -> int = x.foo",
		"equality on fn":       "pure func f(x: P0[int -> int]) -> bool = x == x",
		"show of fn":           "pure func f(x: P0[int -> int]) -> string = show(x)",
		"match literal":        "pure func f(x: P0[int]) -> int = match x { 1 => 1, _ => 2 }",
		"record mismatch":      "pure func f(x: P0[int]) -> {w: int} = x",
		"arithmetic":           "pure func f(x: P0[int]) -> int = x + 1",
		"cons":                 "pure func f(x: P0[int]) -> [int] = x :: [1]",
		"record update":        "pure func f(x: P0[int]) -> int { let y = {x | a: 1}; 0 }",
		"ordering on fn":       "pure func f(x: P0[int -> int]) -> bool = x < x",
		"string concat":        "pure func f(x: P0[int]) -> string = \"a\" ++ x",
		"interpolation of fn":  "pure func f(x: P0[int -> int]) -> string = \"${x}\"",
		"tuple pattern":        "pure func f(x: P0[int]) -> int = match x { (a, b, c) => 1 }",
		"return mismatch":      "pure func f(x: P0[int]) -> int = x",
		"non-exhaustive match": "pure func f(x: P0[int]) -> bool = match x { Some(_) => true }",
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

// Shapes no sharing can make linear end in a diagnostic within the budget
// (#2363 review, round 2): twin chains through a function type (the
// equivalence check does not compare functions) stop at the size-relative
// unification budget, TC_TYPE_WORK_001; an alias whose two uses build
// different arguments at every level stops the IFC bound walk at the same
// budget.
func TestCheckBlowup_ExponentialArgumentsHitTheBudget(t *testing.T) {
	chain := func(x, body, base, params string, n int) string {
		var b strings.Builder
		fmt.Fprintf(&b, "type %s0%s = %s\n", x, params, base)
		for i := 1; i <= n; i++ {
			fmt.Fprintf(&b, "type %s%d%s = %s\n", x, i, params, strings.ReplaceAll(body, "X", fmt.Sprintf("%s%d", x, i-1)))
		}
		return b.String()
	}
	// The work runs to the budget, so it allocates (and frees) about 700 bytes
	// per unification step: bounded, but well above the other cases.
	const workAlloc = 4 << 30
	main := "export func main() -> () ! {IO} = println(\"x\")\n"
	ufun := "module m\n" + chain("P", "(X[a]) -> X[a]", "a", "[a]", 24) + chain("Q", "(X[a]) -> X[a]", "a", "[a]", 24) +
		"pure func g(u: int) -> P24[int] = unreachable(\"x\")\npure func f(u: int) -> Q24[int] = g(u)\n" + main
	err := checkBounded(t, ufun, blowupLimit, workAlloc)
	if err == nil || !strings.Contains(err.Error(), "TC_TYPE_WORK_001") || !strings.Contains(err.Error(), "too much unification work for its size") {
		t.Fatalf("twin function chains: want TC_TYPE_WORK_001, got %v", err)
	}
	ucap := "module m\n" + chain("P", "(X[(a, b), b], X[a, (b, a)])", "(a, b)", "[a, b]", 20) + "pure func f(x: P20[int, string]) -> int = 0\n" + main
	err = checkBounded(t, ucap, blowupLimit, workAlloc)
	if err == nil || !strings.Contains(err.Error(), "parameter type work") {
		t.Fatalf("distinct arguments at every level: want the IFC work limit, got %v", err)
	}
}
