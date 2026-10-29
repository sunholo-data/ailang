package iface

import (
	"bytes"
	"encoding/json"
	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/types"
	"strings"
	"testing"
)

func TestExplicitEmptyIfaceConversionAndRestore(t *testing.T) {
	astFn := &ast.FuncType{Params: []ast.Type{&ast.SimpleType{Name: "int"}}, Return: &ast.SimpleType{Name: "int"}, Effects: []ast.EffectAnnotation{}}
	check := func(t *testing.T, fn *types.TFunc2) {
		t.Helper()
		if fn.EffectRow == nil || fn.EffectRow.Tail != nil || len(fn.EffectRow.Labels) != 0 || !fn.ConcreteEffectContract || fn.ImplicitEffectContract {
			t.Fatalf("empty contract lost: %+v", fn)
		}
	}
	check(t, astTypeToInternalType(astFn).(*types.TFunc2))
	inferred := &types.TFunc2{Params: []types.Type{types.TInt}, Return: types.TInt, EffectRow: &types.Row{Kind: types.EffectRow, Labels: map[string]types.Type{}, Tail: &types.RowVar{Name: "old", Kind: types.EffectRow}}, ImplicitEffectContract: true}
	check(t, restoreNestedEffectRow(inferred, astFn).(*types.TFunc2))
	outer := &types.TFunc2{Params: []types.Type{inferred}, Return: inferred, EffectRow: types.EmptyEffectRow()}
	restored := applyLabelsFromAST(outer, &ast.FuncDecl{Params: []*ast.Param{{Name: "f", Type: astFn}}, ReturnType: astFn}).(*types.TFunc2)
	check(t, restored.Params[0].(*types.TFunc2))
	check(t, restored.Return.(*types.TFunc2))
}

func TestExplicitEmptyIfaceContractIdentity(t *testing.T) {
	makeIface := func(concrete bool) *Iface {
		cb := &types.TFunc2{Params: []types.Type{types.TInt}, Return: types.TInt, EffectRow: types.EmptyEffectRow(), ConcreteEffectContract: concrete, ImplicitEffectContract: !concrete}
		i := NewIface("contract")
		i.AddType("Callback", 0)
		i.TypeAliases["Callback"] = cb
		i.AddType("Box", 0)
		i.AddConstructor("Box", "Box", []types.Type{cb}, &types.TCon{Name: "Box"})
		i.AddExport("keep", &types.Scheme{Type: &types.TFunc2{Params: []types.Type{cb}, Return: cb, EffectRow: types.EmptyEffectRow()}}, true)
		return i
	}
	a, _ := makeIface(false).ToNormalizedJSON()
	b, _ := makeIface(true).ToNormalizedJSON()
	if bytes.Equal(a, b) {
		t.Fatal("empty callback ownership disappeared from normalized iface")
	}
	var before, after InterfaceJSON
	if err := json.Unmarshal(a, &before); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &after); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(after.Types[0].Ctors[0], "!{}") || !strings.Contains(after.Types[1].Alias, "!{}") || !strings.Contains(after.Funcs[0].Type, "!{}") {
		t.Fatalf("missing ctor/alias/export contract: %s", b)
	}
	ha, _ := HashProjection(&before)
	hb, _ := HashProjection(&after)
	if bytes.Equal(ha, hb) {
		t.Fatal("contract edit has same signature hash projection")
	}
	if strings.Join(SignatureSet(&before), "\n") == strings.Join(SignatureSet(&after), "\n") {
		t.Fatal("contract edit has same signature set")
	}
}
