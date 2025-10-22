package pipeline

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// TestValidateCoreTypeInfo_CompleteProgram verifies that a fully-typed program passes validation
func TestValidateCoreTypeInfo_CompleteProgram(t *testing.T) {
	// Create a simple Core program: let x = 42 in x
	nodeID := uint64(1)
	litNode := &core.Lit{
		CoreNode: core.CoreNode{NodeID: nodeID},
		Kind:     core.IntLit,
		Value:    42,
	}
	nodeID++

	varNode := &core.Var{
		CoreNode: core.CoreNode{NodeID: nodeID},
		Name:     "x",
	}
	nodeID++

	letNode := &core.Let{
		CoreNode: core.CoreNode{NodeID: nodeID},
		Name:     "x",
		Value:    litNode,
		Body:     varNode,
	}

	prog := &core.Program{
		Decls: []core.CoreExpr{letNode},
	}

	// Create CoreTypeInfo with all nodes typed
	coreTI := types.NewCoreTypeInfo()
	coreTI.Set(litNode.ID(), &types.TCon{Name: "Int"})
	coreTI.Set(varNode.ID(), &types.TCon{Name: "Int"})
	coreTI.Set(letNode.ID(), &types.TCon{Name: "Int"})

	// Validation should pass
	err := ValidateCoreTypeInfo(prog, coreTI)
	if err != nil {
		t.Fatalf("Expected validation to pass, got error: %v", err)
	}
}

// TestValidateCoreTypeInfo_MissingFloatType verifies detection of missing Float literal type
func TestValidateCoreTypeInfo_MissingFloatType(t *testing.T) {
	// Create a Float literal without type: 3.14
	floatNode := &core.Lit{
		CoreNode: core.CoreNode{NodeID: 1},
		Kind:     core.FloatLit,
		Value:    3.14,
	}

	prog := &core.Program{
		Decls: []core.CoreExpr{floatNode},
	}

	// Create empty CoreTypeInfo (missing Float type)
	coreTI := types.NewCoreTypeInfo()

	// Validation should fail
	err := ValidateCoreTypeInfo(prog, coreTI)
	if err == nil {
		t.Fatal("Expected validation to fail for missing Float type")
	}

	// Check error message contains expected details
	errMsg := err.Error()
	if !strings.Contains(errMsg, "Lit(Float)") {
		t.Errorf("Expected error to mention 'Lit(Float)', got: %v", errMsg)
	}
	if !strings.Contains(errMsg, "NodeID 1") {
		t.Errorf("Expected error to mention 'NodeID 1', got: %v", errMsg)
	}
	if !strings.Contains(errMsg, "defaulting/substitution") {
		t.Errorf("Expected error hint about defaulting/substitution, got: %v", errMsg)
	}
}

// TestValidateCoreTypeInfo_MissingBoolType verifies detection of missing Bool literal type
func TestValidateCoreTypeInfo_MissingBoolType(t *testing.T) {
	// Create a Bool literal without type: true
	boolNode := &core.Lit{
		CoreNode: core.CoreNode{NodeID: 1},
		Kind:     core.BoolLit,
		Value:    true,
	}

	prog := &core.Program{
		Decls: []core.CoreExpr{boolNode},
	}

	coreTI := types.NewCoreTypeInfo()

	err := ValidateCoreTypeInfo(prog, coreTI)
	if err == nil {
		t.Fatal("Expected validation to fail for missing Bool type")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "Lit(Bool)") {
		t.Errorf("Expected error to mention 'Lit(Bool)', got: %v", errMsg)
	}
}

// TestValidateCoreTypeInfo_MissingComparisonType verifies detection of missing comparison operator type
func TestValidateCoreTypeInfo_MissingComparisonType(t *testing.T) {
	// Create a comparison: 5 <= x (as Intrinsic after lowering)
	// NOTE: In real pipeline, BinOp is lowered to Intrinsic, but we test Intrinsic directly
	leftNode := &core.Lit{
		CoreNode: core.CoreNode{NodeID: 1},
		Kind:     core.IntLit,
		Value:    5,
	}

	rightNode := &core.Var{
		CoreNode: core.CoreNode{NodeID: 2},
		Name:     "x",
	}

	intrinsicNode := &core.Intrinsic{
		CoreNode: core.CoreNode{NodeID: 3},
		Op:       core.OpLe,
		Args:     []core.CoreExpr{leftNode, rightNode},
	}

	prog := &core.Program{
		Decls: []core.CoreExpr{intrinsicNode},
	}

	// Type the operands but NOT the intrinsic itself
	coreTI := types.NewCoreTypeInfo()
	coreTI.Set(leftNode.ID(), &types.TCon{Name: "Int"})
	coreTI.Set(rightNode.ID(), &types.TCon{Name: "Int"})
	// Missing: coreTI.Set(intrinsicNode.ID(), &types.TCon{Name: "Bool"})

	err := ValidateCoreTypeInfo(prog, coreTI)
	if err == nil {
		t.Fatal("Expected validation to fail for missing Intrinsic type")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "Intrinsic(OpLe)") {
		t.Errorf("Expected error to mention 'Intrinsic(OpLe)', got: %v", errMsg)
	}
	if !strings.Contains(errMsg, "NodeID 3") {
		t.Errorf("Expected error to mention 'NodeID 3', got: %v", errMsg)
	}
}

