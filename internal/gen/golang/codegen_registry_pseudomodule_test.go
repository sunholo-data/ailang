package golang

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

func TestPseudoModuleBuiltinEmitsReferencedHelper(t *testing.T) {
	prog := &core.Program{Decls: []core.CoreExpr{
		&core.Let{
			Name: "negate",
			Value: &core.Lambda{Params: []string{"x"}, Body: &core.App{
				Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "not_Bool"}},
				Args: []core.CoreExpr{&core.Var{Name: "x"}},
			}},
			Body: &core.Var{Name: "negate"},
		},
	}}

	code, err := New("pseudomodule_not").Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	generated := string(code)
	if !strings.Contains(generated, "NotBool(") {
		t.Fatalf("generated code does not reference NotBool helper:\n%s", generated)
	}
	if !strings.Contains(generated, "func NotBool(v interface{}) interface{}") {
		t.Fatalf("generated code references NotBool without emitting its helper:\n%s", generated)
	}
}

func TestPseudoModuleInlineBuiltinResolvesDirectly(t *testing.T) {
	got := New("pseudomodule_inline").resolveInlineBuiltin(
		"$builtin", "_str_eq", []string{`"left"`, `"right"`},
	)
	if got != `"left" == "right"` {
		t.Fatalf("pseudo-module inline lookup = %q, want direct _str_eq expansion", got)
	}
}
