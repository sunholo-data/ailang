package pipeline

import (
	"os"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/elaborate"
	ailerrors "github.com/sunholo-data/ailang/internal/errors"
)

// checkBodySource compiles one module from source in a fresh directory,
// cold (no compile cache), as `check` does.
func checkBodySource(t *testing.T, name, src string) error {
	t.Helper()
	t.Chdir(t.TempDir())
	if err := os.WriteFile(name+".ail", []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Run(Config{Mode: ModeCheck, NoCache: true}, Source{Filename: name + ".ail"})
	return err
}

// #2248: every contract clause type-checks as a bool predicate; an inference
// error is TC_CONTRACT_001 (it used to be printed only under --debug-compile).
func TestContractsAreTypeChecked(t *testing.T) {
	bad := map[string]struct{ src, want string }{
		// The issue's repro: a quantifier made inference fail, hiding i + "x".
		"forall_body_type_error": {`module m
export func total(n: int) -> int ! {}
requires { forall i: 0..2 => i + "x" }
{ 1 }
`, "cannot unify"},
		"plain_type_error": {`module m
export func total(n: int) -> int ! {}
requires { n + "x" }
{ 1 }
`, "cannot unify"},
		"not_a_bool": {`module m
export func total(n: int) -> int ! {}
requires { n + 1 }
{ 1 }
`, "int vs bool"},
		"undefined_variable": {`module m
export func total(n: int) -> int ! {}
requires { nope > 0 }
{ 1 }
`, "undefined variable: nope"},
		"result_in_requires": {`module m
export func total(n: int) -> int ! {}
requires { result > 0 }
{ 1 }
`, "undefined variable: result"},
		"forall_bound_not_int": {`module m
export func total(n: int) -> int ! {}
requires { forall i: "a".."b" => i >= 0 }
{ 1 }
`, "forall lower bound"},
		"forall_body_not_bool": {`module m
export func total(n: int) -> int ! {}
requires { forall i: 0..n => i + 1 }
{ 1 }
`, "forall body"},
		"ensures_result_type": {`module m
export func total(n: int) -> int ! {}
ensures { result == "one" }
{ 1 }
`, "cannot unify"},
		"no_eq_instance": {`module m
type C = R | G
export func pick(n: int) -> C ! {}
ensures { result == R }
{ R }
`, "No instance for Eq"},
	}
	for name, tc := range bad {
		t.Run(name, func(t *testing.T) {
			err := checkBodySource(t, "m", tc.src)
			wantReport(t, err, CodeContract, tc.want)
		})
	}

	good := map[string]string{
		"forall_well_typed": `module m
export func total(n: int) -> int ! {}
requires { n >= 0, forall i: 0..n => i >= 0 && i < n }
ensures { result >= 0 }
{ n }
`,
		"forall_nested": `module m
export func total(n: int) -> int ! {}
requires { forall i: 0..n => forall j: i..n => i <= j }
{ n }
`,
		"record_and_result": `module m
export func net(inv: {subtotal: int, tax: int}) -> int ! {}
requires { inv.subtotal >= 0, inv.tax >= 0 }
ensures { result == inv.subtotal + inv.tax }
{ inv.subtotal + inv.tax }
`,
		"derived_eq_adt": `module m
type C = R | G deriving (Eq)
export func pick(n: int) -> C ! {}
ensures { result == R || result == G }
{ R }
`,
	}
	for name, src := range good {
		t.Run(name, func(t *testing.T) {
			if err := checkBodySource(t, "m", src); err != nil {
				t.Fatalf("a well-typed contract must check: %v", err)
			}
		})
	}
}

// #2232: named test blocks, property blocks, inline properties and inline
// rows type-check against the module (TC_TEST_001), as `ailang test` would
// compile them; test and property bodies are checked pure.
func TestTestBodiesAreTypeChecked(t *testing.T) {
	bad := map[string]struct{ src, want string }{
		// The issue's repro (with a local function in place of std/math).
		"named_test_type_error": {`module m
export pure func k(x: int) -> int = x
test "good" { assert k(3) == 3 }
test "illtyped" { assert k("x") == 3 }
`, `test "illtyped"`},
		"named_test_undefined": {`module m
export pure func k(x: int) -> int = x
test "t" { assert length([1]) == 1 }
`, "undefined variable: length"},
		"bare_nonbool_nonfinal": {`module m
export pure func k(x: int) -> int = x
test "t" { k(1); true }
`, "if condition"},
		"effectful_test": {`module m
import std/io (println)
export func say() -> () ! {IO} = println("hi")
test "t" { let u = say(); true }
`, "checked pure"},
		"property_block": {`module m
export pure func f(x: int) -> int = x
property "p" { forall(x: int) => f(x) + "a" == 1 }
`, `property "p"`},
		"inline_property": {`module m
export pure func f(x: int) -> int
  properties [ forall(y: int) => f(y) == "s" ]
{ x }
`, `property "f property 1"`},
		"inline_row_expected": {`module m
export pure func f(x: int) -> int tests [(1, "x")] { x }
`, `test row "f row 1"`},
		// The runner fails a non-bool final value ("expected bool result");
		// check says so first (review round 2).
		"single_nonbool_final": {`module m
export pure func k(x: int) -> int = x
test "t" { k(1) }
`, "return type annotation"},
		"property_not_bool": {`module m
export pure func k(x: int) -> int = x
property "p" { forall(x: int) => k(x) }
`, `property "p"`},
		"inline_row_arity": {`module m
export pure func f(x: int) -> int tests [((1, 2), 3)] { x }
`, `test row "f row 1"`},
	}
	for name, tc := range bad {
		t.Run(name, func(t *testing.T) {
			err := checkBodySource(t, "m", tc.src)
			wantReport(t, err, elaborate.CodeTestBody, tc.want)
		})
	}

	good := map[string]string{
		"asserts_lets_private_helper": `module m
pure func helper(x: int) -> int = x * 2
export pure func k(x: int) -> int = helper(x)
test "t" {
  let a = k(2);
  assert a == 4;
  a > 0
}
`,
		// The legacy path's single final expression is the check: a bool.
		"single_bool_final": `module m
export pure func k(x: int) -> int = x
test "t" { k(1) == 1 }
`,
		"property_with_binders": `module m
export pure func f(x: int, y: int) -> int = x + y
property "comm" { forall(x: int, y: int) => f(x, y) == f(y, x) }
`,
		"inline_rows_multi_arg": `module m
export pure func add(a: int, b: int) -> int tests [((1, 2), 3), ((0, 0), 0)] { a + b }
`,
		"inline_row_polymorphic": `module m
export pure func same[a](x: a, y: a) -> bool tests [((5, 3), true)] { true }
`,
		"debug_is_ghost": `module m
import std/debug (log)
export pure func k(x: int) -> int = x
test "t" { let u = log("x"); k(1) == 1 }
`,
	}
	for name, src := range good {
		t.Run(name, func(t *testing.T) {
			if err := checkBodySource(t, "m", src); err != nil {
				t.Fatalf("a well-typed test body must check: %v", err)
			}
		})
	}
}

func wantReport(t *testing.T, err error, code, substr string) {
	t.Helper()
	if err == nil {
		t.Fatalf("want %s (%q), got no error", code, substr)
	}
	rep, ok := ailerrors.AsReport(err)
	if !ok || rep.Code != code {
		t.Fatalf("want a %s report, got %v", code, err)
	}
	if !strings.Contains(rep.Message, substr) {
		t.Fatalf("%s message lacks %q: %s", code, substr, rep.Message)
	}
	if rep.Span == nil || rep.Span.Start.Line == 0 {
		t.Fatalf("%s carries no source position: %v", code, err)
	}
}

// Once a body is type-checked, its matches fall under TC_MATCH_002 (#2246):
// a match inside a quantifier was out of reach while `forall` made the
// contract's inference fail, and a test body's match was never inferred.
func TestNewlyCheckedBodiesFallUnderMatchExhaustiveness(t *testing.T) {
	cases := map[string]string{
		"match_in_forall": `module m
export func total(n: int) -> int ! {}
requires { forall i: 0..n => match i { 0 => true } }
{ 1 }
`,
		"match_in_test": `module m
type C = R | G
export pure func k(c: C) -> int = 1
test "t" { match R { R => true } }
`,
		"match_in_property": `module m
export pure func k(x: int) -> int = x
property "p" { forall(x: int) => match x { 0 => true } }
`,
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			err := checkBodySource(t, "m", src)
			if err == nil || !strings.Contains(err.Error(), "TC_MATCH_002") {
				t.Fatalf("want TC_MATCH_002, got %v", err)
			}
		})
	}
}