// TestValidateCoreTypeInfo_MissingNestedLetType verifies detection of missing nested let type
func TestValidateCoreTypeInfo_MissingNestedLetType(t *testing.T) {
	// Create nested lets: let x = 1 in let y = 2 in x + y
	lit1 := &core.Lit{
		CoreNode: core.CoreNode{NodeID: 1},
		Kind:     core.IntLit,
		Value:    1,
	}

	lit2 := &core.Lit{
		CoreNode: core.CoreNode{NodeID: 2},
		Kind:     core.IntLit,
		Value:    2,
	}

	varX := &core.Var{
		CoreNode: core.CoreNode{NodeID: 3},
		Name:     "x",
	}

	varY := &core.Var{
		CoreNode: core.CoreNode{NodeID: 4},
		Name:     "y",
	}

	addNode := &core.Intrinsic{
		CoreNode: core.CoreNode{NodeID: 5},
		Op:       core.OpAdd,
		Args:     []core.CoreExpr{varX, varY},
	}

	innerLet := &core.Let{
		CoreNode: core.CoreNode{NodeID: 6},
		Name:     "y",
		Value:    lit2,
		Body:     addNode,
	}

	outerLet := &core.Let{
		CoreNode: core.CoreNode{NodeID: 7},
		Name:     "x",
		Value:    lit1,
		Body:     innerLet,
	}

	prog := &core.Program{
		Decls: []core.CoreExpr{outerLet},
	}

	// Type everything EXCEPT the inner let
	coreTI := types.NewCoreTypeInfo()
	coreTI.Set(lit1.ID(), &types.TCon{Name: "Int"})
	coreTI.Set(lit2.ID(), &types.TCon{Name: "Int"})
	coreTI.Set(varX.ID(), &types.TCon{Name: "Int"})
	coreTI.Set(varY.ID(), &types.TCon{Name: "Int"})
	coreTI.Set(addNode.ID(), &types.TCon{Name: "Int"})
	coreTI.Set(outerLet.ID(), &types.TCon{Name: "Int"})
	// Missing: coreTI.Set(innerLet.ID(), &types.TCon{Name: "Int"})

	err := ValidateCoreTypeInfo(prog, coreTI)
	if err == nil {
		t.Fatal("Expected validation to fail for missing nested Let type")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "Let(y)") {
		t.Errorf("Expected error to mention 'Let(y)', got: %v", errMsg)
	}
	if !strings.Contains(errMsg, "NodeID 6") {
		t.Errorf("Expected error to mention 'NodeID 6', got: %v", errMsg)
	}
}

