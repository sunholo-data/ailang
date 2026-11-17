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
	// ev and od are generic in n and, independently, in their result.
	for _, want := range []string{"inc", "inc$numF", "shapes", "shapes$numF", "ev", "od", "ev$numFI", "od$numFI", "ev$numFF", "ev$numIF"} {
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
	if m := prog.Meta["ev$numFI"]; m == nil || m.VariantOf != "ev" || len(m.Contracts) != 1 || m.IsExport {
		t.Errorf("ev$numFI's metadata: %+v", m)
	}
}

// A float use, from another module, of a binding generic over more numeric
// types than its variants are compiled for eagerly is TC_NUMVAR_001 there; the
// int use, and a float use in its own module, check.
func TestNumericVariants_WideBindingFromAnotherModule(t *testing.T) {
	lib := "module q/w\nexport pure func W(x, y) = {a: x + 1, b: y + 0, c: 0, d: 0, e: 1}\n" +
		"export pure func local() -> float = W(1.5, 2.5).a\n"
	_, err := writeAndCheck(t, map[string]string{
		"q/w.ail": lib,
		"q/m.ail": "module q/m\nimport q/w (W)\nexport func main() -> () ! {IO} = println(show(W(1.5, 2).a))\n",
	}, "q/m.ail")
	wantCode(t, err, types.NumVariantLimitCode, "this use needs one of them at float")
	_, err = writeAndCheck(t, map[string]string{
		"q/w.ail": lib,
		"q/m.ail": "module q/m\nimport q/w (W)\nexport func main() -> () ! {IO} = println(show(W(1.5, 2.5).b))\n",
	}, "q/m.ail")
	wantCode(t, err, types.NumVariantLimitCode, "this use needs 2 of them at float")
	// Also from inside a generic function, whose float variant needs it.
	_, err = writeAndCheck(t, map[string]string{
		"q/w.ail": lib,
		"q/m.ail": "module q/m\nimport q/w (W)\npure func g(x) = W(x, 2).a\nexport func main() -> () ! {IO} = println(show(g(1.5)))\n",
	}, "q/m.ail")
	wantCode(t, err, types.NumVariantLimitCode, "W from q/w")
	if _, err := writeAndCheck(t, map[string]string{
		"q/w.ail": lib,
		"q/m.ail": "module q/m\nimport q/w (W, local)\nexport func main() -> () ! {IO} = println(\"${show(W(1, 2).a)} ${show(local())}\")\n",
	}, "q/m.ail"); err != nil {
		t.Fatalf("the int use and the library's own float use must check: %v", err)
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
