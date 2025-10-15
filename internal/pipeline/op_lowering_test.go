package pipeline

import (
	"testing"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// TestOpLowering_FloatEquality tests that float equality operations
// are correctly lowered to eq_Float instead of eq_Int when variables are involved.
// This is a regression test for the bug where `let b: float = 0.0; b == 0.0`
// would incorrectly call eq_Int instead of eq_Float.
func TestOpLowering_FloatEquality(t *testing.T) {
	// Create an intrinsic == operation with two float arguments
	intrinsic := &core.Intrinsic{
		CoreNode: core.CoreNode{NodeID: 42},
		Op:       core.OpEq,
		Args: []core.CoreExpr{
			&core.Var{CoreNode: core.CoreNode{NodeID: 1}, Name: "b"},
			&core.Lit{CoreNode: core.CoreNode{NodeID: 2}, Kind: core.FloatLit, Value: 0.0},
		},
	}

	// Create resolved constraint that says this == operation uses Float type
	resolvedConstraints := map[uint64]*types.ResolvedConstraint{
		42: {
			NodeID:    42,
			ClassName: "Eq",
			Type:      types.TFloat,
			Method:    "eq",
		},
	}

	// Create OpLowerer with resolved constraints
	typeEnv := types.NewTypeEnv()
	lowerer := NewOpLowerer(typeEnv)
	lowerer.SetResolvedConstraints(resolvedConstraints)

	// Lower the intrinsic
	lowered := lowerer.lowerExpr(intrinsic)

	// Verify it was lowered to an App node
	app, ok := lowered.(*core.App)
	if !ok {
		t.Fatalf("Expected App node, got %T", lowered)
	}

	// Verify the function is a builtin reference to eq_Float
	builtinRef, ok := app.Func.(*core.VarGlobal)
	if !ok {
		t.Fatalf("Expected VarGlobal for builtin, got %T", app.Func)
	}

	if builtinRef.Ref.Module != "$builtin" {
		t.Errorf("Expected $builtin module, got %s", builtinRef.Ref.Module)
	}

	if builtinRef.Ref.Name != "eq_Float" {
		t.Errorf("Expected eq_Float builtin, got %s (REGRESSION: should use Float, not Int)", builtinRef.Ref.Name)
	}
}

// TestOpLowering_IntEquality verifies that integer equality still works correctly
func TestOpLowering_IntEquality(t *testing.T) {
	intrinsic := &core.Intrinsic{
		CoreNode: core.CoreNode{NodeID: 100},
		Op:       core.OpEq,
		Args: []core.CoreExpr{
			&core.Var{CoreNode: core.CoreNode{NodeID: 3}, Name: "a"},
			&core.Lit{CoreNode: core.CoreNode{NodeID: 4}, Kind: core.IntLit, Value: int64(0)},
		},
	}

	resolvedConstraints := map[uint64]*types.ResolvedConstraint{
		100: {
			NodeID:    100,
			ClassName: "Eq",
			Type:      types.TInt,
			Method:    "eq",
		},
	}

	typeEnv := types.NewTypeEnv()
	lowerer := NewOpLowerer(typeEnv)
	lowerer.SetResolvedConstraints(resolvedConstraints)

	lowered := lowerer.lowerExpr(intrinsic)

	app, ok := lowered.(*core.App)
	if !ok {
		t.Fatalf("Expected App node, got %T", lowered)
	}

	builtinRef, ok := app.Func.(*core.VarGlobal)
	if !ok {
		t.Fatalf("Expected VarGlobal for builtin, got %T", app.Func)
	}

	if builtinRef.Ref.Name != "eq_Int" {
		t.Errorf("Expected eq_Int builtin, got %s", builtinRef.Ref.Name)
	}
}

// TestOpLowering_FallbackToHeuristics tests that when no constraint is available,
// the lowerer falls back to heuristics (e.g., for OpNot, OpConcat)
func TestOpLowering_FallbackToHeuristics(t *testing.T) {
	intrinsic := &core.Intrinsic{
		CoreNode: core.CoreNode{NodeID: 200},
		Op:       core.OpNot,
		Args: []core.CoreExpr{
			&core.Var{CoreNode: core.CoreNode{NodeID: 5}, Name: "flag"},
		},
	}

	// No resolved constraints - should fall back to heuristics
	typeEnv := types.NewTypeEnv()
	lowerer := NewOpLowerer(typeEnv)
	lowerer.SetResolvedConstraints(map[uint64]*types.ResolvedConstraint{})

	lowered := lowerer.lowerExpr(intrinsic)

	app, ok := lowered.(*core.App)
	if !ok {
		t.Fatalf("Expected App node, got %T", lowered)
	}

	builtinRef, ok := app.Func.(*core.VarGlobal)
	if !ok {
		t.Fatalf("Expected VarGlobal for builtin, got %T", app.Func)
	}

	// OpNot should default to Bool
	if builtinRef.Ref.Name != "not_Bool" {
		t.Errorf("Expected not_Bool builtin, got %s", builtinRef.Ref.Name)
	}
}

// TestGetTypeSuffixFromType verifies the type to suffix mapping
func TestGetTypeSuffixFromType(t *testing.T) {
	tests := []struct {
		name     string
		typ      types.Type
		expected string
	}{
		{"TInt", types.TInt, "Int"},
		{"TFloat", types.TFloat, "Float"},
		{"TBool", types.TBool, "Bool"},
		{"TString", types.TString, "String"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getTypeSuffixFromType(tt.typ)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}