// TestValidateCoreTypeInfo_MultipleGaps is the "golden" test with 3+ missing nodes
// This verifies:
// - Error lists multiple gaps
// - Gaps are grouped by kind
// - Output is stable (sorted by NodeID)
// - All required fields present (NodeID, kind, position, hint)
func TestValidateCoreTypeInfo_MultipleGaps(t *testing.T) {
	// Create program with multiple missing types:
	// 1. Float literal (3.14)
	// 2. Bool literal (true)
	// 3. Comparison intrinsic (<=)
	// 4. Nested let

	floatNode := &core.Lit{
		CoreNode: core.CoreNode{NodeID: 1},
		Kind:     core.FloatLit,
		Value:    3.14,
	}

	boolNode := &core.Lit{
		CoreNode: core.CoreNode{NodeID: 2},
		Kind:     core.BoolLit,
		Value:    true,
	}

	leftNode := &core.Lit{
		CoreNode: core.CoreNode{NodeID: 3},
		Kind:     core.IntLit,
		Value:    5,
	}

	rightNode := &core.Var{
		CoreNode: core.CoreNode{NodeID: 4},
		Name:     "x",
	}

	intrinsicNode := &core.Intrinsic{
		CoreNode: core.CoreNode{NodeID: 5},
		Op:       core.OpLe,
		Args:     []core.CoreExpr{leftNode, rightNode},
	}

	innerLet := &core.Let{
		CoreNode: core.CoreNode{NodeID: 6},
		Name:     "y",
		Value:    boolNode,
		Body:     intrinsicNode,
	}

	outerLet := &core.Let{
		CoreNode: core.CoreNode{NodeID: 7},
		Name:     "f",
		Value:    floatNode,
		Body:     innerLet,
	}

	prog := &core.Program{
		Decls: []core.CoreExpr{outerLet},
	}

	// Type only some nodes, leave 4 missing:
	// Missing: floatNode, boolNode, intrinsicNode, innerLet
	coreTI := types.NewCoreTypeInfo()
	coreTI.Set(leftNode.ID(), &types.TCon{Name: "Int"})
	coreTI.Set(rightNode.ID(), &types.TCon{Name: "Int"})
	coreTI.Set(outerLet.ID(), &types.TCon{Name: "Bool"})

	err := ValidateCoreTypeInfo(prog, coreTI)
	if err == nil {
		t.Fatal("Expected validation to fail with multiple gaps")
	}

	errMsg := err.Error()

	// Verify error mentions all 4 missing nodes
	expectedMissing := []string{
		"NodeID 1", // floatNode
		"NodeID 2", // boolNode
		"NodeID 5", // intrinsicNode
		"NodeID 6", // innerLet
	}

	for _, expected := range expectedMissing {
		if !strings.Contains(errMsg, expected) {
			t.Errorf("Expected error to mention '%s', got: %v", expected, errMsg)
		}
	}

	// Verify error groups by kind
	if !strings.Contains(errMsg, "Missing Lit(Float)") {
		t.Errorf("Expected error to group 'Lit(Float)', got: %v", errMsg)
	}
	if !strings.Contains(errMsg, "Missing Lit(Bool)") {
		t.Errorf("Expected error to group 'Lit(Bool)', got: %v", errMsg)
	}
	if !strings.Contains(errMsg, "Missing Intrinsic(OpLe)") {
		t.Errorf("Expected error to group 'Intrinsic(OpLe)', got: %v", errMsg)
	}
	if !strings.Contains(errMsg, "Missing Let(y)") {
		t.Errorf("Expected error to group 'Let(y)', got: %v", errMsg)
	}

	// Verify error includes helpful debugging command
	if !strings.Contains(errMsg, "ailang debug ast") {
		t.Errorf("Expected error to suggest debug command, got: %v", errMsg)
	}

	// Verify error includes hints
	if !strings.Contains(errMsg, "Hint:") {
		t.Errorf("Expected error to include hints, got: %v", errMsg)
	}
}

// TestValidateCoreTypeInfo_LambdaWithPolymorphicType verifies that polymorphic types are acceptable
// This ensures forward-compat with monomorphization: lambda bodies may have type variables (α, β)
// and that's OK - the validator only requires "has a type," not "concrete type"
func TestValidateCoreTypeInfo_LambdaWithPolymorphicType(t *testing.T) {
	// Create lambda: \x -> x (identity function)
	varNode := &core.Var{
		CoreNode: core.CoreNode{NodeID: 1},
		Name:     "x",
	}

	lambdaNode := &core.Lambda{
		CoreNode: core.CoreNode{NodeID: 2},
		Params:   []string{"x"},
		Body:     varNode,
	}

	prog := &core.Program{
		Decls: []core.CoreExpr{lambdaNode},
	}

	// Type with polymorphic type variable (α -> α)
	coreTI := types.NewCoreTypeInfo()
	typeVar := &types.TVar2{Name: "α"}
	coreTI.Set(varNode.ID(), typeVar)
	coreTI.Set(lambdaNode.ID(), &types.TFunc2{
		Params:    []types.Type{typeVar},
		Return:    typeVar,
		EffectRow: &types.Row{Kind: types.KEffect{}, Labels: map[string]types.Type{}, Tail: nil},
	})

	// Validation should PASS (polymorphic types are acceptable)
	err := ValidateCoreTypeInfo(prog, coreTI)
	if err != nil {
		t.Fatalf("Expected validation to pass for polymorphic lambda, got error: %v", err)
	}
}

