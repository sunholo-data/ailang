package pipeline

import (
	"testing"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// TestComparisonInLambdaBodies tests that comparison operators use operand types (not result type Bool)
// This is a regression test for the bug where comparison operators would fail with
// "Operator '>' has no implementation for type Bool" in lambda bodies.
//
// The bug: For `x > 0`, the intrinsic node has type Bool (the result), but lowering
// needs to look at the operand type (Int) to choose gt_Int (not gt_Bool which doesn't exist).
func TestComparisonWithIntOperands(t *testing.T) {
	// Create intrinsic: x > 0 (where x is Int)
	intrinsic := &core.Intrinsic{
		CoreNode: core.CoreNode{NodeID: 42},
		Op:       core.OpGt,
		Args: []core.CoreExpr{
			&core.Var{CoreNode: core.CoreNode{NodeID: 1}, Name: "x"}, // Int
			&core.Lit{CoreNode: core.CoreNode{NodeID: 2}, Kind: core.IntLit, Value: int64(0)},
		},
	}

	// CoreTI: intrinsic itself has type Bool (the result), operands have type Int
	coreTI := types.NewCoreTypeInfo()
	coreTI.Set(intrinsic.ID(), types.TBool)        // Result type
	coreTI.Set(intrinsic.Args[0].ID(), types.TInt) // Operand type
	coreTI.Set(intrinsic.Args[1].ID(), types.TInt)

	// Resolved constraint for the Ord class
	resolvedConstraints := map[uint64]*types.ResolvedConstraint{
		42: {
			NodeID:    42,
			ClassName: "Ord",
			Type:      types.TInt, // Operand type
			Method:    "gt",
		},
	}

	// Create lowerer with CoreTI
	typeEnv := types.NewTypeEnv()
	lowerer := NewOpLowerer(typeEnv, coreTI)
	lowerer.SetResolvedConstraints(resolvedConstraints)

	// Lower the intrinsic - should use operand type (Int), not result type (Bool)
	lowered := lowerer.lowerExpr(intrinsic)

	// Verify it was lowered to an App node
	app, ok := lowered.(*core.App)
	if !ok {
		t.Fatalf("Expected App node, got %T", lowered)
	}

	// Verify the function is a builtin reference to gt_Int (NOT gt_Bool)
	builtinRef, ok := app.Func.(*core.VarGlobal)
	if !ok {
		t.Fatalf("Expected VarGlobal for builtin, got %T", app.Func)
	}

	if builtinRef.Ref.Module != "$builtin" {
		t.Errorf("Expected $builtin module, got %s", builtinRef.Ref.Module)
	}

	if builtinRef.Ref.Name != "gt_Int" {
		t.Errorf("Expected gt_Int builtin, got %s (BUG: using result type Bool instead of operand type Int)", builtinRef.Ref.Name)
	}
}

// TestComparisonWithFloatOperands tests comparison with Float operands
func TestComparisonWithFloatOperands(t *testing.T) {
	// Create intrinsic: x < 0.0 (where x is Float)
	intrinsic := &core.Intrinsic{
		CoreNode: core.CoreNode{NodeID: 100},
		Op:       core.OpLt,
		Args: []core.CoreExpr{
			&core.Var{CoreNode: core.CoreNode{NodeID: 10}, Name: "x"},
			&core.Lit{CoreNode: core.CoreNode{NodeID: 11}, Kind: core.FloatLit, Value: 0.0},
		},
	}

	// CoreTI: result is Bool, operands are Float
	coreTI := types.NewCoreTypeInfo()
	coreTI.Set(intrinsic.ID(), types.TBool)
	coreTI.Set(intrinsic.Args[0].ID(), types.TFloat)
	coreTI.Set(intrinsic.Args[1].ID(), types.TFloat)

	resolvedConstraints := map[uint64]*types.ResolvedConstraint{
		100: {
			NodeID:    100,
			ClassName: "Ord",
			Type:      types.TFloat,
			Method:    "lt",
		},
	}

	typeEnv := types.NewTypeEnv()
	lowerer := NewOpLowerer(typeEnv, coreTI)
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

	if builtinRef.Ref.Name != "lt_Float" {
		t.Errorf("Expected lt_Float builtin, got %s", builtinRef.Ref.Name)
	}
}

// TestAllComparisonOperators tests all six comparison operators
func TestAllComparisonOperators(t *testing.T) {
	operators := []struct {
		name string
		op   core.IntrinsicOp
		want string // Expected builtin name
	}{
		{"less than", core.OpLt, "lt_Int"},
		{"less or equal", core.OpLe, "le_Int"},
		{"greater than", core.OpGt, "gt_Int"},
		{"greater or equal", core.OpGe, "ge_Int"},
		{"equality", core.OpEq, "eq_Int"},
		{"inequality", core.OpNe, "ne_Int"},
	}

	for _, tt := range operators {
		t.Run(tt.name, func(t *testing.T) {
			intrinsic := &core.Intrinsic{
				CoreNode: core.CoreNode{NodeID: 200},
				Op:       tt.op,
				Args: []core.CoreExpr{
					&core.Var{CoreNode: core.CoreNode{NodeID: 20}, Name: "a"},
					&core.Var{CoreNode: core.CoreNode{NodeID: 21}, Name: "b"},
				},
			}

			// Result is Bool, operands are Int
			coreTI := types.NewCoreTypeInfo()
			coreTI.Set(intrinsic.ID(), types.TBool)
			coreTI.Set(intrinsic.Args[0].ID(), types.TInt)
			coreTI.Set(intrinsic.Args[1].ID(), types.TInt)

			// Determine constraint class based on operator
			className := "Ord"
			if tt.op == core.OpEq || tt.op == core.OpNe {
				className = "Eq"
			}

			resolvedConstraints := map[uint64]*types.ResolvedConstraint{
				200: {
					NodeID:    200,
					ClassName: className,
					Type:      types.TInt,
					Method:    getMethodName(tt.op),
				},
			}

			typeEnv := types.NewTypeEnv()
			lowerer := NewOpLowerer(typeEnv, coreTI)
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

			if builtinRef.Ref.Name != tt.want {
				t.Errorf("Expected %s builtin, got %s", tt.want, builtinRef.Ref.Name)
			}
		})
	}
}

// getMethodName returns the method name for an operator
func getMethodName(op core.IntrinsicOp) string {
	switch op {
	case core.OpLt:
		return "lt"
	case core.OpLe:
		return "lte"
	case core.OpGt:
		return "gt"
	case core.OpGe:
		return "gte"
	case core.OpEq:
		return "eq"
	case core.OpNe:
		return "neq"
	default:
		return ""
	}
}

// TestIsComparisonOrEqualityOp tests the helper function
func TestIsComparisonOrEqualityOp(t *testing.T) {
	tests := []struct {
		op   core.IntrinsicOp
		want bool
	}{
		{core.OpLt, true},
		{core.OpLe, true},
		{core.OpGt, true},
		{core.OpGe, true},
		{core.OpEq, true},
		{core.OpNe, true},
		{core.OpAdd, false},
		{core.OpSub, false},
		{core.OpMul, false},
		{core.OpConcat, false},
		{core.OpAnd, false},
		{core.OpOr, false},
	}

	for _, tt := range tests {
		got := isComparisonOrEqualityOp(tt.op)
		if got != tt.want {
			t.Errorf("isComparisonOrEqualityOp(%v) = %v, want %v", tt.op, got, tt.want)
		}
	}
}
