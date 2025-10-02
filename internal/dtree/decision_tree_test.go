package dtree

import (
	"testing"

	"github.com/sunholo/ailang/internal/core"
)

// TestDecisionTree_SimpleBoolMatch tests decision tree compilation for Bool match
func TestDecisionTree_SimpleBoolMatch(t *testing.T) {
	// match x { true => 1, false => 0 }
	arms := []core.MatchArm{
		{
			Pattern: &core.LitPattern{Value: true},
			Body:    &core.Lit{Kind: core.IntLit, Value: 1},
		},
		{
			Pattern: &core.LitPattern{Value: false},
			Body:    &core.Lit{Kind: core.IntLit, Value: 0},
		},
	}

	compiler := NewDecisionTreeCompiler(arms)
	tree := compiler.Compile()

	// Should create a switch node
	switchNode, ok := tree.(*SwitchNode)
	if !ok {
		t.Fatalf("Expected SwitchNode, got %T", tree)
	}

	// Should have 2 cases (true and false)
	if len(switchNode.Cases) != 2 {
		t.Errorf("Expected 2 cases, got %d", len(switchNode.Cases))
	}

	// Check that true and false are in the cases
	if _, ok := switchNode.Cases[true]; !ok {
		t.Error("Missing case for true")
	}
	if _, ok := switchNode.Cases[false]; !ok {
		t.Error("Missing case for false")
	}
}

// TestDecisionTree_WithWildcard tests decision tree with wildcard
func TestDecisionTree_WithWildcard(t *testing.T) {
	// match x { true => 1, _ => 0 }
	arms := []core.MatchArm{
		{
			Pattern: &core.LitPattern{Value: true},
			Body:    &core.Lit{Kind: core.IntLit, Value: 1},
		},
		{
			Pattern: &core.WildcardPattern{},
			Body:    &core.Lit{Kind: core.IntLit, Value: 0},
		},
	}

	compiler := NewDecisionTreeCompiler(arms)
	tree := compiler.Compile()

	// Should create a switch node with default
	switchNode, ok := tree.(*SwitchNode)
	if !ok {
		t.Fatalf("Expected SwitchNode, got %T", tree)
	}

	if switchNode.Default == nil {
		t.Error("Expected default branch for wildcard")
	}
}

// TestDecisionTree_AllWildcards tests decision tree with all wildcards
func TestDecisionTree_AllWildcards(t *testing.T) {
	// match x { _ => 42 }
	arms := []core.MatchArm{
		{
			Pattern: &core.WildcardPattern{},
			Body:    &core.Lit{Kind: core.IntLit, Value: 42},
		},
	}

	compiler := NewDecisionTreeCompiler(arms)
	tree := compiler.Compile()

	// Should create a leaf node directly
	leaf, ok := tree.(*LeafNode)
	if !ok {
		t.Fatalf("Expected LeafNode for wildcard-only match, got %T", tree)
	}

	if leaf.ArmIndex != 0 {
		t.Errorf("Expected arm index 0, got %d", leaf.ArmIndex)
	}
}

// TestCanCompileToTree tests the heuristic for when to use decision trees
func TestCanCompileToTree(t *testing.T) {
	tests := []struct {
		name     string
		arms     []core.MatchArm
		expected bool
	}{
		{
			name: "Single arm - not worth it",
			arms: []core.MatchArm{
				{Pattern: &core.LitPattern{Value: true}},
			},
			expected: false,
		},
		{
			name: "Two wildcards - not worth it",
			arms: []core.MatchArm{
				{Pattern: &core.WildcardPattern{}},
				{Pattern: &core.WildcardPattern{}},
			},
			expected: false,
		},
		{
			name: "Multiple literals - worth it",
			arms: []core.MatchArm{
				{Pattern: &core.LitPattern{Value: true}},
				{Pattern: &core.LitPattern{Value: false}},
				{Pattern: &core.WildcardPattern{}},
			},
			expected: true,
		},
		{
			name: "Multiple constructors - worth it",
			arms: []core.MatchArm{
				{Pattern: &core.ConstructorPattern{Name: "Some"}},
				{Pattern: &core.ConstructorPattern{Name: "None"}},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CanCompileToTree(tt.arms)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