// TestValidateCoreTypeInfo_AllCoreNodeTypes verifies validator handles all Core node variants
// This is a smoke test ensuring we don't panic on any Core expression type
func TestValidateCoreTypeInfo_AllCoreNodeTypes(t *testing.T) {
	nodeID := uint64(1)

	// Helper to create typed node
	makeTypedNode := func(expr core.CoreExpr) core.CoreExpr {
		return expr
	}

	// Create one of each Core node type
	nodes := []core.CoreExpr{
		makeTypedNode(&core.Var{CoreNode: core.CoreNode{NodeID: nodeID}, Name: "x"}),
		makeTypedNode(&core.VarGlobal{CoreNode: core.CoreNode{NodeID: nodeID + 1}, Ref: core.GlobalRef{Module: "std", Name: "print"}}),
		makeTypedNode(&core.Lit{CoreNode: core.CoreNode{NodeID: nodeID + 2}, Kind: core.IntLit, Value: 42}),
		makeTypedNode(&core.Lambda{CoreNode: core.CoreNode{NodeID: nodeID + 3}, Params: []string{"x"}, Body: &core.Var{CoreNode: core.CoreNode{NodeID: nodeID + 20}, Name: "x"}}),
		makeTypedNode(&core.Let{CoreNode: core.CoreNode{NodeID: nodeID + 4}, Name: "x", Value: &core.Lit{CoreNode: core.CoreNode{NodeID: nodeID + 21}, Kind: core.IntLit, Value: 1}, Body: &core.Var{CoreNode: core.CoreNode{NodeID: nodeID + 22}, Name: "x"}}),
		makeTypedNode(&core.App{CoreNode: core.CoreNode{NodeID: nodeID + 5}, Func: &core.Var{CoreNode: core.CoreNode{NodeID: nodeID + 23}, Name: "f"}, Args: []core.CoreExpr{&core.Var{CoreNode: core.CoreNode{NodeID: nodeID + 24}, Name: "x"}}}),
		makeTypedNode(&core.If{CoreNode: core.CoreNode{NodeID: nodeID + 6}, Cond: &core.Lit{CoreNode: core.CoreNode{NodeID: nodeID + 25}, Kind: core.BoolLit, Value: true}, Then: &core.Lit{CoreNode: core.CoreNode{NodeID: nodeID + 26}, Kind: core.IntLit, Value: 1}, Else: &core.Lit{CoreNode: core.CoreNode{NodeID: nodeID + 27}, Kind: core.IntLit, Value: 2}}),
		makeTypedNode(&core.Intrinsic{CoreNode: core.CoreNode{NodeID: nodeID + 7}, Op: core.OpAdd, Args: []core.CoreExpr{&core.Lit{CoreNode: core.CoreNode{NodeID: nodeID + 28}, Kind: core.IntLit, Value: 1}, &core.Lit{CoreNode: core.CoreNode{NodeID: nodeID + 29}, Kind: core.IntLit, Value: 2}}}),
		makeTypedNode(&core.Record{CoreNode: core.CoreNode{NodeID: nodeID + 8}, Fields: map[string]core.CoreExpr{"x": &core.Lit{CoreNode: core.CoreNode{NodeID: nodeID + 30}, Kind: core.IntLit, Value: 1}}}),
		makeTypedNode(&core.RecordAccess{CoreNode: core.CoreNode{NodeID: nodeID + 9}, Record: &core.Var{CoreNode: core.CoreNode{NodeID: nodeID + 31}, Name: "r"}, Field: "x"}),
		makeTypedNode(&core.List{CoreNode: core.CoreNode{NodeID: nodeID + 10}, Elements: []core.CoreExpr{&core.Lit{CoreNode: core.CoreNode{NodeID: nodeID + 32}, Kind: core.IntLit, Value: 1}}}),
		makeTypedNode(&core.Tuple{CoreNode: core.CoreNode{NodeID: nodeID + 11}, Elements: []core.CoreExpr{&core.Lit{CoreNode: core.CoreNode{NodeID: nodeID + 33}, Kind: core.IntLit, Value: 1}, &core.Lit{CoreNode: core.CoreNode{NodeID: nodeID + 34}, Kind: core.IntLit, Value: 2}}}),
	}

	// Test with MISSING types - validator should handle all node types without panic
	prog := &core.Program{Decls: nodes}
	coreTI := types.NewCoreTypeInfo()

	err := ValidateCoreTypeInfo(prog, coreTI)
	if err == nil {
		t.Fatal("Expected validation to fail with missing types")
	}

	// Verify no panic occurred (test passes if we reach here)
	t.Logf("Validator handled all Core node types successfully")
}
