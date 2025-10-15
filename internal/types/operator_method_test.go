package types

import (
	"fmt"
	"testing"

	"github.com/sunholo/ailang/internal/core"
)

// TestOperatorMethodMapping tests the OperatorMethod function directly
func TestOperatorMethodMapping(t *testing.T) {
	tests := []struct {
		operator string
		isUnary  bool
		expected string
	}{
		// Binary operators
		{"+", false, "add"},
		{"-", false, "sub"},
		{"*", false, "mul"},
		{"/", false, "div"},
		{"==", false, "eq"},
		{"!=", false, "neq"},
		{"<", false, "lt"},
		{"<=", false, "lte"},
		{">", false, "gt"},
		{">=", false, "gte"},

		// Unary operators
		{"-", true, "neg"},
		{"!", true, "not"},

		// Unknown operators
		{"unknown", false, ""},
		{"unknown", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.operator+"_unary_"+fmt.Sprintf("%t", tt.isUnary), func(t *testing.T) {
			result := OperatorMethod(tt.operator, tt.isUnary)
			if result != tt.expected {
				t.Errorf("OperatorMethod(%q, %t) = %q, want %q",
					tt.operator, tt.isUnary, result, tt.expected)
			}
		})
	}
}

// TestFillOperatorMethodsIntegration tests that FillOperatorMethods correctly sets method names
func TestFillOperatorMethodsIntegration(t *testing.T) {
	// Create a type checker with mock resolved constraints
	tc := NewCoreTypeChecker()

	// Create a mock BinOp node
	binOp := &core.BinOp{
		CoreNode: core.CoreNode{NodeID: 1},
		Op:       "*",
		Left:     &core.Lit{CoreNode: core.CoreNode{NodeID: 2}, Kind: core.IntLit, Value: 2},
		Right:    &core.Lit{CoreNode: core.CoreNode{NodeID: 3}, Kind: core.IntLit, Value: 3},
	}

	// Create a mock resolved constraint for the BinOp node
	tc.resolvedConstraints = make(map[uint64]*ResolvedConstraint)
	tc.resolvedConstraints[1] = &ResolvedConstraint{
		NodeID:    1,
		ClassName: "Num",
		Type:      &TCon{Name: "Int"},
		Method:    "", // Initially empty - should be filled by FillOperatorMethods
	}

	// Call FillOperatorMethods
	tc.FillOperatorMethods(binOp)

	// Verify that the method was set correctly
	constraint := tc.resolvedConstraints[1]
	if constraint.Method != "mul" {
		t.Errorf("Expected method 'mul' for '*' operator, got '%s'", constraint.Method)
	}
}

// TestMultipleOperatorsInExpression tests that all operators in a complex expression get correct methods
func TestMultipleOperatorsInExpression(t *testing.T) {
	tc := NewCoreTypeChecker()

	// Create a complex expression: (2 + 3) * 4
	addOp := &core.BinOp{
		CoreNode: core.CoreNode{NodeID: 1},
		Op:       "+",
		Left:     &core.Lit{CoreNode: core.CoreNode{NodeID: 2}, Kind: core.IntLit, Value: 2},
		Right:    &core.Lit{CoreNode: core.CoreNode{NodeID: 3}, Kind: core.IntLit, Value: 3},
	}

	mulOp := &core.BinOp{
		CoreNode: core.CoreNode{NodeID: 4},
		Op:       "*",
		Left:     addOp,
		Right:    &core.Lit{CoreNode: core.CoreNode{NodeID: 5}, Kind: core.IntLit, Value: 4},
	}

	// Create resolved constraints for both operations
	tc.resolvedConstraints = make(map[uint64]*ResolvedConstraint)
	tc.resolvedConstraints[1] = &ResolvedConstraint{
		NodeID:    1,
		ClassName: "Num",
		Type:      &TCon{Name: "Int"},
		Method:    "", // Should become "add"
	}
	tc.resolvedConstraints[4] = &ResolvedConstraint{
		NodeID:    4,
		ClassName: "Num",
		Type:      &TCon{Name: "Int"},
		Method:    "", // Should become "mul"
	}

	// Call FillOperatorMethods on the root expression
	tc.FillOperatorMethods(mulOp)

	// Verify both methods were set correctly
	if tc.resolvedConstraints[1].Method != "add" {
		t.Errorf("Expected method 'add' for '+' operator, got '%s'", tc.resolvedConstraints[1].Method)
	}
	if tc.resolvedConstraints[4].Method != "mul" {
		t.Errorf("Expected method 'mul' for '*' operator, got '%s'", tc.resolvedConstraints[4].Method)
	}
}

// TestBinaryVsUnaryOperators ensures unary and binary operators with same symbol are handled correctly
func TestBinaryVsUnaryOperators(t *testing.T) {
	tests := []struct {
		name     string
		operator string
		isUnary  bool
		expected string
	}{
		{"binary_minus", "-", false, "sub"},
		{"unary_minus", "-", true, "neg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := OperatorMethod(tt.operator, tt.isUnary)
			if result != tt.expected {
				t.Errorf("OperatorMethod(%q, %t) = %q, want %q",
					tt.operator, tt.isUnary, result, tt.expected)
			}
		})
	}
}

// TestNoRegressionAllOperatorsAddBug ensures the specific bug where all operators became "add" doesn't happen
func TestNoRegressionAllOperatorsAddBug(t *testing.T) {
	operators := []string{"*", "/", "-", "==", "!=", "<", ">", "<=", ">="}

	for _, op := range operators {
		t.Run("operator_"+op, func(t *testing.T) {
			method := OperatorMethod(op, false)
			if method == "add" && op != "+" {
				t.Errorf("REGRESSION: Operator '%s' incorrectly mapped to 'add' method", op)
			}
			if method == "" {
				t.Errorf("Operator '%s' has no method mapping", op)
			}
		})
	}
}
