package runner

import (
	"path/filepath"
	"strings"
	"testing"
)

// monoCaptureSource: monomorphisation copies a polymorphic let-bound lambda's
// body to its call site, and resolves an alias (`let f = g`) by name. Before
// #2401 the copied body's free names were re-resolved at the call site, so a
// let, a lambda parameter or a pattern variable there captured them, a local
// named like a top-level function captured a call of that function, and an
// alias followed a later rebinding. Comment: what every engine printed on
// dev (35f5ad6d6).
const monoCaptureSource = `module mono_capture

import std/io (println)
import std/list (length)

func r(n: int) -> int = n * 7 + 1

func viaPattern(k: int) -> int {
  let f = \x. k;
  match Some(5) { Some(k) => f(0), None => 0 }
}

func total(items: [int], tax: int) -> int {
  let addTax = \x. x + tax;
  match items { [] => 0, tax :: rest => addTax(100) }
}

func count(n: int, xss: [[int]]) -> int {
  let size = \ys. length(ys) + n;
  match xss { [] => 0, n :: _ => size(n) }
}

export func main() -> () ! {IO} {
  let k = 10;
  let f = \x. k;
  let g = \k. f(3);
  println(show(g(99)));
  println(show({ let k = 99; f(3) }));
  println(show(match Some(99) { Some(k) => f(3), None => 0 }));
  println(show(viaPattern(1)));
  println(show(total([7], 1)));
  println(show(count(100, [[1, 2]])));
  let r = \y. r(3) + 1;
  println(show(r(0)));
  let a = \x. x;
  let b = a;
  let a = \x. 7;
  println(show([b(3), a(3)]))
}
`

var monoCaptureWant = strings.Join([]string{
	"10",     // 99: the parameter k of g captured f's k
	"10",     // 99: the block's let k
	"10",     // 99: the pattern's k
	"1",      // 5
	"101",    // 101 (the pattern's tax was not read by addTax's copy here)
	"102",    // strict VM: arith ADD: type mismatch Int vs List (eval: error)
	"23",     // 24: the copied r(3) called the local r
	"[3, 7]", // [7, 7]: the alias b followed the later a
}, "\n") + "\n"

func TestMonomorphisationDoesNotCapture(t *testing.T) {
	file := filepath.Join(t.TempDir(), "mono_capture.ail")
	for _, engine := range allEngines {
		t.Run(engine, func(t *testing.T) {
			if got := runOnEngine(t, file, monoCaptureSource, engine); got != monoCaptureWant {
				t.Fatalf("stdout:\n%s\nwant:\n%s", got, monoCaptureWant)
			}
		})
	}
}
