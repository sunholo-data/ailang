package elaborate

import (
	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/types"
	"testing"
)

func TestFunctionEffectEmptyContract(t *testing.T) {
	for _, tt := range []struct {
		name                     string
		effects                  []ast.EffectAnnotation
		open, concrete, implicit bool
	}{
		{"omitted", nil, true, false, true},
		{"explicit_empty", []ast.EffectAnnotation{}, false, true, false},
		{"concrete", []ast.EffectAnnotation{{Name: "IO"}}, true, true, false},
		{"polymorphic", []ast.EffectAnnotation{{Name: "e", IsRowVar: true}}, true, false, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := NewElaborator()
			fn := e.astTypeToInternalType(&ast.FuncType{Params: []ast.Type{&ast.SimpleType{Name: "int"}}, Return: &ast.SimpleType{Name: "int"}, Effects: tt.effects}).(*types.TFunc2)
			if (fn.EffectRow.Tail != nil) != tt.open || fn.ConcreteEffectContract != tt.concrete || fn.ImplicitEffectContract != tt.implicit {
				t.Fatalf("contract: row=%v concrete=%v implicit=%v", fn.EffectRow, fn.ConcreteEffectContract, fn.ImplicitEffectContract)
			}
		})
	}
}
