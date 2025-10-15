package eval

import (
	"testing"

	"github.com/sunholo/ailang/internal/core"
)

// TestGuards_BasicTrue tests a guard that evaluates to true
func TestGuards_BasicTrue(t *testing.T) {
	// match 5 { x if true => "yes", x => "no" }
	match := &core.Match{
		Scrutinee: &core.Lit{Kind: core.IntLit, Value: 5},
		Arms: []core.MatchArm{
			{
				Pattern: &core.VarPattern{Name: "x"},
				Guard:   &core.Lit{Kind: core.BoolLit, Value: true},
				Body:    &core.Lit{Kind: core.StringLit, Value: "yes"},
			},
			{
				Pattern: &core.VarPattern{Name: "x"},
				Body:    &core.Lit{Kind: core.StringLit, Value: "no"},
			},
		},
	}

	eval := NewCoreEvaluator()
	result, err := eval.evalCoreMatch(match)
	if err != nil {
		t.Fatalf("Evaluation failed: %v", err)
	}

	strVal, ok := result.(*StringValue)
	if !ok {
		t.Fatalf("Expected StringValue, got %T", result)
	}

	if strVal.Value != "yes" {
		t.Errorf("Expected 'yes', got '%s'", strVal.Value)
	}
}

// TestGuards_BasicFalse tests a guard that evaluates to false (should skip to next arm)
func TestGuards_BasicFalse(t *testing.T) {
	// match 5 { x if false => "yes", x => "no" }
	match := &core.Match{
		Scrutinee: &core.Lit{Kind: core.IntLit, Value: 5},
		Arms: []core.MatchArm{
			{
				Pattern: &core.VarPattern{Name: "x"},
				Guard:   &core.Lit{Kind: core.BoolLit, Value: false},
				Body:    &core.Lit{Kind: core.StringLit, Value: "yes"},
			},
			{
				Pattern: &core.VarPattern{Name: "x"},
				Body:    &core.Lit{Kind: core.StringLit, Value: "no"},
			},
		},
	}

	eval := NewCoreEvaluator()
	result, err := eval.evalCoreMatch(match)
	if err != nil {
		t.Fatalf("Evaluation failed: %v", err)
	}

	strVal, ok := result.(*StringValue)
	if !ok {
		t.Fatalf("Expected StringValue, got %T", result)
	}

	if strVal.Value != "no" {
		t.Errorf("Expected 'no', got '%s'", strVal.Value)
	}
}

// TestGuards_MultipleSequential tests multiple guards in sequence
func TestGuards_MultipleSequential(t *testing.T) {
	// match 5 {
	//   x if false => "first",
	//   x if false => "second",
	//   x if true => "third",
	//   x => "fourth"
	// }
	match := &core.Match{
		Scrutinee: &core.Lit{Kind: core.IntLit, Value: 5},
		Arms: []core.MatchArm{
			{
				Pattern: &core.VarPattern{Name: "x"},
				Guard:   &core.Lit{Kind: core.BoolLit, Value: false},
				Body:    &core.Lit{Kind: core.StringLit, Value: "first"},
			},
			{
				Pattern: &core.VarPattern{Name: "x"},
				Guard:   &core.Lit{Kind: core.BoolLit, Value: false},
				Body:    &core.Lit{Kind: core.StringLit, Value: "second"},
			},
			{
				Pattern: &core.VarPattern{Name: "x"},
				Guard:   &core.Lit{Kind: core.BoolLit, Value: true},
				Body:    &core.Lit{Kind: core.StringLit, Value: "third"},
			},
			{
				Pattern: &core.VarPattern{Name: "x"},
				Body:    &core.Lit{Kind: core.StringLit, Value: "fourth"},
			},
		},
	}

	eval := NewCoreEvaluator()
	result, err := eval.evalCoreMatch(match)
	if err != nil {
		t.Fatalf("Evaluation failed: %v", err)
	}

	strVal, ok := result.(*StringValue)
	if !ok {
		t.Fatalf("Expected StringValue, got %T", result)
	}

	if strVal.Value != "third" {
		t.Errorf("Expected 'third', got '%s'", strVal.Value)
	}
}

// TestGuards_AccessBinding tests that guard can access pattern bindings
func TestGuards_AccessBinding(t *testing.T) {
	// match true { x if x => "bound true", _ => "other" }
	match := &core.Match{
		Scrutinee: &core.Lit{Kind: core.BoolLit, Value: true},
		Arms: []core.MatchArm{
			{
				Pattern: &core.VarPattern{Name: "x"},
				Guard:   &core.Var{Name: "x"}, // Access the bound variable
				Body:    &core.Lit{Kind: core.StringLit, Value: "bound true"},
			},
			{
				Pattern: &core.WildcardPattern{},
				Body:    &core.Lit{Kind: core.StringLit, Value: "other"},
			},
		},
	}

	eval := NewCoreEvaluator()
	result, err := eval.evalCoreMatch(match)
	if err != nil {
		t.Fatalf("Evaluation failed: %v", err)
	}

	strVal, ok := result.(*StringValue)
	if !ok {
		t.Fatalf("Expected StringValue, got %T", result)
	}

	if strVal.Value != "bound true" {
		t.Errorf("Expected 'bound true', got '%s'", strVal.Value)
	}
}

// TestGuards_NonBoolError tests that non-Bool guards cause errors
func TestGuards_NonBoolError(t *testing.T) {
	// match 5 { x if 42 => "yes", x => "no" }
	// Guard must be Bool, not Int
	match := &core.Match{
		Scrutinee: &core.Lit{Kind: core.IntLit, Value: 5},
		Arms: []core.MatchArm{
			{
				Pattern: &core.VarPattern{Name: "x"},
				Guard:   &core.Lit{Kind: core.IntLit, Value: 42},
				Body:    &core.Lit{Kind: core.StringLit, Value: "yes"},
			},
			{
				Pattern: &core.VarPattern{Name: "x"},
				Body:    &core.Lit{Kind: core.StringLit, Value: "no"},
			},
		},
	}

	eval := NewCoreEvaluator()
	_, err := eval.evalCoreMatch(match)
	if err == nil {
		t.Fatal("Expected error for non-Bool guard")
	}

	// Check error message contains "Bool"
	if !containsString(err.Error(), "Bool") {
		t.Errorf("Expected error about Bool, got: %v", err)
	}
}

// TestGuards_AllFail tests that all failing guards leads to non-exhaustive error
func TestGuards_AllFail(t *testing.T) {
	// match 5 { x if false => "first", x if false => "second" }
	match := &core.Match{
		Scrutinee: &core.Lit{Kind: core.IntLit, Value: 5},
		Arms: []core.MatchArm{
			{
				Pattern: &core.VarPattern{Name: "x"},
				Guard:   &core.Lit{Kind: core.BoolLit, Value: false},
				Body:    &core.Lit{Kind: core.StringLit, Value: "first"},
			},
			{
				Pattern: &core.VarPattern{Name: "x"},
				Guard:   &core.Lit{Kind: core.BoolLit, Value: false},
				Body:    &core.Lit{Kind: core.StringLit, Value: "second"},
			},
		},
	}

	eval := NewCoreEvaluator()
	_, err := eval.evalCoreMatch(match)
	if err == nil {
		t.Fatal("Expected error for non-exhaustive match")
	}

	// Check error message contains "no pattern matched"
	if !containsString(err.Error(), "no pattern matched") {
		t.Errorf("Expected 'no pattern matched' error, got: %v", err)
	}
}

// Helper function to check if string contains substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && hasSubstring(s, substr)
}

func hasSubstring(s, substr string) bool {
	if s == substr {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
