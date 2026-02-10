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

	// CoreTI: intrinsic itself has type Bool (result), operands have type Float
	coreTI := types.NewCoreTypeInfo()
	coreTI.Set(intrinsic.ID(), types.TBool)          // Result type
	coreTI.Set(intrinsic.Args[0].ID(), types.TFloat) // Operand type (variable b)
	coreTI.Set(intrinsic.Args[1].ID(), types.TFloat) // Operand type (literal 0.0)

	// Create resolved constraint that says this == operation uses Float type
	resolvedConstraints := map[uint64]*types.ResolvedConstraint{
		42: {
			NodeID:    42,
			ClassName: "Eq",
			Type:      types.TFloat,
			Method:    "eq",
		},
		1: { // For the first argument (variable b)
			NodeID:    1,
			ClassName: "Eq",
			Type:      types.TFloat,
			Method:    "eq",
		},
	}

	// Create OpLowerer with CoreTI and resolved constraints
	typeEnv := types.NewTypeEnv()
	lowerer := NewOpLowerer(typeEnv, coreTI)
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
	lowerer := NewOpLowerer(typeEnv, types.NewCoreTypeInfo())
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
	lowerer := NewOpLowerer(typeEnv, types.NewCoreTypeInfo())
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
		{
			"list[Int]", // DX-17: canonical form is lowercase
			&types.TApp{
				Constructor: &types.TCon{Name: "list"},
				Args:        []types.Type{types.TInt},
			},
			"List",
		},
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

// TestOpLowering_ContractExpressions_IntComparison tests that contract expressions
// (requires/ensures) have their Intrinsic nodes lowered through OpLowering.
// This is the core regression test for M-CONTRACTS-OPLOWERING.
func TestOpLowering_ContractExpressions_IntComparison(t *testing.T) {
	// Create a program with a contract: requires { x >= 0 }
	geIntrinsic := &core.Intrinsic{
		CoreNode: core.CoreNode{NodeID: 50},
		Op:       core.OpGe,
		Args: []core.CoreExpr{
			&core.Var{CoreNode: core.CoreNode{NodeID: 51}, Name: "x"},
			&core.Lit{CoreNode: core.CoreNode{NodeID: 52}, Kind: core.IntLit, Value: int64(0)},
		},
	}

	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Var{CoreNode: core.CoreNode{NodeID: 1}, Name: "x"}, // dummy decl
		},
		Meta: map[string]*core.DeclMeta{
			"absolute": {
				Name: "absolute",
				Contracts: []*core.Contract{
					{
						Kind:     core.RequiresKind,
						Expr:     geIntrinsic,
						Message:  "(x >= 0)",
						Location: "test.ail:3",
					},
				},
			},
		},
	}

	coreTI := types.NewCoreTypeInfo()
	coreTI.Set(50, types.TBool)
	coreTI.Set(51, types.TInt)
	coreTI.Set(52, types.TInt)

	resolvedConstraints := map[uint64]*types.ResolvedConstraint{
		50: {NodeID: 50, ClassName: "Ord", Type: types.TInt, Method: "ge"},
	}

	lowerer := NewOpLowerer(types.NewTypeEnv(), coreTI)
	lowerer.SetResolvedConstraints(resolvedConstraints)

	lowered, err := lowerer.Lower(prog)
	if err != nil {
		t.Fatalf("Lower failed: %v", err)
	}

	// Verify contract expression was lowered
	contracts := lowered.Meta["absolute"].Contracts
	if len(contracts) != 1 {
		t.Fatalf("Expected 1 contract, got %d", len(contracts))
	}

	contract := contracts[0]
	if contract.Kind != core.RequiresKind {
		t.Errorf("Expected RequiresKind, got %v", contract.Kind)
	}
	if contract.Message != "(x >= 0)" {
		t.Errorf("Expected message preserved, got %q", contract.Message)
	}

	// The Intrinsic should be lowered to App($builtin.ge_Int, ...)
	app, ok := contract.Expr.(*core.App)
	if !ok {
		t.Fatalf("Expected contract Expr to be App (lowered), got %T", contract.Expr)
	}

	builtinRef, ok := app.Func.(*core.VarGlobal)
	if !ok {
		t.Fatalf("Expected VarGlobal for builtin, got %T", app.Func)
	}

	if builtinRef.Ref.Module != "$builtin" {
		t.Errorf("Expected $builtin module, got %s", builtinRef.Ref.Module)
	}
	if builtinRef.Ref.Name != "ge_Int" {
		t.Errorf("Expected ge_Int builtin, got %s", builtinRef.Ref.Name)
	}
}

// TestOpLowering_ContractExpressions_NoContracts verifies programs without
// contracts pass through Lower() unchanged.
func TestOpLowering_ContractExpressions_NoContracts(t *testing.T) {
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Lit{CoreNode: core.CoreNode{NodeID: 1}, Kind: core.IntLit, Value: int64(42)},
		},
		Meta: map[string]*core.DeclMeta{
			"func1": {Name: "func1", Contracts: nil},
			"func2": {Name: "func2", Contracts: []*core.Contract{}},
		},
	}

	lowerer := NewOpLowerer(types.NewTypeEnv(), types.NewCoreTypeInfo())
	lowered, err := lowerer.Lower(prog)
	if err != nil {
		t.Fatalf("Lower failed: %v", err)
	}

	// Meta should be preserved with both entries
	if len(lowered.Meta) != 2 {
		t.Fatalf("Expected 2 meta entries, got %d", len(lowered.Meta))
	}
	if lowered.Meta["func1"].Name != "func1" {
		t.Errorf("Expected func1 metadata preserved")
	}
}

