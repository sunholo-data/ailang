package runner

import (
	"path/filepath"
	"strings"
	"testing"
)

// floatShadowSource: a block, or a parenthesised let-in, in an expression
// position next to a read of the name its let shadows. Before #2388 ANF
// completion moved that let out over the whole enclosing expression, so the
// sibling read saw the inner value on EVERY engine (the evaluator too).
const floatShadowSource = `module float_shadow

import std/io (println)
import std/string (stringToInt)

func add(a: int, b: int) -> int = a + b

func ap(f: int -> int, x: int) -> int = f(x)

func letIn(a: int) -> int = let r = a in let s = (let r = 2 in r * 10) in r + s

export func main() -> () ! {IO} {
  let r = 1;
  println(show(add({ let r = 2; r * 10 }, r)));
  println(show(add(r, { let r = 2; r * 10 })));
  println(show(r + { let r = 2; r * 10 } + r));
  println(show([{ let r = 2; r }, r]));
  println(show({a: { let r = 2; r }, b: r}.b));
  println(show(if { let r = 0; r > 0 } then r else r + 10));
  println(show(match ({ let r = 4; r }, r) { (x, y) => x * 3 + y }));
  println(show(match stringToInt("${show({ let r = 2; r * 10 })}${show(r)}") { Some(n) => n, None => 0 }));
  println(show(ap(\z. z + r, { let r = 3; r })));
  println(show(letIn(1)))
}
`

// What lexical scope prints. Comment: what every engine printed on dev.
var floatShadowWant = strings.Join([]string{
	"21",     // call argument, read by a later sibling: 22
	"21",     // call argument, read by an earlier sibling: 22 (the call itself sat inside the floated let)
	"22",     // binary operands on both sides: 24
	"[2, 1]", // list element: [2, 2]
	"1",      // record field: 2
	"11",     // if condition: 10
	"13",     // tuple element: 16
	"201",    // string interpolation: 202
	"4",      // higher-order argument captured by the lambda beside it: 6
	"21",     // let-in value: 22
}, "\n") + "\n"

func TestFloatedLetDoesNotCaptureSiblings(t *testing.T) {
	file := filepath.Join(t.TempDir(), "float_shadow.ail")
	for _, engine := range allEngines {
		t.Run(engine, func(t *testing.T) {
			if got := runOnEngine(t, file, floatShadowSource, engine); got != floatShadowWant {
				t.Fatalf("stdout:\n%s\nwant:\n%s", got, floatShadowWant)
			}
		})
	}
}
