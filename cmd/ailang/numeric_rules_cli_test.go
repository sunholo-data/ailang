package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// #2393 review round 2, end to end on both engines.

// R1: a variable is int only when it carries a literal and no parameter
// position anywhere in the type takes it. Literal-free generics in containers,
// and a function in a result, stay generic exactly as before.
func TestNumericRules_LiteralFreeBindingsStayGeneric(t *testing.T) {
	bin := buildAilang(t)
	dir := writeTree(t, map[string]string{"t1.ail": `module t1
import std/list (map)
func add(a, b) = a + b
func mul(a, b) = a * b
let ops = (add, mul)
let fns = [add, mul]
let rec0 = {plus: add, times: mul}
func getOps() = (add, mul)
func pick(b) = if b then Some(add) else None
func mk() = \x. x + 1
func inc(x) = x + 1
func incs() = (inc, {f: inc})
export func main() -> () ! {IO} {
  match ops { (f, g) => println("tuple ${show(f(1.5, 2.0))} ${show(g(2, 3))}") };
  println("list ${show(map(\h. h(1.5, 2.0), fns))}");
  println("rec ${show(rec0.plus(1.5, 2.0))} ${show(rec0.times(2, 3))}");
  match getOps() { (f, g) => println("getOps ${show(f(1.5, 2.0))} ${show(g(2, 3))}") };
  match pick(true) { Some(f) => println("pick ${show(f(1.5, 2.0))}"), None => () };
  println("mk ${show(mk()(2))} ${show(mk()(2.5))}");
  match incs() { (f, r) => println("incs ${show(f(2.5))} ${show(r.f(2.5))}") }
}
`})
	for _, want := range []string{"tuple 3.5 6", "list [3.5, 3.0]", "rec 3.5 6", "getOps 3.5 6", "pick 3.5", "mk 3 3.5", "incs 3.5 3.5\n"} {
		runBoth(t, bin, dir, "t1.ail", want)
	}
}

