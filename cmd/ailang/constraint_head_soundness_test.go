package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #2247: a class constraint on a type with a constructor at its head and a
// variable inside (Num[Box[α]], Ord[[a]], Fractional[Option[α]]) was neither
// resolved (it is not ground) nor deferred to anything that checks it (it is
// not a bare variable), so it was dropped. `check` passed every program below
// and the evaluator then failed with RT002 (scrutinee: int) and the VM with
// GET_TAG on Int. The same constraint with a ground argument (Num[Box[int]])
// was always rejected; now both are.
func TestConstraintOnAppliedTypeIsResolved(t *testing.T) {
	bin := buildAilang(t)
	cases := []struct {
		module string
		src    string
		want   string // substring of the no-instance diagnostic
	}{
		// The issue's three repros, verbatim.
		{"un25_local_poly", `type Box[a] = Full(a) | Empty
pure func g[a](o: Box[a]) -> int = match o { Full(_) => 1, Empty => 0 }
export func main() -> () ! {IO} = println(show(g(7)))`, "No instance for Num[Box["},
		{"un28_list_lit", `pure func g[a](xs: [a]) -> int = match xs { [] => 0, _ :: _ => 1 }
export func main() -> () ! {IO} = println(show(g(7)))`, "No instance for Num[list["},
		{"un11_toplevel_prelude", `pure func g(x) = match x { Some(_) => 1, None => 2 }
export func main() -> () ! {IO} = println(show(g(42)))`, "No instance for Num[Option["},
		// The review's other shapes: let-bound, inline and letrec lambdas.
		{"un1_let_prelude", `export func main() -> () ! {IO} = {
  let g = \x. match x { Some(_) => 1, None => 2 };
  println(show(g(42)))
}`, "No instance for Num[Option["},
		{"un7_inline", `export func main() -> () ! {IO} = println(show((\x. match x { Some(_) => 1, None => 2 })(42)))`, "No instance for Num[Option["},
		{"un8_letrec", `export func main() -> () ! {IO} = {
  letrec g = \x. match x { Some(_) => 1, None => g(None) };
  println(show(g(42)))
}`, "No instance for Num[Option["},
		// Every class, not just Num.
		{"float_lit", `type Box[a] = Full(a) | Empty
pure func g[a](o: Box[a]) -> int = match o { Full(_) => 1, Empty => 0 }
export func main() -> () ! {IO} = println(show(g(7.5)))`, "No instance for Fractional[Box["},
		{"result_lit", `import std/result (Result, Ok, Err)
pure func isOk[a, e](r: Result[a, e]) -> bool = match r { Ok(_) => true, Err(_) => false }
export func main() -> () ! {IO} = println(show(isOk(3)))`, "No instance for Num[Result["},
		{"ord_option", `import std/option (Option, Some, None)
pure func less[a](x: Option[a], y: Option[a]) -> bool = x < y
export func main() -> () ! {IO} = println("defined")`, "No instance for Ord[Option[a]]"},
		{"ord_list", `pure func less[a](xs: [a], ys: [a]) -> bool = xs < ys
export func main() -> () ! {IO} = println("defined")`, "No instance for Ord[list[a]]"},
		{"num_list", `pure func dbl[a](xs: [a]) -> [a] = xs + xs
export func main() -> () ! {IO} = println("defined")`, "No instance for Num[list[a]]"},
		// Inside a lambda that is never called: still unsatisfiable.
		{"let_lambda_lit", `type Box[a] = Full(a) | Empty
pure func g[a](o: Box[a]) -> int = match o { Full(_) => 1, Empty => 0 }
export func main() -> () ! {IO} = {
  let h = \u. g(7);
  println(show(h(())))
}`, "No instance for Num[Box["},
	}
	for _, c := range cases {
		t.Run(c.module, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), c.module+".ail")
			src := "module " + c.module + "\n" + c.src + "\n"
			if err := os.WriteFile(file, []byte(src), 0o644); err != nil {
				t.Fatal(err)
			}
			stdout, stderr, code := runAilangBin(t, bin, "check", file)
			if code == 0 {
				t.Fatalf("check must reject %s, passed:\n%s", c.module, stdout)
			}
			if out := stdout + stderr; !strings.Contains(out, c.want) {
				t.Errorf("want %q in the diagnostic, got:\n%s", c.want, out)
			}
		})
	}
}

