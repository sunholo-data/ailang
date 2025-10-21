package pipeline

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// TestConcatOperator_TypeGuidedLowering verifies that the ++ operator
// correctly uses type-guided lowering without ANF guessing
func TestConcatOperator_TypeGuidedLowering(t *testing.T) {
	tests := []struct {
		name            string
		inferredType    types.Type
		expectedBuiltin string
	}{
		{
			name:            "string concatenation",
			inferredType:    types.TString,
			expectedBuiltin: "concat_String",
		},
		{
			name: "list concatenation",
			inferredType: &types.TApp{
				Constructor: &types.TCon{Name: "List"},
				Args:        []types.Type{types.TInt},
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
				Args: []core.CoreExpr{
					&core.Var{CoreNode: core.CoreNode{NodeID: 1}, Name: "left"},
					&core.Var{CoreNode: core.CoreNode{NodeID: 2}, Name: "right"},
				},
			}

			// Create lowerer with CoreTI populated (type-guided approach)
			typeEnv := types.NewTypeEnv()
			coreTI := types.NewCoreTypeInfo()
			coreTI.Set(intrinsic.ID(), tt.inferredType)

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

// TestOpLowering_TypeMismatchError verifies that helpful error messages
// are generated when operator types don't match
func TestOpLowering_TypeMismatchError(t *testing.T) {
	// Create an intrinsic with an unsupported operator-type combination
	intrinsic := &core.Intrinsic{
		CoreNode: core.CoreNode{NodeID: 100},
		Op:       core.OpAdd, // Addition doesn't work on strings
		Args: []core.CoreExpr{
			&core.Var{CoreNode: core.CoreNode{NodeID: 1}, Name: "left"},
			&core.Var{CoreNode: core.CoreNode{NodeID: 2}, Name: "right"},
		},
	}

	// Create lowerer with CoreTI indicating string type
	typeEnv := types.NewTypeEnv()
	coreTI := types.NewCoreTypeInfo()
	coreTI.Set(intrinsic.ID(), types.TString) // String type, but operator is +

	lowerer := NewOpLowerer(typeEnv, coreTI)

	// Lower the intrinsic (this should add an error)
	_ = lowerer.lowerExpr(intrinsic)

	// Verify that an error was added
	if len(lowerer.errors) == 0 {
		t.Errorf("Expected error for unsupported operator-type combination, got none")
	}

	// Verify error message is helpful
	if len(lowerer.errors) > 0 {
		errMsg := lowerer.errors[0].Error()
		if !strings.Contains(errMsg, "no implementation") {
			t.Errorf("Expected helpful error message, got: %s", errMsg)
		}
	}
}

// TestOpLowering_FallbackPath verifies that the fallback path still works
// when CoreTI is unavailable (backward compatibility)
func TestOpLowering_FallbackPath(t *testing.T) {
	// Create a concat intrinsic
	intrinsic := &core.Intrinsic{
		CoreNode: core.CoreNode{NodeID: 100},
		Op:       core.OpConcat,
		Args: []core.CoreExpr{
			&core.Var{CoreNode: core.CoreNode{NodeID: 1}, Name: "left"},
			&core.Var{CoreNode: core.CoreNode{NodeID: 2}, Name: "right"},
		},
	}

	// Create lowerer WITHOUT populating CoreTI (fallback scenario)
	typeEnv := types.NewTypeEnv()
	coreTI := types.NewCoreTypeInfo() // Empty - no type information

	lowerer := NewOpLowerer(typeEnv, coreTI)

	// Lower the intrinsic (should use default fallback)
	lowered := lowerer.lowerExpr(intrinsic)

	// Verify it was lowered to an App node
	app, ok := lowered.(*core.App)
	if !ok {
		t.Fatalf("Expected App node, got %T", lowered)
	}

	// Verify the function is a builtin reference
	builtinRef, ok := app.Func.(*core.VarGlobal)
	if !ok {
		t.Fatalf("Expected VarGlobal for builtin, got %T", app.Func)
	}

	// Default for ++ is concat_String (backward compatibility)
	if builtinRef.Ref.Name != "concat_String" {
		t.Errorf("Expected concat_String (default), got %s", builtinRef.Ref.Name)
	}
}
