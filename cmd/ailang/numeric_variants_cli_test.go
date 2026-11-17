package main

import (
	"strings"
	"testing"
)

// #2378: an integer literal in a generic top-level binding takes the type the
// binding is used at, on every engine, in-module and across modules. The
// operator shapes are the generic matrix's lit_* modules; these are the rules
// around them.

func runBoth(t *testing.T, bin, dir, file, want string, extra ...string) {
	t.Helper()
	for _, engine := range [][]string{nil, {"--strict-bytecode"}} {
		args := append([]string{"run", "--quiet", "--caps", "IO", "--entry", "main"}, extra...)
		args = append(append(args, engine...), file)
		ailangIn(t, bin, dir, args...).want(t, file+" "+strings.Join(engine, ""), 0, want)
	}
}

// The repro of #2378, run on both engines: inc(2.5) is 3.5, and int code is
// unchanged (inc(2) is 3, show(1 + 1) is 2, 10 / 4 is 2).
func TestNumericVariants_Issue2378Repro(t *testing.T) {
	bin := buildAilang(t)
	dir := writeTree(t, map[string]string{"lc.ail": `module lc
func inc(x) = x + 1
export func main() -> () ! {IO} = println("${show(inc(2.5))} ${show(inc(2))} ${show(1 + 1)} ${show(10 / 4)}")
`})
	ailangIn(t, bin, dir, "check", "lc.ail").want(t, "check", 0, "No errors found")
	runBoth(t, bin, dir, "lc.ail", "3.5 3 2 2")
}

// The rule: a numeric type variable that occurs in no parameter type is not
// generic. A constant and a result-only literal are ints (or floats when
// written 3.0); a variable in a parameter type stays generic. Using such an int
// at float is a check error located at the use, naming the fix (it passed
// check and failed at run time before).
func TestNumericVariants_ConstantsAndResultsAreNotGeneric(t *testing.T) {
	bin := buildAilang(t)
	dir := writeTree(t, map[string]string{
		"lib/c.ail": `module lib/c
export pure func K() = 3
export pure func cfg() = {lo: 1, hi: 10}
export pure func W(x) = {a: x + 1, b: 0}
export pure func sign(x) = if x > 0 then 1 else -1
`,
		"ok.ail": `module ok
import lib/c (K, cfg, W, sign)
let N = 3
let F = 3.0
export func main() -> () ! {IO} = println("${show(N * 2)} ${show(F * 0.5)} ${show(K())} ${show(cfg().hi)} ${show(W(1.5).a)} ${show(W(1.5).b)} ${show(sign(-2.5))}")
`,
		"bad.ail": `module bad
let N = 3
export func main() -> () ! {IO} = println(show(N * 0.5))
`,
	})
	runBoth(t, bin, dir, "ok.ail", "6 1.5 3 10 2.5 0 -1")
	ailangIn(t, bin, dir, "check", "bad.ail").want(t, "check bad", 1, "bad.ail:3", "this value is an int", "`N` is an int: its integer literal (at bad.ail:2:9)", "write that literal as a float (3.0)")
	r := ailangIn(t, bin, dir, "iface", "lib/c.ail")
	r.want(t, "iface", 0)
	for _, want := range []string{`"(())-\u003eint"`, `"(())-\u003e{hi: int, lo: int}"`, `"(a)-\u003e{a: a, b: int}"`, `"(a)-\u003eint"`} {
		if !strings.Contains(r.out, want) {
			t.Errorf("iface lacks %s:\n%s", want, r.out)
		}
	}
}

// A binding generic over more than NumVariantEager parameter-borne numeric
// types gets only the variants its own module uses. Compiling a library that
// uses it never fails; a use, in another module, at a combination nobody
// compiled is TC_NUMVAR_001 located at that use (in --json too), never a
// run-time failure. A use that needs no such variant runs.
func TestNumericVariants_WideBindingAcrossModules(t *testing.T) {
	bin := buildAilang(t)
	dir := writeTree(t, map[string]string{
		"lib/w.ail": `module lib/w
export pure func W(a, b, c, d, e) = (a + 1, b + 1, c + 1, d + 1, e + 1)
export pure func local() = W(1.5, 1, 1, 1, 1)
`,
		"lib/v.ail": `module lib/v
import lib/w (W)
export pure func V(x) = W(x, x, x, x, x)
`,
		"good.ail": `module good
import lib/w (W, local)
import lib/v (V)
export func main() -> () ! {IO} = println("${show(W(1, 2, 3, 4, 5))} ${show(local())} ${show(V(2))}")
`,
		"bad.ail": `module bad
import lib/v (V)
export func main() -> () ! {IO} = println(show(V(2.5)))
`,
	})
	runBoth(t, bin, dir, "good.ail", "(2, 3, 4, 5, 6) (2.5, 2, 2, 2, 2) (3, 3, 3, 3, 3)")
	ailangIn(t, bin, dir, "check", "bad.ail").want(t, "check bad", 1, "TC_NUMVAR_001", "bad.ail:3")
	r := ailangIn(t, bin, dir, "check", "--json", "bad.ail")
	r.want(t, "check --json bad", 1, `"code": "TC_NUMVAR_001"`, `"line": 3`)
}