// Negative controls: constraints that are solved, defaulted or correctly
// deferred to a caller still check and run identically on both engines.
const constraintHeadControls = `module constraint_head_controls
import std/list (foldl, map)
import std/option (Option, Some, None)
type Box[a] = Full(a) | Empty
pure func g[a](o: Box[a]) -> int = match o { Full(_) => 1, Empty => 0 }
pure func maxOf[a](x: a, y: a) -> a = if x > y then x else y
pure func addAny(x, y) = x + y
pure func sumAll(xs: [int]) -> int = foldl(\acc x. acc + x, 0, xs)
pure func ins[a](x: a, xs: [a]) -> [a] = match xs { [] => [x], y :: rest => if x <= y then x :: xs else y :: ins(x, rest) }
pure func sameList[a](xs: [a], ys: [a]) -> bool = xs == ys
pure func bump(r) = {r | ver: r.ver + 1}
pure func firstOr(o, d) = match o { Some(v) => v, None => d }
export func main() -> () ! {IO} = {
  println(show(g(Full(7))));
  println(show(g(Full(7.5))));
  println(show(g(Empty)));
  println(show(maxOf(3, 4)));
  println(show(maxOf(3.5, 1.25)));
  println(show(maxOf("a", "b")));
  println(show(addAny(1, 2)));
  println(show(addAny(1.5, 2.0)));
  println(show(sumAll([1, 2, 3])));
  println(show(ins(2, [1, 3])));
  println(show(sameList([1, 2], [1, 2])));
  println(show(None == None));
  println(show([] == []));
  println(show(Some(1) == Some(1)));
  println(show(bump({ver: 1, name: "x"}).ver));
  println(show(firstOr(Some(5), 0)));
  println(show(firstOr(None, 2.5)));
  let x = 1;
  let y = 2.5;
  println(show(x + 1));
  println(show(y * 2.0));
  println(show(1 + 2.5));
  println(show(map(\z. z * 2, [1, 2, 3])));
  let mx = \a. \b. if a > b then a else b;
  println(show(mx(3.14)(2.71)));
  println(show(mx(10)(20)))
}
`

func TestConstraintOnAppliedTypeControls(t *testing.T) {
	bin := buildAilang(t)
	file := filepath.Join(t.TempDir(), "constraint_head_controls.ail")
	if err := os.WriteFile(file, []byte(constraintHeadControls), 0o644); err != nil {
		t.Fatal(err)
	}
	if stdout, stderr, code := runAilangBin(t, bin, "check", file); code != 0 {
		t.Fatalf("controls must check:\n%s%s", stdout, stderr)
	}
	want := strings.Join([]string{
		"1", "1", "0", "4", "3.5", "b", "3", "3.5", "6", "[1, 2, 3]",
		"true", "true", "true", "true", "2", "5", "2.5", "2", "5.0", "3.5",
		"[2, 4, 6]", "3.14", "20",
	}, "\n")
	for _, engine := range [][]string{
		{"run", "--quiet", "--caps", "IO", file},
		{"run", "--quiet", "--caps", "IO", "--bytecode", "--strict-bytecode", file},
	} {
		stdout, stderr, code := runAilangBin(t, bin, engine...)
		if code != 0 {
			t.Errorf("%v: exit %d\n%s", engine, code, stderr)
			continue
		}
		if got := strings.TrimSpace(stdout); got != want {
			t.Errorf("%v:\ngot\n%s\nwant\n%s", engine, got, want)
		}
	}
}
