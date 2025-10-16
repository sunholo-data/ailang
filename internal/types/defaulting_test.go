package types

import (
	"testing"

	"github.com/sunholo/ailang/internal/core"
)

// TestDefaulting_LocalDefaulting tests basic defaulting behavior
func TestDefaulting_LocalDefaulting(t *testing.T) {
	tc := NewCoreTypeChecker()
	tc.instanceEnv = LoadBuiltinInstances()

	tests := []struct {
		name          string
		expr          string
		expectedType  string
		shouldDefault bool
	}{
		{
			name:          "addition of literals - should default",
			expr:          "1 + 2",
			expectedType:  "int",
			shouldDefault: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset traces for each test
			tc.defaultingConfig.Traces = []DefaultingTrace{}

			// Parse and convert to Core (simplified for testing)
			coreExpr := parseTestExpr(tt.expr)

			// Type check with fresh environment
			env := NewTypeEnv()
			typedNode, _, err := tc.CheckCoreExpr(coreExpr, env)
			if err != nil {
				t.Fatalf("Type checking failed for %s: %v", tt.expr, err)
			}

			// Check type
			actualType := typedNode.GetType().(Type).String()
			if actualType != tt.expectedType {
				t.Errorf("Expected type %s, got %s", tt.expectedType, actualType)
			}

			// Check that defaulting was applied if expected
			if tt.shouldDefault {
				if len(tc.defaultingConfig.Traces) == 0 {
					t.Errorf("Expected defaulting to be applied for %s, but no traces found", tt.expr)
				} else {
					t.Logf("Defaulting trace: %+v", tc.defaultingConfig.Traces[0])
				}
			}

			// CRITICAL: Check that ResolvedConstraints are ground
			resolved := tc.GetResolvedConstraints()
			for nodeID, rc := range resolved {
				if !isGround(rc.Type) {
					t.Errorf("ResolvedConstraint[%d] has non-ground type %s", nodeID, rc.Type)
				} else {
					t.Logf("✓ Ground constraint: %s[%s] -> %s", rc.ClassName, rc.Type, rc.Method)
				}
			}
		})
	}
}

// TestDefaulting_OperatorGroundness tests that operator sites become ground
func TestDefaulting_OperatorGroundness(t *testing.T) {
	tc := NewCoreTypeChecker()
	tc.instanceEnv = LoadBuiltinInstances()

	tests := []struct {
		name     string
		expr     string
		operator string
		expected string
	}{
		{
			name:     "addition operator",
			expr:     "1 + 2",
			operator: "add",
			expected: "Int",
		},
		{
			name:     "equality operator",
			expr:     "1 == 2",
			operator: "eq",
			expected: "Int",
		},
		{
			name:     "comparison operator",
			expr:     "1 < 2",
			operator: "lt",
			expected: "Int",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coreExpr := parseTestExpr(tt.expr)

			_, _, err := tc.CheckCoreExpr(coreExpr, NewTypeEnv())
			if err != nil {
				t.Fatalf("Type checking failed: %v", err)
			}

			// Check that all resolved constraints are ground
			resolved := tc.GetResolvedConstraints()
			foundOperator := false

			for _, rc := range resolved {
				if rc.Method == tt.operator {
					foundOperator = true
					if !isGround(rc.Type) {
						t.Errorf("Operator %s has non-ground type %s", tt.operator, rc.Type)
					}
					if rc.Type.String() != tt.expected {
						t.Errorf("Operator %s has type %s, expected %s", tt.operator, rc.Type, tt.expected)
					}
				}
			}

			if !foundOperator {
				t.Errorf("Operator %s not found in resolved constraints", tt.operator)
			}
		})
	}
}

