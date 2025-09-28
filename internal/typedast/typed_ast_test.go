package typedast

import (
	"testing"
	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/core"
)

func TestTypedExpr(t *testing.T) {
	typedExpr := TypedExpr{
		NodeID:    42,
		Span:      ast.Pos{Line: 10, Column: 5, File: "test.ail"},
		Type:      "Int", // Using string instead of types.Type
		EffectRow: nil,
		Core: &core.Lit{
			CoreNode: core.CoreNode{NodeID: 42},
			Kind:     core.IntLit,
			Value:    int64(5),
		},
	}

	// Test basic fields
	if typedExpr.NodeID != 42 {
		t.Errorf("TypedExpr.NodeID = %v, want %v", typedExpr.NodeID, 42)
	}

	expectedSpan := ast.Pos{Line: 10, Column: 5, File: "test.ail"}
	if typedExpr.Span != expectedSpan {
		t.Errorf("TypedExpr.Span = %v, want %v", typedExpr.Span, expectedSpan)
	}

	if typedExpr.Type != "Int" {
		t.Errorf("TypedExpr.Type = %v, want %v", typedExpr.Type, "Int")
	}

	if typedExpr.EffectRow != nil {
		t.Errorf("TypedExpr.EffectRow = %v, want nil", typedExpr.EffectRow)
	}

	if typedExpr.Core == nil {
		t.Error("TypedExpr.Core should not be nil")
	}
}

func TestTypedVar(t *testing.T) {
	typedVar := &TypedVar{
		TypedExpr: TypedExpr{
			NodeID: 1,
			Span:   ast.Pos{Line: 1, Column: 1, File: "test.ail"},
			Type:   "Int",
		},
		Name: "x",
	}

	// Test Name field
	if typedVar.Name != "x" {
		t.Errorf("TypedVar.Name = %v, want %v", typedVar.Name, "x")
	}

	// Test inherited fields
	if typedVar.NodeID != 1 {
		t.Errorf("TypedVar.NodeID = %v, want %v", typedVar.NodeID, 1)
	}

	// Verify it implements TypedNode interface
	var _ TypedNode = typedVar
}

func TestTypedLit(t *testing.T) {
	typedLit := &TypedLit{
		TypedExpr: TypedExpr{
			NodeID: 1,
			Span:   ast.Pos{Line: 1, Column: 1, File: "test.ail"},
			Type:   "Int",
		},
		Kind:  core.IntLit,
		Value: int64(42),
	}

	// Test fields
	if typedLit.Kind != core.IntLit {
		t.Errorf("TypedLit.Kind = %v, want %v", typedLit.Kind, core.IntLit)
	}

	if typedLit.Value != int64(42) {
		t.Errorf("TypedLit.Value = %v, want %v", typedLit.Value, int64(42))
	}

	// Verify it implements TypedNode interface
	var _ TypedNode = typedLit
}

func TestTypedLambda(t *testing.T) {
	bodyVar := &TypedVar{
		TypedExpr: TypedExpr{
			NodeID: 2,
			Type:   "Int",
		},
		Name: "x",
	}

	typedLambda := &TypedLambda{
		TypedExpr: TypedExpr{
			NodeID: 1,
			Span:   ast.Pos{Line: 1, Column: 1, File: "test.ail"},
			Type:   "Int -> Int",
		},
		Params:     []string{"x"},
		ParamTypes: []interface{}{"Int"},
		Body:       bodyVar,
	}

	// Test fields
	if len(typedLambda.Params) != 1 {
		t.Errorf("TypedLambda.Params length = %v, want %v", len(typedLambda.Params), 1)
	}

	if typedLambda.Body != bodyVar {
		t.Error("TypedLambda.Body not set correctly")
	}

	// Verify it implements TypedNode interface
	var _ TypedNode = typedLambda
}

