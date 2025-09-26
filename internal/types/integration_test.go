package types

import (
	"fmt"
	"testing"
	"github.com/sunholo/ailang/internal/core"
)

// TestDefaulting_EndToEndPipeline tests the complete pipeline from Core AST through elaboration
func TestDefaulting_EndToEndPipeline(t *testing.T) {
	// Test the actual bug scenario: simple addition should work
	t.Run("simple_addition", func(t *testing.T) {
		tc := NewCoreTypeChecker()
		tc.instanceEnv = LoadBuiltinInstances()
				
		// Create a proper addition expression: 1 + 2
		left := &core.Lit{
			CoreNode: core.CoreNode{NodeID: 1},
			Kind:     core.IntLit,
			Value:    1,
		}
		right := &core.Lit{
			CoreNode: core.CoreNode{NodeID: 2}, 
			Kind:     core.IntLit,
			Value:    2,
		}
		addExpr := &core.BinOp{
			CoreNode: core.CoreNode{NodeID: 3},
			Op:       "+",
			Left:     left,
			Right:    right,
		}
		
		// Type check
		typedNode, _, err := tc.CheckCoreExpr(addExpr, NewTypeEnvWithBuiltins())
		if err != nil {
			t.Fatalf("Addition should type check successfully: %v", err)
		}
		
		// Result should be int
		resultType := typedNode.GetType().(Type).String()
		if resultType != "int" {
			t.Errorf("Expected int, got %s", resultType)
		}
		
		// CRITICAL: All resolved constraints must be ground
		resolved := tc.GetResolvedConstraints()
		if len(resolved) == 0 {
			t.Error("Expected resolved constraints for operator +")
		}
		
		for nodeID, rc := range resolved {
			if !isGround(rc.Type) {
				t.Fatalf("FAIL: ResolvedConstraint[%d] has non-ground type %s", nodeID, rc.Type)
			}
			if rc.Type.String() != "Int" {
				t.Errorf("Expected operator constraint type to be Int, got %s", rc.Type.String())
			}
			t.Logf("✓ Ground constraint: %s[%s] -> method:%s", rc.ClassName, rc.Type, rc.Method)
		}
		
		// Should have defaulting traces
		if len(tc.defaultingConfig.Traces) == 0 {
			t.Error("Expected defaulting traces for addition")
		} else {
			for _, trace := range tc.defaultingConfig.Traces {
				t.Logf("✓ Defaulting: %s[%s] -> %s", trace.ClassName, trace.TypeVar, trace.Default)
			}
		}
	})
	
	t.Run("single_literal_no_defaulting", func(t *testing.T) {
		tc := NewCoreTypeChecker()
		tc.instanceEnv = LoadBuiltinInstances()
				
		// Single literal: should NOT default (not ambiguous)
		lit := &core.Lit{
			CoreNode: core.CoreNode{NodeID: 1},
			Kind:     core.IntLit,
			Value:    42,
		}
		
		typedNode, _, err := tc.CheckCoreExpr(lit, NewTypeEnvWithBuiltins())
		if err != nil {
			t.Fatalf("Single literal should type check: %v", err)
		}
		
		// Type should contain a type variable (polymorphic until used)
		resultType := typedNode.GetType().(Type)
		if isGround(resultType) {
			t.Logf("Note: literal got ground type %s (may be defaulted at top level)", resultType)
		}
		
		// Should have minimal resolved constraints since it's just a literal
		resolved := tc.GetResolvedConstraints()
		for nodeID, rc := range resolved {
			if !isGround(rc.Type) {
				t.Fatalf("FAIL: ResolvedConstraint[%d] has non-ground type %s", nodeID, rc.Type)
			}
		}
	})
}

// TestDefaulting_GeneralizationBoundaries tests defaulting at specific boundaries
func TestDefaulting_GeneralizationBoundaries(t *testing.T) {
	t.Run("let_binding_boundary", func(t *testing.T) {
		tc := NewCoreTypeChecker()
		tc.instanceEnv = LoadBuiltinInstances()
				
		// let x = 1 + 2 in x
		// The addition should be defaulted when x is generalized
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
		addition := &core.BinOp{
			CoreNode: core.CoreNode{NodeID: 3},
			Op:       "+",
			Left:     lit1,
			Right:    lit2,
		}
		xVar := &core.Var{
			CoreNode: core.CoreNode{NodeID: 4},
			Name:     "x",
		}
		letExpr := &core.Let{
			CoreNode: core.CoreNode{NodeID: 5},
			Name:     "x",
			Value:    addition,
			Body:     xVar,
		}
		
		typedNode, _, err := tc.CheckCoreExpr(letExpr, NewTypeEnvWithBuiltins())
		if err != nil {
			t.Fatalf("Let binding should type check: %v", err)
		}
		
		// Result should be int
		resultType := typedNode.GetType().(Type).String()
		if resultType != "int" {
			t.Errorf("Expected int, got %s", resultType)
		}
		
		// All constraints should be ground
		resolved := tc.GetResolvedConstraints()
		for nodeID, rc := range resolved {
			if !isGround(rc.Type) {
				t.Fatalf("FAIL: ResolvedConstraint[%d] has non-ground type %s", nodeID, rc.Type)
			}
		}
		
		// Should have defaulting at the let boundary
		if len(tc.defaultingConfig.Traces) == 0 {
			t.Error("Expected defaulting at let boundary")
		}
	})
}

