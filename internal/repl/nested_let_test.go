package repl

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// replValue evaluates input on r and returns the printed value without its
// ` :: Type` suffix, or the whole output when it is not a value.
func replValue(r *REPL, input string) string {
	var out bytes.Buffer
	r.ProcessExpression(input, &out)
	s := strings.TrimSpace(stripANSI(out.String()))
	if i := strings.LastIndex(s, " :: "); i >= 0 && !strings.Contains(s, "\n") {
		s = s[:i]
	}
	return strings.Trim(s, `"`)
}

func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// newPreludeREPL is a REPL set up as the CLI and the wasm playground set it
// up: builtins and the prelude imported.
func newPreludeREPL() *REPL {
	r := New()
	r.HandleCommand(":import std/prelude", io.Discard)
	return r
}

// nestedLetMatrix: a block or a parenthesised let-in in every position an
// expression can take, next to a read of the name its let shadows. Since #2388
// a let the program wrote stays nested in the value it belongs to; the REPL's
// ANF verifier still described the old floated shape and rejected all of them
// with "ANF verification error" (#2407). Expected values are `ailang run`'s.
var nestedLetMatrix = []struct{ in, want string }{
	{`let r = 1 in let add = \a. \b. a * 7 + b in add((let r = 2 in r * 10))(r)`, "141"},
	{`let r = 1 in let add = \a. \b. a * 7 + b in add(r)((let r = 2 in r * 10))`, "27"},
	{`let r = 1 in let add = \a. \b. a * 7 + b in add({ let r = 2; r * 10 })(r)`, "141"},
	{`let r = 1 in { let r = 2; r * 10 } * 7 + r`, "141"},
	{`let r = 1 in r * 7 + { let r = 2; r * 10 }`, "27"},
	{`let r = 1 in (let r = 2 in r * 10) * 7 + r`, "141"},
	{`let r = 1 in [{ let r = 2; r }, r]`, "[2, 1]"},
	{`let r = 1 in ({ let r = 2; r }, r)`, "(2, 1)"},
	{`let r = 1 in {a: { let r = 2; r }, b: r}.a * 7 + {a: { let r = 2; r }, b: r}.b`, "15"},
	{`let r = 1 in if { let r = 0; r > 0 } then r else r + 10`, "11"},
	{`let r = 1 in if (let r = 0 in r > 0) then r else r + 10`, "11"},
	{`let r = 1 in match { let r = 5; r } { 5 => r, _ => 0 }`, "1"},
	{`let r = 1 in let s = { let r = 2; r * 10 } in r + s`, "21"},
	{`let r = 1 in let s = (let r = 2 in r * 10) in r + s`, "21"},
	{`let r = 1 in let f = \x. { let r = x; r * 2 } + r in f(5)`, "11"},
	{`let r = 1 in let ap = \g. \x. g(x) in ap(\z. z + r)({ let r = 3; r })`, "4"},
	{`let r = 1 in { let r = 2; { let r = 3; r } * 10 + r } * 100 + r`, "3201"},
	{`let r = 1 in let s = { let r = 2; r * 10 } in [r, s]`, "[1, 20]"},
	{`let r = 1 in let f = (let k = 1 in \x. x) in f(r) + f(2)`, "3"},
	{`let r = 1 in let v = { let a = 1; a + 1 } in v + r`, "3"},
	{`let k = 10 in let f = \x. k in let g = \k. f(3) in g(99)`, "10"},
	{`let k = 10 in let f = \x. k in match (99, 0) { (k, z) => f(3) }`, "10"},
	{`let a = \x. x in let b = a in let a = \x. 7 in b(3)`, "3"},
	{`{ let a = 1; a } + 2`, "3"},
	{`let x = if true then 1 else 2 in x`, "1"},
	{`let y = match 1 { 1 => 2, _ => 3 } in y`, "2"},
	{`let t = (1, 2) in t`, "(1, 2)"},
	{`(1, 2)`, "(1, 2)"},
}

func TestREPLNestedLetPositions(t *testing.T) {
	r := newPreludeREPL()
	for _, c := range nestedLetMatrix {
		if got := replValue(r, c.in); got != c.want {
			t.Errorf("%s\n  got  %s\n  want %s", c.in, got, c.want)
		}
	}
}

// The same matrix through the registry the wasm playground uses
// (cmd/wasm: NewWithVersion, SetRegistry, the prelude import, then
// ProcessExpression per ailangEval call).
func TestREPLNestedLetPositions_WasmSetup(t *testing.T) {
	r := NewWithVersion("test", "now")
	reg := NewModuleRegistry()
	r.SetRegistry(reg)
	reg.SetEffContext(r.GetEffContext())
	r.HandleCommand(":import std/prelude", io.Discard)
	for _, c := range nestedLetMatrix {
		if got := replValue(r, c.in); got != c.want {
			t.Errorf("%s\n  got  %s\n  want %s", c.in, got, c.want)
		}
	}
}

