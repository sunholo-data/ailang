package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/testutil"
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
		// An alias is judged by what it stands for, in both directions.
		{"alias_to_list_lit", `type Stack[a] = [a]
pure func sz[a](s: Stack[a]) -> int = match s { [] => 0, _ :: _ => 1 }
export func main() -> () ! {IO} = println(show(sz(7)))`, "No instance for Num[list["},
		{"alias_to_string_arith", `type Tagged[t] = string
pure func dbl[t](x: Tagged[t]) -> Tagged[t] = x + x
export func main() -> () ! {IO} = println(show(dbl("a")))`, "No instance for Num[Tagged[t]]"},
		// An alias body that names a type variable it does not bind (a typo, or
		// `type Any = a`) must not turn a ground constraint into a deferred one:
		// expanding Num[Count] to Num[integer] made it a constraint on a fresh
		// variable, which the top-level boundary dropped, so inc("x") checked
		// and crashed (#2285 second review, v01-v06).
		{"free_var_alias", `type Any = a
pure func inc(x: Any) -> Any = x + 1
export func main() -> () ! {IO} = println(show(inc(41)))`, "No instance for Num[Any]"},
		{"free_var_alias_str", `type Any = a
pure func inc(x: Any) -> Any = x + 1
pure func useS(s: string) -> string = s
export func main() -> () ! {IO} = println(useS(inc("s")))`, "No instance for Num[Any]"},
		{"free_var_alias_lt", `type Any = a
pure func lt(x: Any, y: Any) -> bool = x < y
export func main() -> () ! {IO} = println(show(lt(\u. u, \u. u)))`, "No instance for Ord[Any]"},
		{"phantom_param_unused_var", `type Any2[b] = c
pure func lt(x: Any2[int], y: Any2[int]) -> bool = x < y
export func main() -> () ! {IO} = println(show(lt(Some(1), Some(2))))`, "No instance for Ord[Any2[int]]"},
		{"typo_alias", `type Count = integer
pure func inc(c: Count) -> Count = c + 1
pure func tag(s: string) -> string = s
export func main() -> () ! {IO} = { println(show(inc(41))); println(tag(inc("x"))) }`, "No instance for Num[Count]"},
		{"typo_alias_eq_fn", `type Handler = handler
pure func same(a: Handler, b: Handler) -> bool = a == b
export func main() -> () ! {IO} = println(show(same(\u. u + 1, \u. u + 1)))`, "No instance for Eq[Handler]"},
		// A self-referential alias is reported as written, not unfolded.
		{"self_referential_alias", `type L[a] = [L[a]]
pure func f(x: L[int]) -> bool = x == x
export func main() -> () ! {IO} = println("hi")`, "No instance for Eq[L[int]] in scope"},
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
			out := stdout + stderr
			if !strings.Contains(out, c.want) {
				t.Errorf("want %q in the diagnostic, got:\n%s", c.want, out)
			}
			if strings.Contains(out, "list[list[list[") {
				t.Errorf("diagnostic must name the alias as written, not unfold it:\n%s", out)
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

// Aliases of primitives are seen through before a constraint is judged
// (#2247 review): Num[Id[a]] with `type Id[a] = int` is Num[int]. Before the
// alias expansion the #2247 fix rejected every one of these, and several were
// already wrong on dev (Num[Score], Eq[Id[a]] and Num[Same[int]] were rejected;
// Meters[u] crashed at run time because `m * 2.0` ran on the Int dictionary).
const constraintAliasControls = `module constraint_alias_controls
type Score = int
type Id[a] = int
type Ref[a] = Id[a]
type Same[a] = a
type Meters[u] = float
type Tagged[t] = string
type UserId[t] = int
type User = {id: UserId[User], name: string}
pure func inc[a](x: Id[a]) -> Id[a] = x + 1
pure func incS(x: Id[string]) -> Id[string] = x + 1
pure func incScore(x: Score) -> Score = x + 1
pure func sameScore(x: Score, y: Score) -> bool = x == y
pure func lessRef[a](x: Ref[a], y: Ref[a]) -> bool = x < y
pure func sameId[a](x: Id[a], y: Id[a]) -> bool = x == y
pure func sameIds[a](x: [Id[a]], y: [Id[a]]) -> bool = x == y
pure func incSame[a](x: Same[a]) -> Same[a] = x + 1
pure func dbl[u](m: Meters[u]) -> Meters[u] = m * 2.0
pure func cmpm[u](m: Meters[u], n: Meters[u]) -> bool = m >= n
pure func lessT[t](a: Tagged[t], b: Tagged[t]) -> bool = a < b
pure func nextId(u: User) -> int = u.id + 1
export func main() -> () ! {IO} = {
  println(show(inc(41)));
  println(show(incS(41)));
  println(show(incScore(41)));
  println(show(sameScore(3, 3)));
  println(show(lessRef(1, 2)));
  println(show(sameId(1, 2)));
  println(show(sameIds([1], [1])));
  println(show(incSame(41)));
  println(show(dbl(1.5)));
  println(show(cmpm(1.0, 2.0)));
  println(show(lessT("a", "b")));
  println(show(nextId({id: 4, name: "x"})))
}
`

func TestConstraintAliasControls(t *testing.T) {
	bin := buildAilang(t)
	file := filepath.Join(t.TempDir(), "constraint_alias_controls.ail")
	if err := os.WriteFile(file, []byte(constraintAliasControls), 0o644); err != nil {
		t.Fatal(err)
	}
	if stdout, stderr, code := runAilangBin(t, bin, "check", file); code != 0 {
		t.Fatalf("alias controls must check:\n%s%s", stdout, stderr)
	}
	want := strings.Join([]string{
		"42", "42", "42", "true", "true", "false", "true", "42", "3.0", "false", "true", "5",
	}, "\n")
	assertBothEngines(t, want, file, func(args ...string) (string, string, int) {
		return runAilangBin(t, bin, args...)
	})
}

// A record alias resolves Eq through the record's declared name, on both
// engines: `deriving (Eq)` on a record alias crashed the evaluator
// ("derived Eq for P: expected TaggedValue") and an alias of it (`type Q = P`)
// was rejected (#2285 second review, r02 and r07).
const constraintRecordAliasControls = `module constraint_record_alias_controls
type P = {x: int} deriving (Eq)
type Q = P
pure func sameP(p: P, q: P) -> bool = p == q
pure func sameQ(p: Q, q: Q) -> bool = p == q
export func main() -> () ! {IO} = {
  println(show(sameP({x: 1}, {x: 1})));
  println(show(sameP({x: 1}, {x: 2})));
  println(show(sameQ({x: 1}, {x: 1})));
  println(show(sameQ({x: 1}, {x: 3})))
}
`

func TestConstraintRecordAliasControls(t *testing.T) {
	bin := buildAilang(t)
	file := filepath.Join(t.TempDir(), "constraint_record_alias_controls.ail")
	if err := os.WriteFile(file, []byte(constraintRecordAliasControls), 0o644); err != nil {
		t.Fatal(err)
	}
	if stdout, stderr, code := runAilangBin(t, bin, "check", file); code != 0 {
		t.Fatalf("record alias controls must check:\n%s%s", stdout, stderr)
	}
	assertBothEngines(t, "true\nfalse\ntrue\nfalse", file, func(args ...string) (string, string, int) {
		return runAilangBin(t, bin, args...)
	})
}

// The alias may come from another module, through a chain.
func TestConstraintAliasAcrossModules(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "xm"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"xm/lib.ail": `module xm/lib
export type Id[a] = int
export type Score = int
export type Ref[a] = Id[a]
export pure func inc[a](x: Id[a]) -> Id[a] = x + 1
`,
		"xm/main.ail": `module xm/main
import xm/lib (Id, Score, Ref, inc)
pure func dbl[a](x: Id[a]) -> Id[a] = x * 2
pure func lessR[a](x: Ref[a], y: Ref[a]) -> bool = x < y
pure func same(x: Score, y: Score) -> bool = x == y
export func main() -> () ! {IO} = { println(show(inc(41))); println(show(dbl(21))); println(show(lessR(1, 2))); println(show(same(3, 3))) }
`,
	}
	for rel, src := range files {
		if err := os.WriteFile(filepath.Join(dir, rel), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run := func(args ...string) (string, string, int) {
		return testutil.RunBounded(t, dir, 60*time.Second, bin, args...)
	}
	if stdout, stderr, code := run("check", "xm/main.ail"); code != 0 {
		t.Fatalf("cross-module alias program must check:\n%s%s", stdout, stderr)
	}
	assertBothEngines(t, "42\n42\ntrue\ntrue", "xm/main.ail", run)
}

func assertBothEngines(t *testing.T, want, file string, run func(args ...string) (string, string, int)) {
	t.Helper()
	for _, engine := range [][]string{
		{"run", "--quiet", "--caps", "IO", file},
		{"run", "--quiet", "--caps", "IO", "--bytecode", "--strict-bytecode", file},
	} {
		stdout, stderr, code := run(engine...)
		if code != 0 {
			t.Errorf("%v: exit %d\n%s", engine, code, stderr)
			continue
		}
		if got := strings.TrimSpace(stdout); got != want {
			t.Errorf("%v:\ngot\n%s\nwant\n%s", engine, got, want)
		}
	}
}
