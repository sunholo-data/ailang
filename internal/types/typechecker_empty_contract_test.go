package types

import (
	"github.com/sunholo-data/ailang/internal/ast"
	"testing"
)

func TestLegacyFunctionExplicitEmptyContract(t *testing.T) {
	tc := NewTypeChecker()
	omitted := tc.astTypeToType(&ast.FuncType{Return: &ast.SimpleType{Name: "int"}}).(*TFunc2)
	explicit := tc.astTypeToType(&ast.FuncType{Return: &ast.SimpleType{Name: "int"}, Effects: []ast.EffectAnnotation{}}).(*TFunc2)
	if omitted.EffectRow.Tail != nil || omitted.ConcreteEffectContract {
		t.Fatal("legacy omitted behavior changed")
	}
	if explicit.EffectRow.Tail != nil || !explicit.ConcreteEffectContract || explicit.ImplicitEffectContract {
		t.Fatalf("explicit empty lost: %+v", explicit)
	}
}
