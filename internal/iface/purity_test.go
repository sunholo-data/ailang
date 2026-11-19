package iface

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

func TestDerivedPurity_Rows(t *testing.T) {
	empty := types.EmptyEffectRow()
	open := &types.Row{Kind: types.EffectRow, Labels: map[string]types.Type{}, Tail: &types.RowVar{Name: "e", Kind: types.EffectRow}}
	other := &types.Row{Kind: types.EffectRow, Labels: map[string]types.Type{}, Tail: &types.RowVar{Name: "other", Kind: types.EffectRow}}
	io := &types.Row{Kind: types.EffectRow, Labels: map[string]types.Type{"IO": types.TUnit}}
	mixed := &types.Row{Kind: types.EffectRow, Labels: map[string]types.Type{"IO": types.TUnit}, Tail: open.Tail}
	callback := &types.TFunc2{Params: []types.Type{types.TInt}, Return: types.TInt, EffectRow: open}
	declared := &ast.FuncDecl{Effects: []ast.EffectAnnotation{{Name: "e", IsRowVar: true}}}
	fn := func(row *types.Row, params ...types.Type) types.Type {
		return &types.TFunc2{Params: params, Return: types.TInt, EffectRow: row}
	}
	for _, tc := range []struct {
		name string
		typ  types.Type
		decl *ast.FuncDecl
		want bool
	}{
		{"nil row", fn(nil), nil, true}, {"empty", fn(empty), nil, true},
		{"label", fn(io), nil, false}, {"unrelated tail", fn(other, callback), &ast.FuncDecl{}, false},
		{"shared latent tail", fn(open, callback), &ast.FuncDecl{}, true},
		{"declared shared tail", fn(open, callback), declared, false},
		{"missing declaration fails closed", fn(open, callback), nil, false},
		{"outer label and shared tail", fn(mixed, callback), &ast.FuncDecl{}, false},
		{"callback only", fn(empty, callback), declared, true},
		{"nested callback", fn(open, &types.TList{Element: callback}), &ast.FuncDecl{}, true},
		{"tail only in return", &types.TFunc2{Return: callback, EffectRow: open}, &ast.FuncDecl{}, false},
		{"closed outer returning effectful function", &types.TFunc2{Return: fn(io), EffectRow: empty}, nil, true},
		{"scalar", types.TInt, nil, true},
		{"record polymorphism is not an effect", &types.TRecord2{Row: &types.Row{Kind: types.RecordRow, Labels: map[string]types.Type{}, Tail: &types.RowVar{Name: "r", Kind: types.RecordRow}}}, nil, true},
		{"nested record function", &types.TRecord2{Row: &types.Row{Kind: types.RecordRow, Labels: map[string]types.Type{"f": fn(io)}}}, nil, false},
		{"pure function in value", &types.TList{Element: fn(empty)}, nil, true},
		{"effectful function in value", &types.TList{Element: fn(io)}, nil, false},
		{"open function in value", &types.TList{Element: fn(open)}, nil, false},
		{"callback nested in value", &types.TList{Element: fn(empty, callback)}, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := derivePurity(&types.Scheme{Type: tc.typ}, tc.decl); got != tc.want {
				t.Fatalf("purity=%v want %v", got, tc.want)
			}
		})
	}
	if derivePurity(nil, nil) || derivePurity(&types.Scheme{}, nil) {
		t.Fatal("missing checked type must fail closed")
	}
	// The keyword is deliberately irrelevant to metadata.
	for _, keyword := range []bool{false, true} {
		if derivePurity(&types.Scheme{Type: fn(io)}, &ast.FuncDecl{IsPure: keyword}) {
			t.Fatal("keyword overwrote checked IO row")
		}
	}
}

func TestDerivedPurity_CyclicValue(t *testing.T) {
	record := &types.TRecord{Fields: map[string]types.Type{}}
	record.Fields["self"] = record
	record.Fields["callback"] = &types.TFunc2{Return: types.TInt, EffectRow: &types.Row{Kind: types.EffectRow, Labels: map[string]types.Type{"IO": types.TUnit}}}
	if derivePurity(&types.Scheme{Type: record}, nil) {
		t.Fatal("cycle hid nested IO")
	}
}

// Build's published signal, not just the helper, must reflect checked rows.
func TestBuilder_DerivedPurity(t *testing.T) {
	env := types.NewTypeEnv()
	env.BindScheme("effectful", &types.Scheme{Type: &types.TFunc2{Return: types.TInt, EffectRow: &types.Row{Kind: types.EffectRow, Labels: map[string]types.Type{"IO": types.TUnit}}}})
	builder := NewBuilder("test/purity", env)
	prog := &core.Program{Decls: []core.CoreExpr{&core.Let{Name: "effectful", Value: &core.Lit{Kind: core.IntLit, Value: 1}}}, Meta: map[string]*core.DeclMeta{"effectful": {IsExport: true}}}
	ifc, err := builder.Build(prog, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if ifc.Exports["effectful"].Purity {
		t.Fatal("builder still publishes stub purity")
	}
}