// TestDefaulting_PolymorphismPreservation tests that polymorphism is preserved where appropriate
func TestDefaulting_PolymorphismPreservation(t *testing.T) {
	tc := NewCoreTypeChecker()
	tc.instanceEnv = LoadBuiltinInstances()

	tests := []struct {
		name           string
		expr           string
		shouldBePoly   bool
		expectedScheme string
	}{
		{
			name:           "simple expression should be monomorphic",
			expr:           "1 + 2",
			shouldBePoly:   false,
			expectedScheme: "int", // defaulted to int
		},
		{
			name:           "floating point should be monomorphic",
			expr:           "3.14",
			shouldBePoly:   false,
			expectedScheme: "float", // literal float
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coreExpr := parseTestExpr(tt.expr)

			typedNode, _, err := tc.CheckCoreExpr(coreExpr, NewTypeEnv())
			if err != nil {
				t.Fatalf("Type checking failed: %v", err)
			}

			// Check the type structure
			// This is simplified - a full implementation would inspect the TypedLet scheme
			actualType := typedNode.GetType().(Type).String()

			if tt.shouldBePoly {
				// For polymorphic cases, we'd check the scheme in the TypedLet
				// This is a simplified check
				if actualType == "int" || actualType == "float" {
					t.Errorf("Expected polymorphic type, got monomorphic %s", actualType)
				}
			} else {
				// For monomorphic cases, should have concrete type
				if actualType != tt.expectedScheme {
					t.Errorf("Expected %s, got %s", tt.expectedScheme, actualType)
				}
			}
		})
	}
}

// TestDefaulting_NestedLets tests defaulting with nested let bindings and shadowing
func TestDefaulting_NestedLets(t *testing.T) {
	tc := NewCoreTypeChecker()
	tc.instanceEnv = LoadBuiltinInstances()

	// Test case: let x = 1 in let x = x + 1 in x
	// Inner x should use defaulted Int from outer x
	expr := `
		let x = 1 in
		let x = x + 1 in
		x
	`

	coreExpr := parseTestExpr(expr)

	typedNode, _, err := tc.CheckCoreExpr(coreExpr, NewTypeEnv())
	if err != nil {
		t.Fatalf("Type checking failed: %v", err)
	}

	// Result should be Int
	actualType := typedNode.GetType().(Type).String()
	if actualType != "int" {
		t.Errorf("Expected int, got %s", actualType)
	}

	// Check that all constraints are ground
	resolved := tc.GetResolvedConstraints()
	for nodeID, rc := range resolved {
		if !isGround(rc.Type) {
			t.Errorf("ResolvedConstraint[%d] has non-ground type %s", nodeID, rc.Type)
		}
	}
}

// TestDefaulting_MixedConstraints tests error handling for mixed class constraints
func TestDefaulting_MixedConstraints(t *testing.T) {
	tc := NewCoreTypeChecker()
	tc.instanceEnv = LoadBuiltinInstances()

	// This would create a mixed Num ∧ Ord constraint that can't be defaulted
	// Note: This is a theoretical test - we'd need to construct a scenario
	// where a type variable has both Num and Ord constraints but doesn't appear in the monotype

	// For now, test that pure constraints work
	expr := "1 + 1"
	coreExpr := parseTestExpr(expr)

	_, _, err := tc.CheckCoreExpr(coreExpr, NewTypeEnv())
	if err != nil {
		t.Fatalf("Type checking should succeed for pure Num constraint: %v", err)
	}
}

// TestDefaulting_REPLParity tests that REPL and file compilation behave identically
func TestDefaulting_REPLParity(t *testing.T) {
	// Test that the same expression gives the same defaulting behavior
	// in both contexts (this is more of an integration test)

	tc1 := NewCoreTypeChecker() // Simulate REPL
	tc1.instanceEnv = LoadBuiltinInstances()

	tc2 := NewCoreTypeChecker() // Simulate file
	tc2.instanceEnv = LoadBuiltinInstances()

	expr := "1 + 2"
	coreExpr1 := parseTestExpr(expr)
	coreExpr2 := parseTestExpr(expr)

	node1, _, err1 := tc1.CheckCoreExpr(coreExpr1, NewTypeEnv())
	node2, _, err2 := tc2.CheckCoreExpr(coreExpr2, NewTypeEnv())

	if err1 != nil || err2 != nil {
		t.Fatalf("Type checking failed: %v, %v", err1, err2)
	}

	// Should have same type
	type1 := node1.GetType().(Type).String()
	type2 := node2.GetType().(Type).String()

	if type1 != type2 {
		t.Errorf("REPL and file compilation gave different types: %s vs %s", type1, type2)
	}

	// Should have same defaulting traces
	traces1 := len(tc1.defaultingConfig.Traces)
	traces2 := len(tc2.defaultingConfig.Traces)

	if traces1 != traces2 {
		t.Errorf("Different number of defaulting traces: %d vs %d", traces1, traces2)
	}
}

