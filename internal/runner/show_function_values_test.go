package runner

import (
	"path/filepath"
	"testing"
)

// Function values on every engine (M-V1-SHOW-TOTAL review item 2): show of a
// function, builtin, closure or a constructor used as a value is
// `<function>`, and an n-ary constructor used as a value is callable. The
// VM used to lower an unapplied `Some` to a zero-field ADT: show printed
// `Some` there and `<function>` on the evaluator, and map(Some, xs) failed
// with "CallClosure: expected closure, got ADT". This runs the programs; the
// value-level differential (internal/vm) cannot see engine representation.
func TestShowFunctionValuesAllEngines(t *testing.T) {
	const src = `module show_fn_values
import std/io (println)
import std/list (map, foldl)
import std/string (toUpper)
import std/option (Option, Some, None)
import std/result (Result, Ok, Err)

func add(a: int, b: int) -> int = a + b

func twice(f: int -> Option[int], x: int) -> Option[int] = f(x)

export func main() -> () ! {IO} {
  let k = 3;
  let some = Some;
  let ok = Ok;
  println(show(Some));
  println(show(None));
  println(show([Some, Some]));
  println(show((Ok, Err, None)));
  println(show(some));
  println(show(some(1)));
  println(show(ok("x")));
  println(show(map(Some, [1, 2])));
  println(show(map(Err, ["e"])));
  println(show(twice(Some, 7)));
  println(show(\x. x + k));
  println(show(add));
  println(show(toUpper));
  println(show({f: Some, g: \x. x, n: None}));
  println(show(foldl(\acc x. acc + x, 0, [1, 2, 3])))
}
`
	const want = "<function>\nNone\n[<function>, <function>]\n(<function>, <function>, None)\n<function>\nSome(1)\nOk(\"x\")\n[Some(1), Some(2)]\n[Err(\"e\")]\nSome(7)\n<function>\n<function>\n<function>\n{f: <function>, g: <function>, n: None}\n6\n"
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	file := filepath.Join(t.TempDir(), "show_fn_values.ail")
	for _, engine := range allEngines {
		t.Run(engine, func(t *testing.T) {
			if got := runOnEngine(t, file, src, engine); got != want {
				t.Fatalf("%s:\n%s", engine, firstLineDiff(got, want))
			}
		})
	}
}
