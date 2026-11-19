package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/pipeline"
)

// nameScopeEnginesSource: compile-time passes that resolve a name to "its"
// binding (monomorphisation, operator lowering, Var type resolution) must
// respect lexical scope, on every engine. Comment: what every engine printed
// on dev (c6b2b0727).
const nameScopeEnginesSource = `module name_scope

import std/io (println)

func cmp[a](o: Option[a], k: a) -> bool {
  let x = 1;
  match o { Some(x) => x == k, None => false }
}

func lt[a](o: Option[a], k: a) -> bool {
  let x = 1;
  match o { Some(x) => x < k, None => false }
}

func eqAfter[a](k: a, m: a) -> bool {
  let x = k;
  let s = { let x = 2; x };
  x == m
}

func neParam[a](x: a, m: a) -> bool {
  let s = { let x = 2; x };
  x != m
}

func geLambda[a](m: a) -> bool {
  let x = 1;
  let f = \x. x >= m;
  f(m)
}

func dv[a](x: a, m: a) -> a {
  let s = { let x = 2.5; x };
  x / m
}

export func main() -> () ! {IO} {
  println(show([cmp(Some("s"), "s"), lt(Some("a"), "b")]));
  let x = 1;
  let g = \p. match p { (x, y) => x == y };
  println(show(g(("a", "a"))));
  println(show([eqAfter("s", "s"), eqAfter(1.5, 1.5), neParam("a", "b"), geLambda("b"), geLambda(1.5)]));
  println(show(dv(7, 2)));
  let f = \x. x;
  let ap3 = \f. f(3);
  let viaMatch = \o. match o { Some(f) => f(3), None => 0 };
  let pair = \x. (x, x);
  let q = \pair. pair(4);
  println(show([ap3(\z. z + 1), viaMatch(Some(\z. z + 10))]));
  println(show(q(\z. (z, z + 1))));
  println(f("s"))
}
`

var nameScopeEnginesWant = strings.Join([]string{
	"[true, true]",                   // eq_Int: expected IntValue ... got *eval.StringValue
	"true",                           // (same crash class, tuple pattern in a lambda)
	"[true, true, true, true, true]", // eq_Int crash
	"3",                              // strict VM: arith DIV: type mismatch Float vs Int
	"[4, 13]",                        // [3, 3]: the parameter f was replaced by the outer f
	"(4, 5)",                         // (4, 4)
	"s",
}, "\n") + "\n"

func TestNameScopeAllEngines(t *testing.T) {
	file := filepath.Join(t.TempDir(), "name_scope.ail")
	for _, engine := range allEngines {
		t.Run(engine, func(t *testing.T) {
			if got := runOnEngine(t, file, nameScopeEnginesSource, engine); got != nameScopeEnginesWant {
				t.Fatalf("stdout:\n%s\nwant:\n%s", got, nameScopeEnginesWant)
			}
		})
	}
}

// valueRestrictionSource: a let the program wrote inside a binding's value
// stays nested there (#2388), so the value restriction treats `let … in` as
// ML does: a value when both parts are. The let-in and block forms now agree
// (on dev the block form was monomorphic and the let-in form, floated, was
// generalised whatever its inner value).
const valueRestrictionSource = `module vrestrict

import std/io (println)
import std/list (map)

func letIn(n: int) -> string = let f = (let k = 1 in \x. x) in "${show(f(n))}${f("s")}"

func block(n: int) -> string {
  let f = { let k = 1; \x. x };
  "${show(f(n))}${f("s")}"
}

func wrap(n: int) -> string = let w = (let k = 0 in \x. [x]) in "${show(map(w, [n]))}${show(w("s"))}"

func curried(n: int) -> string = let c = (let k = 1 in \a. \b. a) in "${show(c(n)("x"))}${c("y")(2)}"

func multi(n: int) -> string = let f = (let k = 1 in let j = 2 in \x. x) in "${show(f(n))}${f("s")}"

func letrecInside(n: int) -> string {
  let f = { letrec g = \m. if m <= 0 then 0 else g(m - 1); \x. x };
  "${show(f(n))}${f("s")}"
}

func numeric() -> string {
  let k = { let k0 = 1; \x. x + x };
  "${show(k(1))} ${show(k(1.5))}"
}

export func main() -> () ! {IO} {
  println(letIn(3));
  println(block(3));
  println(wrap(3));
  println(curried(3));
  println(multi(3));
  println(letrecInside(3));
  println(numeric())
}
`

