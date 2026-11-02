package pipeline

import (
	"strings"
	"testing"
)

// The rejections in TestEffectRowOpenClosedM2Deferred came from #1958 M2
// (830934d23), reverted for v0.56.1: with M2's open-vs-closed check, a lambda
// whose body unions two effectful calls was rejected by a concrete callback
// slot, because inference closes that lambda's row to {} (combineTails drops
// distinct tails). M2 can return once inference keeps those rows open; the
// must-accept cases below are the regression guard for that.
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

// #1958 M2 (reverted for v0.56.1, see the comment on the test above): an
// annotated IO callback must not fit a closed pure slot. Re-enable with M2.
func TestEffectRowOpenClosedM2Deferred(t *testing.T) {
	t.Skip("#1958 M2 reverted for v0.56.1: needs inference to keep multi-effect lambda rows open first")
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
