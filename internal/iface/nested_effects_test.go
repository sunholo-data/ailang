package iface

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/types"
)

// TestRestoreNestedEffectRow_PreservesParamEffects covers the
// M-IFACE-NESTED-EFFECTS regression: when a top-level function declared
// `func foo(cb: (X) -> () ! {IO}) -> ()` is type-checked, the inferred
// scheme's nested function-typed parameter loses its effect row (the
// typechecker erases nested effect rows during unification). Without this
// restoration, importing modules see `cb: (X) -> ()` (closed empty effect)
// and any caller passing an effectful lambda fails to unify with
// "incompatible closed rows: r1 has extra labels [], r2 has extra labels [IO]".
func TestRestoreNestedEffectRow_PreservesParamEffects(t *testing.T) {
	// Simulate the typechecker's output: nested function type with effects erased.
	typedCallbackParam := &types.TFunc2{
		Params:    []types.Type{&types.TCon{Name: "StreamChunk"}},
		EffectRow: types.EmptyEffectRow(), // ← erased by unification
		Return:    types.TUnit,
	}
	typedFn := &types.TFunc2{
		Params:    []types.Type{typedCallbackParam},
		EffectRow: &types.Row{Kind: types.EffectRow, Labels: map[string]types.Type{"AI": types.TUnit, "IO": types.TUnit}},
		Return:    types.TUnit,
	}

	// Simulate the AST FuncDecl with the original `! {IO}` annotation on the param.
	astCallbackParam := &ast.FuncType{
		Params: []ast.Type{&ast.SimpleType{Name: "StreamChunk"}},
		Return: &ast.SimpleType{Name: "()"},
		Effects: []ast.EffectAnnotation{
			{Name: "IO", IsRowVar: false},
		},
	}
	astFn := &ast.FuncDecl{
		Name:       "dispatch_step",
		Params:     []*ast.Param{{Name: "cb", Type: astCallbackParam}},
		ReturnType: &ast.SimpleType{Name: "()"},
	}

	restored, ok := applyLabelsFromAST(typedFn, astFn).(*types.TFunc2)
	if !ok {
		t.Fatalf("applyLabelsFromAST: expected *types.TFunc2, got %T", restored)
	}
	if len(restored.Params) != 1 {
		t.Fatalf("restored.Params: want 1, got %d", len(restored.Params))
	}
	restoredCb, ok := restored.Params[0].(*types.TFunc2)
	if !ok {
		t.Fatalf("restored.Params[0]: expected *types.TFunc2, got %T", restored.Params[0])
	}
	if restoredCb.EffectRow == nil || len(restoredCb.EffectRow.Labels) == 0 {
		t.Fatalf("restored callback param has no effect row; want {IO}")
	}
	if _, hasIO := restoredCb.EffectRow.Labels["IO"]; !hasIO {
		t.Errorf("restored callback param effect row = %v; want IO label present", restoredCb.EffectRow.Labels)
	}
}

// TestRestoreNestedEffectRow_NoEffectsInAST verifies the restoration does
// NOT erase a typechecker-inferred row when the AST has no explicit effect
// annotation (row-polymorphic / inferred-only callback types).
func TestRestoreNestedEffectRow_NoEffectsInAST(t *testing.T) {
	originalRow := &types.Row{Kind: types.EffectRow, Labels: map[string]types.Type{"FS": types.TUnit}}
	typedCallbackParam := &types.TFunc2{
		Params:    []types.Type{&types.TCon{Name: "int"}},
		EffectRow: originalRow,
		Return:    types.TUnit,
	}
	astCallbackParam := &ast.FuncType{
		Params:  []ast.Type{&ast.SimpleType{Name: "int"}},
		Return:  &ast.SimpleType{Name: "()"},
		Effects: nil, // AST has no `! {...}` annotation
	}

	got := restoreNestedEffectRow(typedCallbackParam, astCallbackParam)
	gotFn, ok := got.(*types.TFunc2)
	if !ok {
		t.Fatalf("restoreNestedEffectRow: expected *types.TFunc2, got %T", got)
	}
	if _, hasFS := gotFn.EffectRow.Labels["FS"]; !hasFS {
		t.Errorf("restoreNestedEffectRow erased the typechecker-inferred FS effect: %v", gotFn.EffectRow.Labels)
	}
}
