package pipeline

import (
	"strings"
	"testing"
)

// #1958 re-land F-A1/F-A2: an effect union inside a lambda must not close to
// {} just because its operands are still bare tails when it is built. Before
// the fix, an IO lambda passed through a PURE std/list slot was rejected when
// its body made one call and accepted when it made two (D-R2: both reject).
const unionResolveHeader = "module union_resolve\nimport std/list (map, any)\nimport std/io (println)\nimport std/debug as Debug\n" +
	"func ex(n: int) -> int ! {IO} = { println(\"e\"); n }\n"

func TestEffectUnionResolve_PureSlotRejectsEveryBodyShape(t *testing.T) {
	for name, src := range map[string]string{
		"single call":   `func via(xs: [int]) -> [int] ! {IO} = map(\x. ex(x), xs)`,
		"named":         `func via(xs: [int]) -> [int] ! {IO} = map(ex, xs)`,
		"any single":    `func via(xs: [int]) -> bool ! {IO} = any(\x. ex(x) > 0, xs)`,
		"block":         `func via(xs: [int]) -> [int] ! {IO} = map(\x. { println(show(x)); x }, xs)`,
		"two calls":     `func via(xs: [int]) -> [int] ! {IO} = map(\x. { ex(x); ex(x) }, xs)`,
		"branches":      `func via(xs: [int]) -> [int] ! {IO} = map(\x. if x > 1 then ex(x) else ex(0), xs)`,
		"any block":     `func via(xs: [int]) -> bool ! {IO} = any(\x. { println(show(x)); true }, xs)`,
		"effectful arg": `func via(xs: [int]) -> [int] ! {IO} = map(\x. ex(ex(x)), xs)`,
	} {
		t.Run(name, func(t *testing.T) {
			err := checkModuleSource(t, "union_resolve", unionResolveHeader+src+"\n")
			if err == nil || !strings.Contains(err.Error(), "IO") {
				t.Fatalf("an IO lambda must not fit a pure callback slot: %v", err)
			}
			if strings.Contains(err.Error(), "internal invariant") {
				t.Fatalf("want an effect diagnostic, got an invariant: %v", err)
			}
		})
	}
}

// F-A2: a recursive self-call contributes nothing to its own row (rho = L u rho
// has the least solution rho = L), but a callback parameter's row is NOT the
// self row and must still be charged.
func TestEffectUnionResolve_RecursionAbsorbsOnlyTheSelfRow(t *testing.T) {
	loop := "func loop[e](f: (int) -> () ! {e}, n: int) -> () ! {e} = if n == 0 then () else { f(n); loop(f, n - 1) }\n"
	for _, tt := range []struct {
		name, src string
		reject    bool
	}{
		{"recursion through a ghost effect (debug_concurrent shape)", "func logN(tag: string, i: int) -> () = if i == 3 then () else let _ = Debug.log(\"${tag}:${show(i)}\") in logN(tag, i + 1)\n", false},
		{"callback row still charged", loop + "pure func bad(n: int) -> () = loop(\\x. println(show(x)), n)\n", true},
		// The callback's `e` is an enclosing-scope row, not the self row: absorbing
		// it would hide it. Base accepted this recursive form only because the union
		// dropped `e`; the non-recursive `f(1)` is rejected on base and here alike.
		{"recursive body must declare its callback row", "func bad[e](f: (int) -> () ! {e}, n: int) -> () = if n == 0 then () else { f(n); bad(f, n - 1) }\n", true},
		{"callback row in IO caller", loop + "func good(n: int) -> () ! {IO} = loop(\\x. println(show(x)), n)\n", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := checkModuleSource(t, "union_resolve", unionResolveHeader+tt.src)
			if !tt.reject {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), "effect checking failed") {
				t.Fatalf("callback effects must be charged through recursion: %v", err)
			}
		})
	}
}