// R2: which variables carry a literal is part of the interface. An imported
// literal-free generic is not taken to carry one (a let using it stays
// generic), the iface names the literal-carrying variables, and an importer's
// cache follows a dependency that gains or loses a literal.
func TestNumericRules_LiteralFlagCrossesModules(t *testing.T) {
	bin := buildAilang(t)
	lit := "module lib/ar\nexport func add(a, b) = a + b + 0\nexport func sq(x) = x * x\nexport func inc(x) = x + 1\n"
	free := "module lib/ar\nexport func add(a, b) = a + b\nexport func sq(x) = x * x\nexport func inc(x) = x + 1\n"
	dir := writeTree(t, map[string]string{"lib/ar.ail": free,
		"x1.ail": `module x1
import lib/ar (add)
export func main() -> () ! {IO} {
  let g = { let k = 1; \u. add(u, u) };
  let a = g(21);
  println("${show(a)} ${show(a / 4)} ${show(g(0.25))}")
}
`,
		"x3.ail": `module x3
import lib/ar (add, sq)
func pairs(ps: [(float, float)], qs: [(int, int)]) -> string {
  letrec n2 = \p. match p { (a, b) => add(sq(a), sq(b)) };
  "${show(n2((3.0, 4.0)))} ${show(n2((3, 4)))}"
}
export func main() -> () ! {IO} = println(pairs([], []))
`,
		"use.ail": "module use\nimport lib/ar (add)\nexport func main() -> () ! {IO} = println(show(add(1.5, 2.0)))\n",
		// A function over an imported literal-free generic carries no literal.
		"lib/two.ail": "module lib/two\nimport lib/ar (add, inc)\nexport func dbl(x) = add(x, x)\nexport func inc2(x) = inc(inc(x))\n",
	})
	runBoth(t, bin, dir, "x1.ail", "42 10 0.5")
	runBoth(t, bin, dir, "x3.ail", "25.0 25")
	r := ailangIn(t, bin, dir, "iface", "lib/ar.ail")
	r.want(t, "iface", 0)
	if n := strings.Count(r.out, `"literals"`); n != 1 || !strings.Contains(r.out, `"literals": [`) {
		t.Errorf("only inc carries a literal; iface has %d literals fields:\n%s", n, r.out)
	}
	r = ailangIn(t, bin, dir, "iface", "lib/two.ail")
	r.want(t, "iface two", 0)
	if n := strings.Count(r.out, `"literals"`); n != 1 {
		t.Errorf("only inc2 (over the imported inc's literal) carries one; iface has %d literals fields:\n%s", n, r.out)
	}
	// The dependency gains a literal, then loses it: each warm run must
	// agree with the cold one (the importer calls add$numF, then add).
	runBoth(t, bin, dir, "use.ail", "3.5")
	for _, src := range []string{lit, free} {
		if err := os.WriteFile(filepath.Join(dir, "lib/ar.ail"), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		runBoth(t, bin, dir, "use.ail", "3.5")
		runBoth(t, bin, dir, "x1.ail", "42 10 0.5")
	}
}

// R3: one rule for let-bound functions. A non-expansive value ending in a
// lambda — written directly, in a block, or recursive, its literal its own or
// a callee's — behaves like a top-level function; an expansive one is one
// value, and two calls of it at int and float are reported as exactly that.
func TestNumericRules_LetBoundFunctionsAreFunctions(t *testing.T) {
	bin := buildAilang(t)
	dir := writeTree(t, map[string]string{
		"lf.ail": `module lf
func ft(x) = x + 1
export func main() -> () ! {IO} {
  let f = \x. x + 1;
  let g = { let k = 1; \u. ft(u) };
  letrec s = \n. if n <= 0 then 0 else n + s(n - 1);
  let r = \p. {x: p.x * 1};
  let n: int = 2;
  println("${show(ft(n))} ${show(ft(2.5))} ${show(f(n))} ${show(f(2.5))} ${show(g(n))} ${show(g(2.5))} ${show(s(3))} ${show(s(2.5))} ${show(r({x: 7.5}).x)} ${show(r({x: 7}).x)}")
}
`,
		"one.ail": `module one
func inc(x) = x + 1
export func main() -> () ! {IO} {
  let h = if true then \u. inc(u) else \u. u;
  println("${show(h(2))} ${show(h(2.5))}")
}
`,
		"two.ail": `module two
func inc(x) = x + 1
export func main() -> () ! {IO} {
  let h = if true then \u. inc(u) else \u. u;
  let n: int = 2;
  println("${show(h(n))} ${show(h(2.5))}")
}
`,
	})
	runBoth(t, bin, dir, "lf.ail", "3 3.5 3 3.5 3 3.5 6 4.5 7.5 7")
	runBoth(t, bin, dir, "one.ail", "3.0 3.5") // one value: its uses decide float
	ailangIn(t, bin, dir, "check", "two.ail").want(t, "check two", 1,
		"two.ail:4:3: `h` is a single value, not a function", "uses at two.ail:6:19 and two.ail:6:33 need different ones (int and float)")
}

// R5: no compiled suffix — a numeric variant's ($numF) or the local
// uniquifier's ($sN) — reaches a user, in errors, contract violations or
// traces.
func TestNumericRules_NoCompiledSuffixReachesUsers(t *testing.T) {
	bin := buildAilang(t)
	dir := writeTree(t, map[string]string{"lk.ail": `module lk
func inc(x) requires { x > 0 } = x + 1
func dv(x) = x / 0
export func main() -> () ! {IO} {
  let h = \u. inc(u) * 2;
  let h = \u. h(u) + 1;
  println("${show(h(2))} ${show(h(2.5))}");
  println(show(dv(2)))
}
`, "ct.ail": `module ct
func inc(x) requires { x > 0 } = x + 1
export func main() -> () ! {IO} {
  let h = \u. inc(u);
  println(show(h(-2.5)))
}
`})
	suffix := regexp.MustCompile(`\$(num[IF]+|s[0-9]+)`)
	var all []string
	for _, args := range [][]string{
		{"run", "--quiet", "--caps", "IO", "--entry", "main", "lk.ail"},
		{"run", "--quiet", "--caps", "IO", "--entry", "main", "--strict-bytecode", "lk.ail"},
		{"run", "--quiet", "--caps", "IO", "--entry", "main", "--verify-contracts", "ct.ail"},
		{"run", "--quiet", "--caps", "IO", "--entry", "main", "--emit-trace", "jsonl", "--trace-tier", "deep", "ct.ail"},
	} {
		r := ailangIn(t, bin, dir, args...)
		all = append(all, r.out)
		if m := suffix.FindString(r.out); m != "" {
			t.Errorf("%v shows a compiled name (%s):\n%s", args, m, r.out)
		}
	}
	joined := strings.Join(all, "\n")
	for _, want := range []string{"7 8.0", "division by zero", "requires failed in inc", `"name":"h"`, `"name":"inc"`} {
		if !strings.Contains(joined, want) {
			t.Errorf("expected %q in the outputs:\n%s", want, joined)
		}
	}
}
