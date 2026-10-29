package smt

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

func TestEncodeLetLexicalParameterScopeRestored(t *testing.T) {
	previous := activeParamRenames
	activeParamRenames = map[string]string{"n": "$p_n"}
	defer func() { activeParamRenames = previous }()
	expr := &core.Let{Name: "n", Value: &core.Var{Name: "n"}, Body: &core.Var{Name: "n"}}
	got, err := EncodeExpr(expr)
	if err != nil || got != "(let ((n $p_n)) n)" {
		t.Fatalf("let scope capture: %s %v", got, err)
	}
	if activeParamRenames["n"] != "$p_n" {
		t.Fatal("let encoding lost enclosing parameter rename")
	}
}
func TestEncodePatternLexicalParameterScopeRestored(t *testing.T) {
	previous := activeParamRenames
	activeParamRenames = map[string]string{"n": "$p_n", "other": "$p_other"}
	defer func() { activeParamRenames = previous }()
	pat := &core.ConstructorPattern{Name: "Ok", Args: []core.CorePattern{&core.ConstructorPattern{Name: "Some", Args: []core.CorePattern{&core.VarPattern{Name: "n"}}}}}
	got, err := encodePatternScoped(&core.Var{Name: "n"}, pat)
	if err != nil || got != "n" {
		t.Fatalf("pattern capture: %s %v", got, err)
	}
	if activeParamRenames["n"] != "$p_n" || activeParamRenames["other"] != "$p_other" {
		t.Fatal("pattern scope not restored")
	}
}
