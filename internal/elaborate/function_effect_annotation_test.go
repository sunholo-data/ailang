package elaborate

import (
	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/types"
	"testing"
)

func TestFunctionEffectAnnotations(t *testing.T) {
	budget := 3
	e := NewElaborator()
	fn := e.astTypeToInternalType(&ast.FuncType{Return: &ast.SimpleType{Name: "int"}, Effects: []ast.EffectAnnotation{
		{Name: "Rand", Budget: &budget, Params: []ast.EffectParam{{Key: "mode", Value: "seeded"}}},
		{Name: "e", IsRowVar: true},
	}}).(*types.TFunc2)
	row := fn.EffectRow
	if _, ok := row.Labels["Rand"]; !ok {
		t.Fatal("lost concrete label")
	}
	if row.Tail == nil || row.Tail.Name != "e" {
		t.Fatal("lost written tail")
	}
	if row.Budgets["Rand"] == nil || *row.Budgets["Rand"] != 3 || row.Params["Rand"]["mode"] != "seeded" {
		t.Fatal("lost annotation metadata")
	}
	concrete := e.astTypeToInternalType(&ast.FuncType{Return: &ast.SimpleType{Name: "int"}, Effects: []ast.EffectAnnotation{{Name: "IO"}}}).(*types.TFunc2)
	if concrete.EffectRow.Tail == nil {
		t.Fatal("L1-open compatibility tail missing")
	}
}
