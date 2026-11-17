package main

import (
	"strings"
	"testing"
)

// #2378: an integer literal in a generic top-level binding takes the type the
// binding is used at, on every engine, in-module and across modules. The
// operator shapes are the generic matrix's lit_* modules; these are the rules
// around them.

// The repro of #2378, run on both engines: inc(2.5) is 3.5, and int code is
// unchanged (inc(2) is 3, show(1 + 1) is 2, 10 / 4 is 2).
func TestNumericVariants_Issue2378Repro(t *testing.T) {
	bin := buildAilang(t)
	dir := writeTree(t, map[string]string{"lc.ail": `module lc
func inc(x) = x + 1
let K = 3
export func main() -> () ! {IO} = println("${show(inc(2.5))} ${show(inc(2))} ${show(1 + 1)} ${show(10 / 4)} ${show(K * 0.5)} ${show(K / 2)}")
`})
	ailangIn(t, bin, dir, "check", "lc.ail").want(t, "check", 0, "No errors found")
	for _, engine := range [][]string{nil, {"--strict-bytecode"}} {
		args := append([]string{"run", "--quiet", "--caps", "IO", "--entry", "main"}, engine...)
		ailangIn(t, bin, dir, append(args, "lc.ail")...).want(t, "run "+strings.Join(engine, ""), 0, "3.5 3 2 2 1.5 1")
	}
}

// A binding generic over more than NumVariantEager numeric types (a record
// constant of integer fields) gets only the variants its own module uses: in
// the module it runs at float, and another module's float use is
// TC_NUMVAR_001 at check time, never a run-time failure. Its int use is fine.
func TestNumericVariants_WideBindingAcrossModules(t *testing.T) {
	bin := buildAilang(t)
	dir := writeTree(t, map[string]string{
		"lib/w.ail": `module lib/w
export pure func W(x) = {a: x + 1, b: 0, c: 0, d: 0, e: 1}
export pure func local() -> float = W(1.5).a
`,
		"good.ail": `module good
import lib/w (W, local)
export func main() -> () ! {IO} = println("${show(W(1).a)} ${show(local())}")
`,
		"bad.ail": `module bad
import lib/w (W)
export func main() -> () ! {IO} = println(show(W(1.5).a))
`,
	})
	for _, engine := range [][]string{nil, {"--strict-bytecode"}} {
		args := append([]string{"run", "--quiet", "--caps", "IO", "--entry", "main"}, engine...)
		ailangIn(t, bin, dir, append(args, "good.ail")...).want(t, "good", 0, "2 2.5")
	}
	ailangIn(t, bin, dir, "check", "bad.ail").want(t, "check bad", 1, "TC_NUMVAR_001", "bad.ail:3")
	ailangIn(t, bin, dir, "check", "--json", "bad.ail").want(t, "check --json bad", 1, `"code": "TC_NUMVAR_001"`)
}

// A let-bound lambda is one run-time value, so a literal (or a use of a
// binding with variants) keeps it monomorphic: its uses decide its type, and
// uses at two numeric types are a check error, not a run-time one. Used at one
// type it agrees with the same body at top level.
func TestNumericVariants_LetBoundIsMonomorphic(t *testing.T) {
	bin := buildAilang(t)
	dir := writeTree(t, map[string]string{
		"one.ail": `module one
pure func inc(x) = x + 1
pure func top(p) = {x: p.x * 1}
export func main() -> () ! {IO} {
  let f = \p. {x: p.x * 1};
  let g = \y. inc(y);
  println("${show(f({x: 1.5}).x)} ${show(top({x: 1.5}).x)} ${show(g(2.5))} ${show(inc(2.5))}")
}
`,
		"two.ail": `module two
pure func inc(x) = x + 1
export func main() -> () ! {IO} {
  let g = \y. inc(y);
  let n: int = 2;
  println("${show(g(2.5))} ${show(g(n))}")
}
`,
	})
	for _, engine := range [][]string{nil, {"--strict-bytecode"}} {
		args := append([]string{"run", "--quiet", "--caps", "IO", "--entry", "main"}, engine...)
		ailangIn(t, bin, dir, append(args, "one.ail")...).want(t, "one", 0, "1.5 1.5 3.5 3.5")
	}
	ailangIn(t, bin, dir, "check", "two.ail").want(t, "check two", 1, "two.ail:6")
}

// A contract on a generic function is checked at the type of the call: the
// float variant carries the clause at float (it failed at run time with
// --verify-contracts before), and a violation names the function as written.
func TestNumericVariants_ContractsAtTheCallType(t *testing.T) {
	bin := buildAilang(t)
	dir := writeTree(t, map[string]string{
		"ok.ail": `module ok
pure func inc(x) requires { x >= 0 } ensures { result > x } = x + 1
export func main() -> () ! {IO} = println("${show(inc(2))} ${show(inc(2.5))}")
`,
		"neg.ail": `module neg
pure func inc(x) requires { x >= 0 } = x + 1
export func main() -> () ! {IO} = println(show(inc(-2.5)))
`,
	})
	args := []string{"run", "--quiet", "--caps", "IO", "--verify-contracts", "--entry", "main"}
	ailangIn(t, bin, dir, append(args, "ok.ail")...).want(t, "ok", 0, "3 3.5")
	r := ailangIn(t, bin, dir, append(args, "neg.ail")...)
	r.want(t, "neg", 1, "requires failed in inc ")
	if strings.Contains(r.out, "$num") {
		t.Fatalf("a violation names the variant, not the function:\n%s", r.out)
	}
}