// The first contract error reported does not depend on map order: the
// clauses are checked in function-name order (review round 2, M8).
func TestContractErrorsAreDeterministic(t *testing.T) {
	src := `module m
export func zeta(n: int) -> int ! {}
requires { n + "z" }
{ 1 }
export func alpha(n: int) -> int ! {}
requires { n + "a" }
{ 1 }
export func mid(n: int) -> int ! {}
requires { n + "m" }
{ 1 }
`
	var first string
	for i := 0; i < 10; i++ {
		err := checkBodySource(t, "m", src)
		if err == nil {
			t.Fatal("want TC_CONTRACT_001")
		}
		if i == 0 {
			first = err.Error()
			if !strings.Contains(first, "clause of alpha") {
				t.Fatalf("want the alphabetically first function, got %v", err)
			}
		} else if err.Error() != first {
			t.Fatalf("run %d reported %q, run 0 %q", i, err.Error(), first)
		}
	}
}

// forall is a contract-only form: TC_FORALL_001 anywhere else (it has no
// evaluator, VM, lowering or validator case), never a panic.
func TestForallOutsideAContract(t *testing.T) {
	cases := map[string]string{
		"function_body": `module m
export pure func allPos(n: int) -> bool = forall i: 0..n => i >= 0
`,
		"let": `module m
export pure func f(n: int) -> bool {
  let ok = forall i: 0..n => i >= 0;
  ok
}
`,
		"test_body": `module m
export pure func k(x: int) -> int = x
test "q" { forall i: 0..3 => k(i) >= 0 }
`,
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			err := checkBodySource(t, "m", src)
			if err == nil || !strings.Contains(err.Error(), "TC_FORALL_001") {
				t.Fatalf("want TC_FORALL_001, got %v", err)
			}
			if rep, ok := ailerrors.AsReport(err); !ok || rep.Span == nil || rep.Span.Start.Line == 0 {
				t.Fatalf("want a located report, got %v", err)
			}
		})
	}
}
