package pipeline

import (
	"strings"
	"testing"
)

// #2074: a LOCAL let-rec lambda that is both recursive and effectful — the
// standard read/print loop — failed with "same row variable with different
// extensions" after #1958 M1 (ad9aabc57). Top-level recursive functions, pure
// local recursion and non-recursive effectful lambdas were unaffected.
func TestEffectUnionResolve_LocalRecursiveEffectfulLetRec(t *testing.T) {
	const header = "module union_resolve\nimport std/io (println, readLine)\n"
	for _, tt := range []struct {
		name, src string
		reject    bool
	}{
		{"print loop", `export func main() -> () ! {IO} {
  letrec output = \k. if k > 3 then () else { println(show(k)); output(k + 1) };
  output(1)
}`, false},
		{"print loop over a captured bound", `export func main() -> () ! {IO} {
  let limit = 3;
  letrec output = \k. {
    if k > limit then ()
    else {
      let line = show(k);
      println(line);
      output(k + 1)
    }
  };
  output(1);
  println("done")
}`, false},
		{"read loop with an accumulator", `export func main() -> () ! {IO} {
  letrec go = \n acc. if n == 0 then acc else { let l = readLine(); go(n - 1, acc ++ [l]) };
  let xs = go(2, []);
  println(show(xs))
}`, false},
		{"curried read loop", `func readRows(n: int) -> [string] ! {IO} {
  letrec loop = \i. \acc.
    if i >= n then acc
    else {
      let line = readLine();
      loop(i + 1)(acc ++ [line])
    };
  loop(0)([])
}
export func main() -> () ! {IO} = println(show(readRows(2)))`, false},
		{"curried IO loop inside a pure func is still rejected", `pure func bad(n: int) -> () {
  letrec loop = \i. \acc. if i > n then () else { println(acc); loop(i + 1)(acc) };
  loop(1)("x")
}`, true},
		{"recursive IO loop inside a pure func is still rejected", `pure func bad(n: int) -> () {
  letrec output = \k. if k > n then () else { println(show(k)); output(k + 1) };
  output(1)
}`, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := checkModuleSource(t, "union_resolve", header+tt.src+"\n")
			if !tt.reject {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), "IO") {
				t.Fatalf("a pure func must not run IO through a local loop: %v", err)
			}
		})
	}
}