// A top-level REPL `let` binds a new environment: a closure keeps the binding
// it was made with, as lexical scope says, and does not see a later `let` of
// the same name (it printed 99). An alias already held the value.
func TestREPLSessionRebindingDoesNotReachEarlierClosures(t *testing.T) {
	r := newPreludeREPL()
	for _, in := range []string{`let k = 10`, `let f = \x. k`, `let k = 99`} {
		replValue(r, in)
	}
	if got := replValue(r, `f(3)`); got != "10" {
		t.Errorf("f(3) = %s, want 10", got)
	}
	if got := replValue(r, `k`); got != "99" {
		t.Errorf("k = %s, want 99 (the later binding, for later inputs)", got)
	}
	for _, in := range []string{`let g = \x. x`, `let h = g`, `let g = \x. 7`} {
		replValue(r, in)
	}
	if got := replValue(r, `[h(3), g(3)]`); got != "[3, 7]" {
		t.Errorf("[h(3), g(3)] = %s, want [3, 7]", got)
	}
}

// :reset drops the session's bindings and keeps the builtins and prelude;
// bindings made after it are visible (they were lost: only the REPL's copy of
// the environment was replaced, not the evaluator's).
func TestREPLResetKeepsBuiltinsAndNewBindings(t *testing.T) {
	r := newPreludeREPL()
	replValue(r, `let a = 5`)
	r.HandleCommand(":reset", io.Discard)
	if got := replValue(r, `let b = 5 in b + 1`); got != "6" {
		t.Errorf("after :reset, let-in: %s", got)
	}
	replValue(r, `let c = 7`)
	if got := replValue(r, `c + 1`); got != "8" {
		t.Errorf("after :reset, a new top-level let: %s", got)
	}
	if got := replValue(r, `a`); !strings.Contains(got, "undefined variable: a") {
		t.Errorf("after :reset, a binding from before it: %s", got)
	}
	if got := replValue(r, `show(1)`); got != "1" {
		t.Errorf("after :reset, a builtin: %s", got)
	}
	// A prelude function, in both the type and the value environment (it was
	// undefined after a reset, first to the checker, then at run time).
	if got := replValue(r, `println("")`); strings.Contains(got, "undefined") || strings.Contains(got, "error") {
		t.Errorf("after :reset, a prelude function: %s", got)
	}
	// The REPL and its evaluator use one environment, so what a `let` binds
	// is what the next input reads.
	r.HandleCommand(":reset", io.Discard)
	if r.evaluator.Env() != r.env {
		t.Errorf("after :reset the evaluator and the REPL use different environments")
	}
}

// Modules loaded through the registry (the wasm playground's ailangLoadModule,
// the REPL's imports) make local names unique before monomorphisation, as the
// native pipelines do (#2401): a copied lambda body keeps its own k.
func TestRegistryModuleMonomorphisationDoesNotCapture(t *testing.T) {
	reg := NewModuleRegistry()
	code := `module test/capture
export func viaParam(k: int) -> int {
  let f = \x. k;
  let g = \k. f(3);
  g(99)
}
export func viaPattern(k: int) -> int {
  let f = \x. k;
  match (99, 0) { (k, z) => f(0) }
}
export func alias(n: int) -> int {
  let a = \x. x;
  let b = a;
  let a = \x. 7;
  b(n)
}
export func nested(r: int) -> int {
  let s = { let r = 2; r * 10 };
  r * 7 + s + (let r = 3 in r)
}
`
	if _, err := reg.LoadModule("test/capture", code); err != nil {
		t.Fatalf("load: %v", err)
	}
	for fn, want := range map[string]int{"viaParam": 5, "viaPattern": 5, "alias": 5, "nested": 58} {
		v, err := reg.InvokeExport("test/capture", fn, []eval.Value{&eval.IntValue{Value: 5}})
		if err != nil {
			t.Fatalf("%s: %v", fn, err)
		}
		if got, ok := v.(*eval.IntValue); !ok || got.Value != want {
			t.Errorf("%s(5) = %v, want %d", fn, v, want)
		}
	}
}

// The type printed after a value is the expression's type. A structured type
// fell through to "any defaulted constraint's type", so a list, tuple or
// record of numbers printed `:: Int` (and a tuple, until #2407, failed).
func TestREPLDisplaysStructuredTypes(t *testing.T) {
	r := newPreludeREPL()
	for in, want := range map[string]string{
		`[1, 2]`:         "[1, 2] :: [Int]",
		`[[1.5]]`:        "[[1.5]] :: [[Float]]",
		`{a: 1, b: "x"}`: `{a: 1, b: "x"} :: {a: Int, b: String}`,
		`(1, "a")`:       `(1, "a") :: (int, string)`,
		`\x. x + 1`:      "<function> :: int -> int",
		`1 + 2`:          "3 :: Int",
		`1.5 * 2.0`:      "3.0 :: Float",
		`let r = 1 in let s = { let r = 2; r * 10 } in [r, s]`: "[1, 20] :: [Int]",
		// A type with variables left, next to a defaulted number: shown as
		// inferred, not as the number's type.
		`let n = 1 in []`: "[] :: [α2]",
	} {
		var out bytes.Buffer
		r.ProcessExpression(in, &out)
		if got := strings.TrimSpace(stripANSI(out.String())); got != want {
			t.Errorf("%s\n  got  %s\n  want %s", in, got, want)
		}
	}
}