// A let-bound lambda that calls a literal-carrying generic binding is copied
// per use: used at int and at float it gives each its own answer. One that is
// literal-free keeps base's run-time-corrected value and its outputs (the
// review's m09: `a / 4` stays integer division). One with its own literal
// (gen's h: `* 2`) is one value whose uses decide its type, as before #2378:
// in gen(2.5), h(1) is at float.
func TestNumericVariants_LetBoundLambdas(t *testing.T) {
	bin := buildAilang(t)
	dir := writeTree(t, map[string]string{
		"inner.ail": `module inner
import std/list (map)
pure func inc(x) = x + 1
pure func gen(x) { let g = \y. inc(y); let h = \z. g(z) * 2; (h(x), h(1)) }
export func main() -> () ! {IO} {
  let h = \u. inc(u);
  let n: int = 2;
  println("${show(h(2))} ${show(h(2.5))} ${show(h(n))} ${show(map(h, [0.5]))} ${show(gen(2.5))} ${show(gen(2))}")
}
`,
		"free.ail": `module free
func add(a, b) = a + b
export func main() -> () ! {IO} {
  let dbl = \u. add(u, u);
  let a = dbl(21);
  let b = dbl(0.25);
  println("ints=${show(a)} n=${show(a / 4)} f=${show(b)}")
}
`,
		// A lambda under a block (#2388 generalises it) is not copied per use: a
		// literal-free callee leaves it generic, a literal-carrying one makes it
		// one value, here used at float only.
		"block.ail": `module block
pure func dbl(x) = x + x
pure func inc(x) = x + 1
export func main() -> () ! {IO} {
  let f = { let k = 1; \u. dbl(u) };
  let g = { let k = 1; \u. inc(u) };
  println("${show(f(2))} ${show(f(2.5))} ${show(g(2.5))}")
}
`,
	})
	// gen's h is copied per use: h(1) is an int use, and gen's second
	// component, occurring in no parameter, is an int.
	runBoth(t, bin, dir, "inner.ail", "3 3.5 3 [1.5] (7.0, 4) (6, 4)")
	runBoth(t, bin, dir, "free.ail", "ints=42 n=10 f=0.5")
	runBoth(t, bin, dir, "block.ail", "4 5.0 3.5")
}

// A contract on a generic function is checked at the type of the call: the
// float variant carries its clauses at float, every clause of the declaration
// (several requires, requires and ensures, `result`, a clause calling another
// generic function), and a violation names the function as written. A
// monomorphic float function's clause calling a generic one calls its float
// variant.
func TestNumericVariants_ContractsAtTheCallType(t *testing.T) {
	bin := buildAilang(t)
	dir := writeTree(t, map[string]string{
		"ok.ail": `module ok
pure func inc(x) = x + 1
pure func a(x) requires { x >= 0 } ensures { result > x } = x + 1
pure func b(x) requires { x > 0, x < 10 } = x + 1
pure func c(x) requires { inc(x) > 0 } ensures { result == inc(x) } { x + 1 }
pure func f(x: float) -> float ensures { result == inc(x) } { x + 1.0 }
export func main() -> () ! {IO} = println("${show(a(2))} ${show(a(2.5))} ${show(b(2.5))} ${show(c(2.5))} ${show(f(2.5))}")
`,
		"neg.ail": `module neg
pure func inc(x) requires { x >= 0 } = x + 1
export func main() -> () ! {IO} = println(show(inc(-2.5)))
`,
	})
	args := []string{"run", "--quiet", "--caps", "IO", "--verify-contracts", "--entry", "main"}
	ailangIn(t, bin, dir, append(args, "ok.ail")...).want(t, "ok", 0, "3 3.5 3.5 3.5 3.5")
	r := ailangIn(t, bin, dir, append(args, "neg.ail")...)
	r.want(t, "neg", 1, "requires failed in inc ")
	if strings.Contains(r.out, "$num") {
		t.Fatalf("a violation names the variant, not the function:\n%s", r.out)
	}
}

// A run-time error inside a variant names the function as written, on the VM
// and in the --bytecode fallback note.
func TestNumericVariants_VMNamesTheFunction(t *testing.T) {
	bin := buildAilang(t)
	dir := writeTree(t, map[string]string{"e3.ail": `module e3
func qi(x, n) = (x + 1, n / 0)
export func main() -> () ! {IO} = println(show(qi(2.5, 3)))
`})
	for _, engine := range []string{"--strict-bytecode", "--bytecode"} {
		r := ailangIn(t, bin, dir, "run", "--quiet", engine, "--caps", "IO", "--entry", "main", "e3.ail")
		if !strings.Contains(r.out, "e3.qi ") || strings.Contains(r.out, "$num") {
			t.Errorf("%s: want the error to name e3.qi, got:\n%s", engine, r.out)
		}
	}
}