func TestValueRestrictionNestedLet(t *testing.T) {
	file := filepath.Join(t.TempDir(), "vrestrict.ail")
	const want = "3s\n3s\n[[3]][\"s\"]\n3y\n3s\n3s\n2 3.0\n"
	for _, engine := range allEngines {
		t.Run(engine, func(t *testing.T) {
			if got := runOnEngine(t, file, valueRestrictionSource, engine); got != want {
				t.Fatalf("stdout:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

// checkVerdict compiles source and reports the type error text ("" = ok).
func checkVerdict(t *testing.T, name, source string) string {
	t.Helper()
	file := filepath.Join(t.TempDir(), name+".ail")
	if err := os.WriteFile(file, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := pipeline.Run(pipeline.Config{Mode: pipeline.ModeCheck, TransientRoot: true}, pipeline.Source{Filename: file, Code: source})
	if err != nil {
		return err.Error()
	}
	if len(res.Errors) > 0 {
		return res.Errors[0].Error()
	}
	return ""
}

// An expansive inner value (a call, an effect) keeps the binding
// monomorphic in both forms, so using it at two types is a type error.
func TestValueRestrictionExpansiveStaysMonomorphic(t *testing.T) {
	for name, src := range map[string]string{
		"letin_call": "module letin_call\npure func mk(n: int) -> int = n + 1\nfunc h(n: int) -> string = let f = (let c = mk(0) in \\x. x) in \"${show(f(n))}${f(\"s\")}\"\n",
		"block_call": "module block_call\npure func mk(n: int) -> int = n + 1\nfunc h(n: int) -> string {\n  let f = { let c = mk(0); \\x. x };\n  \"${show(f(n))}${f(\"s\")}\"\n}\n",
		"block_io":   "module block_io\nimport std/io (println)\nfunc h(n: int) -> string ! {IO} {\n  let f = { let c = println(\"once\"); \\x. x };\n  \"${show(f(n))}${f(\"s\")}\"\n}\n",
	} {
		t.Run(name, func(t *testing.T) {
			if msg := checkVerdict(t, name, src); !strings.Contains(msg, "int vs string") {
				t.Fatalf("verdict %q, want the int-vs-string unification error", msg)
			}
		})
	}
}

// The review's finding 2: a dummy let inside a tuple element of an if branch.
// The let-bound temporary is generalised, and generalisation closes a
// lambda's effect row to {}, which then cannot meet the other branch's {IO}:
// the same error as dev, and as binding the lambda with a let yourself. The
// version with no binding at all checks. This pins those verdicts.
func TestEffectRowVerdictDummyLet(t *testing.T) {
	const head = "module m\nimport std/io (println)\nfunc p(t: string, v: int) -> int ! {IO} { println(t); v }\nfunc ap(f: (int) -> int, x: int) -> int ! {IO} = f(x)\nexport func main() -> () ! {IO} {\n"
	cases := []struct{ name, body, want string }{
		{"dummy_let", `  let g = if true then (match (1, { let a = "d"; (\x. x) }) { (u, w) => w }) else (\y. p("t", y));`, "failed to unify effect rows"},
		{"let_bound", "  let h = \\x. x;\n  let g = if true then h else (\\y. p(\"t\", y));", "failed to unify effect rows"},
		{"no_binding", `  let g = if true then (match (1, (\x. x)) { (u, w) => w }) else (\y. p("t", y));`, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			src := head + c.body + "\n  println(show(ap(g, 3)))\n}\n"
			msg := checkVerdict(t, c.name, src)
			if (c.want == "") != (msg == "") || !strings.Contains(msg, c.want) {
				t.Fatalf("verdict %q, want %q", msg, c.want)
			}
		})
	}
}