func TestTypedLet(t *testing.T) {
	value := &TypedLit{
		TypedExpr: TypedExpr{
			NodeID: 2,
			Type:   "Int",
		},
		Kind:  core.IntLit,
		Value: int64(5),
	}

	body := &TypedVar{
		TypedExpr: TypedExpr{
			NodeID: 3,
			Type:   "Int",
		},
		Name: "x",
	}

	typedLet := &TypedLet{
		TypedExpr: TypedExpr{
			NodeID: 1,
			Span:   ast.Pos{Line: 1, Column: 1, File: "test.ail"},
			Type:   "Int",
		},
		Name:   "x",
		Scheme: nil, // Simplified - no scheme
		Value:  value,
		Body:   body,
	}

	// Test fields
	if typedLet.Name != "x" {
		t.Errorf("TypedLet.Name = %v, want %v", typedLet.Name, "x")
	}

	if typedLet.Value != value {
		t.Error("TypedLet.Value not set correctly")
	}

	if typedLet.Body != body {
		t.Error("TypedLet.Body not set correctly")
	}

	// Verify it implements TypedNode interface
	var _ TypedNode = typedLet
}

func TestTypedApp(t *testing.T) {
	fn := &TypedVar{
		TypedExpr: TypedExpr{
			NodeID: 1,
			Type:   "(Int, Int) -> Int",
		},
		Name: "add",
	}

	arg1 := &TypedLit{
		TypedExpr: TypedExpr{NodeID: 2, Type: "Int"},
		Kind:      core.IntLit,
		Value:     int64(1),
	}

	arg2 := &TypedLit{
		TypedExpr: TypedExpr{NodeID: 3, Type: "Int"},
		Kind:      core.IntLit,
		Value:     int64(2),
	}

	typedApp := &TypedApp{
		TypedExpr: TypedExpr{
			NodeID: 4,
			Span:   ast.Pos{Line: 1, Column: 1, File: "test.ail"},
			Type:   "Int",
		},
		Func: fn,
		Args: []TypedNode{arg1, arg2},
	}

	// Test fields
	if typedApp.Func != fn {
		t.Error("TypedApp.Func not set correctly")
	}

	if len(typedApp.Args) != 2 {
		t.Errorf("TypedApp.Args length = %v, want %v", len(typedApp.Args), 2)
	}

	// Verify it implements TypedNode interface
	var _ TypedNode = typedApp
}

func TestTypedIf(t *testing.T) {
	cond := &TypedLit{
		TypedExpr: TypedExpr{NodeID: 1, Type: "Bool"},
		Kind:      core.BoolLit,
		Value:     true,
	}

	thenBranch := &TypedLit{
		TypedExpr: TypedExpr{NodeID: 2, Type: "String"},
		Kind:      core.StringLit,
		Value:     "yes",
	}

	elseBranch := &TypedLit{
		TypedExpr: TypedExpr{NodeID: 3, Type: "String"},
		Kind:      core.StringLit,
		Value:     "no",
	}

	typedIf := &TypedIf{
		TypedExpr: TypedExpr{
			NodeID: 4,
			Span:   ast.Pos{Line: 1, Column: 1, File: "test.ail"},
			Type:   "String",
		},
		Cond: cond,
		Then: thenBranch,
		Else: elseBranch,
	}

	// Test fields
	if typedIf.Cond != cond {
		t.Error("TypedIf.Cond not set correctly")
	}

	if typedIf.Then != thenBranch {
		t.Error("TypedIf.Then not set correctly")
	}

	if typedIf.Else != elseBranch {
		t.Error("TypedIf.Else not set correctly")
	}

	// Verify it implements TypedNode interface
	var _ TypedNode = typedIf
}

func TestTypedProgram(t *testing.T) {
	decl1 := &TypedLit{
		TypedExpr: TypedExpr{NodeID: 1, Type: "Int"},
		Kind:      core.IntLit,
		Value:     int64(42),
	}

	decl2 := &TypedVar{
		TypedExpr: TypedExpr{NodeID: 2, Type: "String"},
		Name:      "result",
	}

	typedProgram := &TypedProgram{
		Decls: []TypedNode{decl1, decl2},
	}

	// Test fields
	if len(typedProgram.Decls) != 2 {
		t.Errorf("TypedProgram.Decls length = %v, want %v", len(typedProgram.Decls), 2)
	}

	if typedProgram.Decls[0] != decl1 || typedProgram.Decls[1] != decl2 {
		t.Error("TypedProgram.Decls not set correctly")
	}
}