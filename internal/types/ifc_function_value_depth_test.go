package types

import (
	"fmt"
	"github.com/sunholo-data/ailang/internal/ast"
	"strings"
	"testing"
)

func nestedCallableType(depth int) string {
	out := "string -> string"
	for i := 0; i < depth; i++ {
		out = "() -> (" + out + ")"
	}
	return out
}

// A parameter can return a function repeatedly before the final sink is applied.
// The old three-result probe cutoff silently erased this demand.
func TestIFCFunctionValueReturnedProbeDepth(t *testing.T) {
	for _, depth := range []int{3, 4, 12, 40} {
		t.Run(fmt.Sprint(depth), func(t *testing.T) {
			for _, tainted := range []bool{false, true} {
				var s strings.Builder
				s.WriteString("module r/probes\n" + fvPrelude)
				fmt.Fprintf(&s, "pure func use(f: %s, x: string) -> string { f%s(x) }\n", nestedCallableType(depth), strings.Repeat("()", depth))
				for i := 0; i < depth; i++ {
					next := "gate"
					if i+1 < depth {
						next = fmt.Sprintf("get%d", i+1)
					}
					fmt.Fprintf(&s, "pure func get%d() -> (%s) { %s }\n", i, nestedCallableType(depth-i-1), next)
				}
				arg := `"clean"`
				if tainted {
					arg = `fromInbox("hi")`
				}
				fmt.Fprintf(&s, "export func main() -> string { use(get0, %s) }\n", arg)
				errs := fvCheck(t, s.String())
				if tainted && len(errs) == 0 {
					t.Fatalf("depth %d silently accepted a sink leak", depth)
				}
				if !tainted && len(errs) > 0 {
					t.Fatalf("depth %d rejected clean callback chain: %v", depth, errs)
				}
			}
		})
	}
}

func TestIFCFunctionValueReplayBeyondThirtyTwo(t *testing.T) {
	c := newIFCChecker(&ast.File{})
	leaf := single(&ifcFn{kind: fnDeclared, name: "gate", params: []ifcDemand{{param: "x", forbid: []string{"email"}}}})
	for i := 0; i < 64; i++ {
		leaf = single(&ifcFn{kind: fnLambda, owner: fmt.Sprintf("lambda%d", i), deferred: []ifcDeferred{{target: leaf, args: []ifcVal{{label: LabelConst("email")}}}}})
	}
	c.applyCallable(leaf, nil, &ast.FuncCall{}, LabelBottom(), []string{"deep"}, 0)
	if len(c.errs) == 0 {
		t.Fatal("deep replay silently omitted its sink demand")
	}
}

func TestIFCFunctionValueReplayCycle(t *testing.T) {
	c := newIFCChecker(&ast.File{})
	leaf := single(&ifcFn{kind: fnDeclared, name: "gate", params: []ifcDemand{{param: "x", forbid: []string{"email"}}}})
	s := newSummary("f:loop")
	s.deferred = []ifcDeferred{
		{target: single(&ifcFn{kind: fnTop, name: "loop"})},
		{target: leaf, args: []ifcVal{{label: LabelConst("email")}}},
	}
	c.summaries["loop"] = s
	c.applyCallable(single(&ifcFn{kind: fnTop, name: "loop"}), nil, &ast.FuncCall{}, LabelBottom(), nil, 0)
	if len(c.errs) != 1 {
		t.Fatalf("cycle must terminate without skipping the reachable sink: %v", c.errs)
	}
}

func TestIFCFunctionValueInferredMultipleForbids(t *testing.T) {
	for _, arg := range []string{`"clean"`, `fromInbox("x")`, `fromSecret("x")`} {
		src := "module r/multiforbid\n" + fvPrelude + `
pure func fromSecret(s: string) -> string<secret> { s }
pure func noSecret(s: string{not secret}) -> string { s }
pure func wrap(y: string) -> string { let _ = gate(y); noSecret(y) }
export func main() -> string { wrap(` + arg + `) }
`
		errs := fvCheck(t, src)
		if arg == `"clean"` && len(errs) != 0 {
			t.Fatalf("clean wrapper: %v", errs)
		}
		if arg != `"clean"` && (len(errs) != 1 || errs[0].Kind != SinkRefinementError) {
			t.Fatalf("both inferred forbids must survive: %s: %v", arg, errs)
		}
	}
}

func TestIFCFunctionValueRemainingCarrierForms(t *testing.T) {
	for _, tc := range []struct{ name, decls, body string }{
		{"record update", fvUpper, `let r = { f: upper } in let s = {r | f: gate} in s.f(fromInbox("hi"))`},
		{"array", `import std/array as A
`, `match A.toList(#[gate]) { f :: _ => f(fromInbox("hi")), [] => "" }`},
		{"ADT", `type Box = BoxFn(string -> string)
`, `match BoxFn(gate) { BoxFn(f) => f(fromInbox("hi")) }`},
		{"block letrec", ``, `{ letrec f = \s. gate(s); f(fromInbox("hi")) }`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			errs := fvCheck(t, "module r/carrier\n"+tc.decls+fvPrelude+`export func main() -> string { `+tc.body+` }`)
			if len(errs) == 0 {
				t.Fatal("carrier dropped its sink requirement")
			}
			for _, err := range errs {
				if err.Kind != SinkRefinementError {
					t.Fatalf("wrong diagnostic: %v", errs)
				}
			}
		})
	}
}
