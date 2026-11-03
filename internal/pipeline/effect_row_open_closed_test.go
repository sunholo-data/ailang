package pipeline

import (
	"strings"
	"testing"
)

// #1958: M2's open-vs-closed check (re-landed with F-A1/F-A2/F-B) must not
// reject valid callbacks. v0.56.0's M2 rejected the must-accept cases below
// because inference closed multi-effect lambda rows to {} (the union drop) and
// a pure lambda's closed row could not fit a wider slot (width).
func TestEffectRowOpenClosedAnnotatedCallbacks(t *testing.T) {
	for _, tt := range []struct {
		name, src string
		reject    bool
	}{
		{"direct call", `pure func via(g: (int) -> int ! {IO}, x: int) -> int = g(x)`, true},
		{"named effectful callback", `func loud(x: int) -> int ! {IO} { println("leak"); x }
pure func via(xs: [int]) -> [int] = map(loud, xs)`, true},
		{"option map", `pure func via(g: (int) -> int ! {IO}, x: Option[int]) -> Option[int] = O.map(g, x)`, true},
		{"pure mapE rejected", `pure func via(xs: [int]) -> [int] = mapE(\x. { println(show(x)); x }, xs)`, true},
		{"any IO lambda in IO caller (D-R2: pure slot)", `func via(xs: [int]) -> bool ! {IO} = any(\x. { println(show(x)); true }, xs)`, true},
		{"pure map", `pure func via(xs: [int]) -> [int] = map(\x. x * 2, xs)`, false},
		{"two-effect lambda into concrete slot", `func each(f: string -> () ! {IO}, xs: [string]) -> () ! {IO} = match xs { [] => (), x :: rest => { f(x); each(f, rest) } }
func via(xs: [string]) -> () ! {IO} = each(\x. { println(x); println(x) }, xs)`, false},
		{"effectful argument inside lambda", `func p() -> string ! {IO} = "p"
func ex(s: string, n: int) -> int ! {IO} = { println(s); n }
func withDir(f: string -> int ! {IO}) -> int ! {IO} = f("d")
func via() -> int ! {IO} = withDir(\d. ex(p(), 1))`, false},
		{"branches of effectful calls inside lambda", `func q(n: int) -> int ! {IO} = { println("q"); n }
func withDir(f: string -> int ! {IO}) -> int ! {IO} = f("d")
func via() -> int ! {IO} = withDir(\d. if d == "x" then q(1) else q(2))`, false},
		{"pure lambda into wider slot", `func withDir(f: string -> int ! {IO}) -> int ! {IO} = f("d")
func via() -> int ! {IO} = withDir(\d. 1)`, false},
		// A single concrete call leaves the body row as the call's unresolved tail;
		// it must still open on use (ailang-demos streaming/test_stream_basic.ail).
		{"one-call IO lambda into wider slot", `func w(f: (int) -> () ! {IO, FS}) -> () ! {IO, FS} = f(1)
func via() -> () ! {IO, FS} = w(\c. println("x"))`, false},
		{"one-call block lambda beside a record argument", `func w(l: string, cfg: {h: [int]}, f: (int) -> () ! {IO, FS}) -> () ! {IO, FS} = f(1)
func via() -> () ! {IO, FS} = w("a", {h: []}, \c. { println("x") })`, false},
		{"call then value into wider slot", `func w(f: (int) -> int ! {IO, FS}) -> int ! {IO, FS} = f(1)
func via() -> int ! {IO, FS} = w(\c. { println("x"); c })`, false},
		{"eta-expanded named function into wider slot", `func loud(s: string) -> () ! {IO} = println(s)
func runWide(f: (string) -> () ! {IO, FS}, s: string) -> () ! {IO, FS} = f(s)
func via() -> () ! {IO, FS} = runWide(\s. loud(s), "x")`, false},
		{"mapE lambda", `func via(xs: [int]) -> [int] ! {IO} = mapE(\x. { println(show(x)); x }, xs)`, false},
		{"mapE rowpoly", `func via[e](g: (int) -> int ! {e}, xs: [int]) -> [int] ! {e} = mapE(g, xs)`, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			src := "module row_guard\nimport std/list (map, filter, any, foldl, mapE)\nimport std/io (println)\nimport std/option as O\nimport std/option (Option)\n" + tt.src + "\n"
			err := checkModuleSource(t, "row_guard", src)
			if !tt.reject {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), "IO") {
				t.Fatalf("must reject hidden IO: %v", err)
			}
			if strings.Contains(err.Error(), "internal invariant") {
				t.Fatalf("must give an effect diagnostic: %v", err)
			}
		})
	}
}

// #1958 M2: an annotated IO callback must not fit a closed pure slot.
func TestEffectRowOpenClosedM2(t *testing.T) {
	for _, src := range []string{
		`pure func via(g: (int) -> int ! {IO}, xs: [int]) -> [int] = map(g, xs)`,
		`pure func via(g: (int) -> bool ! {IO}, xs: [int]) -> [int] = filter(g, xs)`,
		`pure func via(g: (int) -> bool ! {IO}, xs: [int]) -> bool = any(g, xs)`,
		`pure func via(g: (int, int) -> int ! {IO}, xs: [int]) -> int = foldl(g, 0, xs)`,
		`type Hook = { run: (int) -> int ! {IO} }
pure func via(h: Hook, xs: [int]) -> [int] = map(h.run, xs)`,
		`pure func via[e](g: (int) -> int ! {IO, e}, xs: [int]) -> [int] = map(g, xs)`,
	} {
		full := "module row_guard\nimport std/list (map, filter, any, foldl, mapE)\nimport std/io (println)\nimport std/option as O\nimport std/option (Option)\n" + src + "\n"
		if err := checkModuleSource(t, "row_guard", full); err == nil || !strings.Contains(err.Error(), "IO") {
			t.Fatalf("must reject hidden IO: %v\n%s", err, src)
		}
	}
}

// #1958 Phase 2b (deferred, F2 still stands for these): a function value READ
// FROM A BINDING whose row is closed does not fit a wider slot. #1826 closes a
// binding's compatibility tail when it is read as a value, so F-B (which opens
// lambda rows at inference) does not reach it, whether the binding is a named
// function or a let-bound lambda. v0.56.1 accepted both only because the
// unguarded row unifier dropped the slot's extra labels. Declare the value with
// the slot's row, make the slot row-polymorphic, or pass the lambda inline
// (an inline or eta-expanded lambda opens on use). When Phase 2b lands, flip
// these to must-accept.
func TestEffectRowOpenClosedPhase2bDeferred(t *testing.T) {
	head := "func loud(s: string) -> () ! {IO} = println(s)\nfunc runWide(f: (string) -> () ! {IO, FS}, s: string) -> () ! {IO, FS} = f(s)\n"
	for name, src := range map[string]string{
		"named":            `func via() -> () ! {IO, FS} = runWide(loud, "x")`,
		"let-bound lambda": `func via() -> () ! {IO, FS} = { let f = \s. println(s); runWide(f, "x") }`,
	} {
		t.Run(name, func(t *testing.T) {
			full := "module row_guard\nimport std/io (println)\n" + head + src + "\n"
			if err := checkModuleSource(t, "row_guard", full); err == nil || !strings.Contains(err.Error(), "FS") {
				t.Fatalf("Phase 2b is deferred: a closed {IO} value must not yet fit {IO, FS}: %v", err)
			}
		})
	}
}
