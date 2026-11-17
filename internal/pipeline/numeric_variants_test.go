package pipeline

import (
	"sort"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

// #2378: every Core form a generic body can hold is copied into its float
// variant, and the program's float uses are renamed to it. The run-time
// answers are pinned end to end by the generic matrix (cmd/ailang); this
// pins the compiled program's shape.
const variantShapes = `module m

pure func inc(x) = x + 1

pure func shapes(x, p, xs) {
  let neg = -x;
  let r = {a: x + 1, b: [x, 0], c: (x, 1)};
  let u = {p | v: p.v + 1};
  let arr = #[x, 2];
  let _a = arr;
  let k = \y. y * 2;
  letrec go = \i acc. if i <= 0 then acc else go(i - 1, acc + x) in {
    let total = match xs {
      [] => 0,
      [h, ...t] => h + go(2, 0)
    };
    if x > 0 then r.a + u.v + k(neg) + total + inc(x) else r.a
  }
}

pure func ev(n) requires { n >= 0 } = if n <= 0 then 0 else od(n - 1) + 1
pure func od(n) = if n <= 0 then 1 else ev(n - 1) + 1

export func main() -> () ! {IO} = println("${show(shapes(1.5, {v: 2.5, w: 0}, [0.5]))} ${show(shapes(1, {v: 2, w: 0}, [1]))} ${show(ev(3.0))} ${show(ev(3))}")
`

func TestNumericVariants_EveryFormIsCopied(t *testing.T) {
	res, err := writeAndCheck(t, map[string]string{"m.ail": variantShapes}, "m.ail")
	if err != nil {
		t.Fatalf("must check: %v", err)
	}
	prog := res.Artifacts.Core
	if prog == nil {
		t.Fatal("no Core program")
	}
	defined := map[string]bool{}
	for _, d := range prog.Decls {
		switch x := d.(type) {
		case *core.Let:
			defined[x.Name] = true
		case *core.LetRec:
			for _, b := range x.Bindings {
				defined[b.Name] = true
			}
		}
	}
	// ev and od are generic in n; their result occurs in no parameter, so it
	// is an int (the #2378 rule), not a second variable.
	for _, want := range []string{"inc", "inc$numF", "shapes", "shapes$numF", "ev", "od", "ev$numF", "od$numF"} {
		if !defined[want] {
			t.Errorf("no declaration %s; have %v", want, keys(defined))
		}
	}
	// The float call in main is renamed, the int one is not, and every name a
	// variant uses is defined (a variant's recursive calls name variants).
	used := map[string]bool{}
	for _, d := range prog.Decls {
		walkCoreExprs(d, func(e core.CoreExpr) {
			if v, ok := e.(*core.Var); ok {
				used[v.Name] = true
			}
		})
	}
	for name := range used {
		if strings.Contains(name, "$num") && !defined[name] {
			t.Errorf("a use names %s, which no declaration defines", name)
		}
	}
	if !used["shapes$numF"] || !used["shapes"] || !used["inc$numF"] {
		t.Errorf("main must call shapes and shapes$numF; uses %v", keys(used))
	}
	// The contract is the binding's, copied for the variant and marked as such.
	if m := prog.Meta["ev$numF"]; m == nil || m.VariantOf != "ev" || len(m.Contracts) != 1 || m.IsExport {
		t.Errorf("ev$numF's metadata: %+v", m)
	}
}

// A binding generic over more parameter-borne numeric types than its variants
// are compiled for eagerly has only the variants its module needs. Another
// module's use at a missing one is TC_NUMVAR_001 located at that use; a
// library whose generic function uses it compiles, and its int use checks.
func TestNumericVariants_WideBindingFromAnotherModule(t *testing.T) {
	lib := "module q/w\nexport pure func W(a, b, c, d, e) = (a + 1, b + 1, c + 1, d + 1, e + 1)\n" +
		"export pure func local() = W(1.5, 1, 1, 1, 1)\n"
	via := "module q/v\nimport q/w (W)\nexport pure func V(x) = W(x, x, x, x, x)\n"
	_, err := writeAndCheck(t, map[string]string{
		"q/w.ail": lib,
		"q/m.ail": "module q/m\nimport q/w (W)\nexport func main() -> () ! {IO} = println(show(W(2, 1.5, 2, 2, 2)))\n",
	}, "q/m.ail")
	wantCode(t, err, types.NumVariantLimitCode, "W from q/w is not compiled with 1 of its 5 numeric types at float")
	_, err = writeAndCheck(t, map[string]string{
		"q/w.ail": lib, "q/v.ail": via,
		"q/m.ail": "module q/m\nimport q/v (V)\nexport func main() -> () ! {IO} = println(show(V(2.5)))\n",
	}, "q/m.ail")
	wantCode(t, err, types.NumVariantLimitCode, "V from q/v")
	// In a generic function of this module, whose float variant needs it.
	_, err = writeAndCheck(t, map[string]string{
		"q/w.ail": lib,
		"q/m.ail": "module q/m\nimport q/w (W)\npure func g(x) = W(x, x, x, x, x)\nexport func main() -> () ! {IO} = println(show(g(1.5)))\n",
	}, "q/m.ail")
	wantCode(t, err, types.NumVariantLimitCode, "g at float needs a variant that cannot be compiled")
	if _, err := writeAndCheck(t, map[string]string{
		"q/w.ail": lib, "q/v.ail": via,
		"q/m.ail": "module q/m\nimport q/w (W, local)\nimport q/v (V)\nexport func main() -> () ! {IO} = println(\"${show(W(1, 2, 3, 4, 5))} ${show(local())} ${show(V(2))}\")\n",
	}, "q/m.ail"); err != nil {
		t.Fatalf("the int uses and the library's own float use must check: %v", err)
	}
}

// A binding no other module can see gets a variant only when this module uses
// it at that type; an exported one gets all of them, since an importer may.
func TestNumericVariants_PrivateBindingsAreGeneratedOnUse(t *testing.T) {
	res, err := writeAndCheck(t, map[string]string{"m.ail": "module m\n" +
		"pure func priv(x) = x + 1\n" +
		"pure func used(x) = x + 2\n" +
		"export pure func pub(x) = x + 3\n" +
		"export func main() -> () ! {IO} = println(\"${show(priv(1))} ${show(used(1.5))}\")\n"}, "m.ail")
	if err != nil {
		t.Fatalf("must check: %v", err)
	}
	defined := map[string]bool{}
	for _, d := range res.Artifacts.Core.Decls {
		if x, ok := d.(*core.Let); ok {
			defined[x.Name] = true
		}
	}
	for name, want := range map[string]bool{"priv$numF": false, "used$numF": true, "pub$numF": true} {
		if defined[name] != want {
			t.Errorf("declaration %s defined = %v, want %v; have %v", name, defined[name], want, keys(defined))
		}
	}
}

// A let-bound lambda that calls a literal-carrying generic binding is copied
// per use: the declaration holding it is replaced by a copy that binds the
// float copy next to the lambda, and the float use calls it. A literal-free
// callee leaves the lambda alone.
func TestNumericVariants_LetBoundLambdaIsCopiedPerUse(t *testing.T) {
	res, err := writeAndCheck(t, map[string]string{"m.ail": "module m\n" +
		"pure func inc(x) = x + 1\n" +
		"pure func add(a, b) = a + b\n" +
		"export func main() -> () ! {IO} {\n" +
		"  let h = \\u. inc(u);\n" +
		"  let d = \\u. add(u, u);\n" +
		"  println(\"${show(h(2))} ${show(h(2.5))} ${show(d(2))} ${show(d(0.5))}\")\n" +
		"}\n"}, "m.ail")
	if err != nil {
		t.Fatalf("must check: %v", err)
	}
	bound, used := map[string]bool{}, map[string]bool{}
	for _, d := range res.Artifacts.Core.Decls {
		walkCoreExprs(d, func(e core.CoreExpr) {
			switch x := e.(type) {
			case *core.Let:
				bound[x.Name] = true
			case *core.Var:
				used[x.Name] = true
			}
		})
	}
	for _, want := range []string{"h", "h$numF", "d"} {
		if !bound[want] || !used[want] {
			t.Errorf("%s: bound %v, used %v; bound %v", want, bound[want], used[want], keys(bound))
		}
	}
	if bound["d$numF"] {
		t.Errorf("a lambda calling a literal-free binding must not be copied; bound %v", keys(bound))
	}
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