// TestOpLowering_ContractExpressions_NilExpr verifies that a contract
// with nil Expr is handled gracefully (no panic).
func TestOpLowering_ContractExpressions_NilExpr(t *testing.T) {
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Lit{CoreNode: core.CoreNode{NodeID: 1}, Kind: core.IntLit, Value: int64(1)},
		},
		Meta: map[string]*core.DeclMeta{
			"f": {
				Name: "f",
				Contracts: []*core.Contract{
					{Kind: core.RequiresKind, Expr: nil, Message: "empty"},
				},
			},
		},
	}

	lowerer := NewOpLowerer(types.NewTypeEnv(), types.NewCoreTypeInfo())
	lowered, err := lowerer.Lower(prog)
	if err != nil {
		t.Fatalf("Lower failed: %v", err)
	}

	contracts := lowered.Meta["f"].Contracts
	if len(contracts) != 1 {
		t.Fatalf("Expected 1 contract, got %d", len(contracts))
	}
	if contracts[0].Expr != nil {
		t.Errorf("Expected nil Expr preserved, got %T", contracts[0].Expr)
	}
}

// TestOpLowering_ContractExpressions_DoesNotMutateOriginal verifies that
// lowering contracts does not mutate the original program.
func TestOpLowering_ContractExpressions_DoesNotMutateOriginal(t *testing.T) {
	geIntrinsic := &core.Intrinsic{
		CoreNode: core.CoreNode{NodeID: 60},
		Op:       core.OpGe,
		Args: []core.CoreExpr{
			&core.Var{CoreNode: core.CoreNode{NodeID: 61}, Name: "x"},
			&core.Lit{CoreNode: core.CoreNode{NodeID: 62}, Kind: core.IntLit, Value: int64(0)},
		},
	}

	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Lit{CoreNode: core.CoreNode{NodeID: 1}, Kind: core.IntLit, Value: int64(1)},
		},
		Meta: map[string]*core.DeclMeta{
			"f": {
				Name: "f",
				Contracts: []*core.Contract{
					{Kind: core.RequiresKind, Expr: geIntrinsic, Message: "test"},
				},
			},
		},
	}

	coreTI := types.NewCoreTypeInfo()
	coreTI.Set(60, types.TBool)
	coreTI.Set(61, types.TInt)
	coreTI.Set(62, types.TInt)

	lowerer := NewOpLowerer(types.NewTypeEnv(), coreTI)
	lowerer.SetResolvedConstraints(map[uint64]*types.ResolvedConstraint{
		60: {NodeID: 60, ClassName: "Ord", Type: types.TInt, Method: "ge"},
	})

	_, err := lowerer.Lower(prog)
	if err != nil {
		t.Fatalf("Lower failed: %v", err)
	}

	// Original program should still have Intrinsic (not mutated)
	originalExpr := prog.Meta["f"].Contracts[0].Expr
	if _, ok := originalExpr.(*core.Intrinsic); !ok {
		t.Errorf("Original program was mutated! Expected Intrinsic, got %T", originalExpr)
	}
}

// TestOpLowering_Concat tests that concat operations are correctly
// lowered to concat_String or concat_List based on operand types.
// This locks in the behavior for the ++ operator.
func TestOpLowering_Concat(t *testing.T) {
	tests := []struct {
		name            string
		leftArg         core.CoreExpr
		rightArg        core.CoreExpr
		expectedBuiltin string
	}{
		{
			name: "string concatenation",
			leftArg: &core.Var{
				CoreNode: core.CoreNode{NodeID: 1},
				Name:     "$tmp1",
			},
			rightArg: &core.Var{
				CoreNode: core.CoreNode{NodeID: 2},
				Name:     "$tmp2",
			},
			expectedBuiltin: "concat_String",
		},
		{
			name: "list concatenation",
			leftArg: &core.Var{
				CoreNode: core.CoreNode{NodeID: 3},
				Name:     "$tmp3",
			},
			rightArg: &core.Var{
				CoreNode: core.CoreNode{NodeID: 4},
				Name:     "$tmp4",
			},
			expectedBuiltin: "concat_List",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create concat intrinsic
			intrinsic := &core.Intrinsic{
				CoreNode: core.CoreNode{NodeID: 100},
				Op:       core.OpConcat,
				Args:     []core.CoreExpr{tt.leftArg, tt.rightArg},
			}

			// Create lowerer with CoreTI populated
			typeEnv := types.NewTypeEnv()
			coreTI := types.NewCoreTypeInfo()

			// Populate CoreTI with the type of the concat intrinsic
			// The type of ++ depends on what it's concatenating
			if tt.expectedBuiltin == "concat_String" {
				// For string concatenation, the intrinsic returns string
				coreTI.Set(intrinsic.ID(), types.TString)
			} else {
				// For list concatenation, the intrinsic returns list[int]
				// DX-17: canonical form is lowercase "list"
				listType := &types.TApp{
					Constructor: &types.TCon{Name: "list"},
					Args:        []types.Type{types.TInt},
				}
				coreTI.Set(intrinsic.ID(), listType)
			}

			lowerer := NewOpLowerer(typeEnv, coreTI)

			// Lower the intrinsic
			lowered := lowerer.lowerExpr(intrinsic)

			// Verify it was lowered to an App node
			app, ok := lowered.(*core.App)
			if !ok {
				t.Fatalf("Expected App node, got %T", lowered)
			}

			// Verify the function is a builtin reference to the expected concat variant
			builtinRef, ok := app.Func.(*core.VarGlobal)
			if !ok {
				t.Fatalf("Expected VarGlobal for builtin, got %T", app.Func)
			}

			if builtinRef.Ref.Name != tt.expectedBuiltin {
				t.Errorf("Expected %s builtin, got %s", tt.expectedBuiltin, builtinRef.Ref.Name)
			}
		})
	}
}