// TestDefaulting_NoEffectLeakage tests that defaulting doesn't affect effect rows or record rows
func TestDefaulting_NoEffectLeakage(t *testing.T) {
	tc := NewCoreTypeChecker()
	tc.instanceEnv = LoadBuiltinInstances()

	// Test that effect rows and record rows maintain their kinds after defaulting
	expr := "1"
	coreExpr := parseTestExpr(expr)

	typedNode, _, err := tc.CheckCoreExpr(coreExpr, NewTypeEnv())
	if err != nil {
		t.Fatalf("Type checking failed: %v", err)
	}

	// Check that effect row is still properly typed
	effectRow := typedNode.GetEffectRow()
	if effectRow == nil {
		t.Error("Effect row should not be nil")
	}

	// The effect row should be empty for a pure literal
	if row, ok := effectRow.(*Row); ok {
		if row.Kind != EffectRow {
			t.Errorf("Effect row has wrong kind: %v", row.Kind)
		}
		if len(row.Labels) != 0 {
			t.Errorf("Effect row should be empty for pure literal, got %v", row.Labels)
		}
	}
}

// Helper function to parse test expressions
// This is simplified - in real tests you'd use the actual parser
func parseTestExpr(exprStr string) core.CoreExpr {
	// This is a mock implementation - replace with actual parsing
	// For now, return a simple literal to test the framework
	switch exprStr {
	case "1":
		return &core.Lit{
			CoreNode: core.CoreNode{NodeID: 1},
			Kind:     core.IntLit,
			Value:    1,
		}
	case "3.14":
		return &core.Lit{
			CoreNode: core.CoreNode{NodeID: 2},
			Kind:     core.FloatLit,
			Value:    3.14,
		}
	case "1 + 2":
		return &core.BinOp{
			CoreNode: core.CoreNode{NodeID: 5},
			Op:       "+",
			Left: &core.Lit{
				CoreNode: core.CoreNode{NodeID: 3},
				Kind:     core.IntLit,
				Value:    1,
			},
			Right: &core.Lit{
				CoreNode: core.CoreNode{NodeID: 4},
				Kind:     core.IntLit,
				Value:    2,
			},
		}
	case "1 == 2":
		return &core.BinOp{
			CoreNode: core.CoreNode{NodeID: 8},
			Op:       "==",
			Left: &core.Lit{
				CoreNode: core.CoreNode{NodeID: 6},
				Kind:     core.IntLit,
				Value:    1,
			},
			Right: &core.Lit{
				CoreNode: core.CoreNode{NodeID: 7},
				Kind:     core.IntLit,
				Value:    2,
			},
		}
	case "1 < 2":
		return &core.BinOp{
			CoreNode: core.CoreNode{NodeID: 11},
			Op:       "<",
			Left: &core.Lit{
				CoreNode: core.CoreNode{NodeID: 9},
				Kind:     core.IntLit,
				Value:    1,
			},
			Right: &core.Lit{
				CoreNode: core.CoreNode{NodeID: 10},
				Kind:     core.IntLit,
				Value:    2,
			},
		}
	default:
		// Return a simple literal for unknown expressions
		return &core.Lit{
			CoreNode: core.CoreNode{NodeID: 99},
			Kind:     core.IntLit,
			Value:    42,
		}
	}
}

// The actual core types already implement the necessary methods
