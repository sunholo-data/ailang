package runner

import (
	"path/filepath"
	"testing"
)

// A curried lambda passed where a two-parameter function is expected is
// called with two arguments. The evaluator applies them one after another;
// the VM refused (`expects 1 args, got 2`). Under --bytecode the failure came
// after "start" was printed, so the run exited 1 instead of falling back.
// Covers the three VM call sites: a HOF builtin's callback (foldl), a
// tail call (apply2) and a non-tail call (applyPlus).
func TestCurriedCallbackOverApplication(t *testing.T) {
	source := `module curried_callbacks
import std/list (foldl)
import std/io (println)

func apply2(f: (int, int) -> int, a: int, b: int) -> int = f(a, b)

func applyPlus(f: (int, int) -> int, a: int, b: int) -> int = f(a, b) + 1

export func main() -> () ! {IO} = {
  println("start");
  println(show(foldl(\acc. \x. acc + x, 0, [1, 2, 3])));
  println(show(apply2(\x. \y. x * 10 + y, 1, 2)));
  println(show(applyPlus(\x. \y. x * 10 + y, 3, 4)))
}
`
	file := filepath.Join(t.TempDir(), "curried_callbacks.ail")
	const want = "start\n6\n12\n35\n"
	for _, engine := range allEngines {
		t.Run(engine, func(t *testing.T) {
			if got := runOnEngine(t, file, source, engine); got != want {
				t.Fatalf("stdout = %q, want %q", got, want)
			}
		})
	}
}