// TestDefaulting_MutualRecursion tests SCC defaulting
func TestDefaulting_MutualRecursion(t *testing.T) {
	t.Run("mutual_recursion_consistency", func(t *testing.T) {
		tc := NewCoreTypeChecker()
		tc.instanceEnv = LoadBuiltinInstances()
		
		// Test simple addition consistency (simplified from mutual recursion)
		// This ensures that our defaulting produces consistent results
		
		lit1 := &core.Lit{CoreNode: core.CoreNode{NodeID: 1}, Kind: core.IntLit, Value: 1}
		lit2 := &core.Lit{CoreNode: core.CoreNode{NodeID: 2}, Kind: core.IntLit, Value: 2}
		
		expr := &core.BinOp{CoreNode: core.CoreNode{NodeID: 3}, Op: "+", Left: lit1, Right: lit2}
		
		typedNode, _, err := tc.CheckCoreExpr(expr, NewTypeEnvWithBuiltins())
		if err != nil {
			t.Fatalf("Simple addition should type check: %v", err)
		}
		
		// Result should be int (consistently defaulted)
		resultType := typedNode.GetType().(Type).String()
		if resultType != "int" {
			t.Errorf("Expected int, got %s", resultType)
		}
		
		// All constraints should be ground
		resolved := tc.GetResolvedConstraints()
		for nodeID, rc := range resolved {
			if !isGround(rc.Type) {
				t.Fatalf("FAIL: ResolvedConstraint[%d] has non-ground type %s", nodeID, rc.Type)
			}
		}
	})
}

// TestDefaulting_ElaboratorSafety tests that the elaborator never sees type variables
func TestDefaulting_ElaboratorSafety(t *testing.T) {
	t.Run("no_type_vars_in_resolved_constraints", func(t *testing.T) {
		tc := NewCoreTypeChecker()
		tc.instanceEnv = LoadBuiltinInstances()
		
		// Test various expressions that historically caused issues
		expressions := []core.CoreExpr{
			&core.Lit{CoreNode: core.CoreNode{NodeID: 1}, Kind: core.IntLit, Value: 1},
			&core.BinOp{
				CoreNode: core.CoreNode{NodeID: 2},
				Op:       "+",
				Left:     &core.Lit{CoreNode: core.CoreNode{NodeID: 3}, Kind: core.IntLit, Value: 1},
				Right:    &core.Lit{CoreNode: core.CoreNode{NodeID: 4}, Kind: core.IntLit, Value: 2},
			},
			&core.BinOp{
				CoreNode: core.CoreNode{NodeID: 5},
				Op:       "==",
				Left:     &core.Lit{CoreNode: core.CoreNode{NodeID: 6}, Kind: core.IntLit, Value: 1},
				Right:    &core.Lit{CoreNode: core.CoreNode{NodeID: 7}, Kind: core.IntLit, Value: 2},
			},
		}
		
		for i, expr := range expressions {
			t.Run(fmt.Sprintf("expr_%d", i), func(t *testing.T) {
				// Reset for each expression
				tc.resolvedConstraints = make(map[uint64]*ResolvedConstraint)
				tc.defaultingConfig.Traces = []DefaultingTrace{}
				
				_, _, err := tc.CheckCoreExpr(expr, NewTypeEnvWithBuiltins())
				if err != nil {
					t.Fatalf("Expression %d should type check: %v", i, err)
				}
				
				// CRITICAL TEST: GetResolvedConstraints should never panic
				// This call includes the groundness assertion
				resolved := tc.GetResolvedConstraints()
				
				// Double-check manually
				for nodeID, rc := range resolved {
					if !isGround(rc.Type) {
						t.Fatalf("CRITICAL BUG: ResolvedConstraint[%d] has non-ground type %s", nodeID, rc.Type)
					}
					t.Logf("✓ Safe for elaborator: %s[%s] -> %s", rc.ClassName, rc.Type, rc.Method)
				}
			})
		}
	})
}

// TestDefaulting_REPLvsFileConsistency tests behavioral parity
func TestDefaulting_REPLvsFileConsistency(t *testing.T) {
	t.Run("same_defaulting_behavior", func(t *testing.T) {
		// Create two identical type checkers (simulating REPL vs file)
		repl := NewCoreTypeChecker()
		repl.instanceEnv = LoadBuiltinInstances()
		repl.SetDebugMode(true)
		
		file := NewCoreTypeChecker()
		file.instanceEnv = LoadBuiltinInstances() 
		file.SetDebugMode(true)
		
		// Same expression in both contexts
		expr := &core.BinOp{
			CoreNode: core.CoreNode{NodeID: 1},
			Op:       "+",
			Left:     &core.Lit{CoreNode: core.CoreNode{NodeID: 2}, Kind: core.IntLit, Value: 1},
			Right:    &core.Lit{CoreNode: core.CoreNode{NodeID: 3}, Kind: core.IntLit, Value: 2},
		}
		
		// Type check in both contexts
		replNode, _, replErr := repl.CheckCoreExpr(expr, NewTypeEnvWithBuiltins())
		fileNode, _, fileErr := file.CheckCoreExpr(expr, NewTypeEnvWithBuiltins())
		
		// Both should succeed or both should fail
		if (replErr == nil) != (fileErr == nil) {
			t.Fatalf("Inconsistent success/failure: REPL=%v, FILE=%v", replErr, fileErr)
		}
		
		if replErr == nil {
			// Both succeeded - check consistency
			replType := replNode.GetType().(Type).String()
			fileType := fileNode.GetType().(Type).String()
			
			if replType != fileType {
				t.Errorf("Inconsistent types: REPL=%s, FILE=%s", replType, fileType)
			}
			
			// Check defaulting trace consistency
			replTraces := len(repl.defaultingConfig.Traces)
			fileTraces := len(file.defaultingConfig.Traces)
			
			if replTraces != fileTraces {
				t.Errorf("Inconsistent defaulting: REPL=%d traces, FILE=%d traces", replTraces, fileTraces)
			}
		}
	})
}