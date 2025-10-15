package elaborate

import (
	"testing"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// TestExhaustiveness_BoolComplete tests exhaustive Bool match
func TestExhaustiveness_BoolComplete(t *testing.T) {
	// match x { true => 1, false => 0 }
	match := &core.Match{
		Scrutinee: &core.Var{Name: "x"},
		Arms: []core.MatchArm{
			{
				Pattern: &core.LitPattern{Value: true},
				Body:    &core.Lit{Kind: core.IntLit, Value: 1},
			},
			{
				Pattern: &core.LitPattern{Value: false},
				Body:    &core.Lit{Kind: core.IntLit, Value: 0},
			},
		},
	}

	checker := NewExhaustivenessChecker()
	exhaustive, missing := checker.CheckExhaustiveness(match, &types.TCon{Name: "Bool"})

	if !exhaustive {
		t.Errorf("Expected exhaustive match, but got missing patterns: %v", missing)
	}
	if len(missing) > 0 {
		t.Errorf("Expected no missing patterns, got: %v", missing)
	}
}

// TestExhaustiveness_BoolIncomplete tests non-exhaustive Bool match
func TestExhaustiveness_BoolIncomplete(t *testing.T) {
	// match x { true => 1 }  -- missing false
	match := &core.Match{
		Scrutinee: &core.Var{Name: "x"},
		Arms: []core.MatchArm{
			{
				Pattern: &core.LitPattern{Value: true},
				Body:    &core.Lit{Kind: core.IntLit, Value: 1},
			},
		},
	}

	checker := NewExhaustivenessChecker()
	exhaustive, missing := checker.CheckExhaustiveness(match, &types.TCon{Name: "Bool"})

	if exhaustive {
		t.Error("Expected non-exhaustive match")
	}
	if len(missing) != 1 {
		t.Errorf("Expected 1 missing pattern, got: %v", missing)
	}
	// Should report false as missing
	if len(missing) > 0 && missing[0] != "false" {
		t.Errorf("Expected missing pattern 'false', got: %v", missing[0])
	}
}

// TestExhaustiveness_WildcardCoversAll tests that wildcard makes match exhaustive
func TestExhaustiveness_WildcardCoversAll(t *testing.T) {
	// match x { true => 1, _ => 0 }
	match := &core.Match{
		Scrutinee: &core.Var{Name: "x"},
		Arms: []core.MatchArm{
			{
				Pattern: &core.LitPattern{Value: true},
				Body:    &core.Lit{Kind: core.IntLit, Value: 1},
			},
			{
				Pattern: &core.WildcardPattern{},
				Body:    &core.Lit{Kind: core.IntLit, Value: 0},
			},
		},
	}

	checker := NewExhaustivenessChecker()
	exhaustive, missing := checker.CheckExhaustiveness(match, &types.TCon{Name: "Bool"})

	if !exhaustive {
		t.Errorf("Expected exhaustive match with wildcard, but got missing patterns: %v", missing)
	}
}

// TestExhaustiveness_VarPatternCoversAll tests that variable pattern makes match exhaustive
func TestExhaustiveness_VarPatternCoversAll(t *testing.T) {
	// match x { true => 1, y => 0 }
	match := &core.Match{
		Scrutinee: &core.Var{Name: "x"},
		Arms: []core.MatchArm{
			{
				Pattern: &core.LitPattern{Value: true},
				Body:    &core.Lit{Kind: core.IntLit, Value: 1},
			},
			{
				Pattern: &core.VarPattern{Name: "y"},
				Body:    &core.Lit{Kind: core.IntLit, Value: 0},
			},
		},
	}

	checker := NewExhaustivenessChecker()
	exhaustive, missing := checker.CheckExhaustiveness(match, &types.TCon{Name: "Bool"})

	if !exhaustive {
		t.Errorf("Expected exhaustive match with variable pattern, but got missing patterns: %v", missing)
	}
}

// TestExhaustiveness_GuardedNotCounted tests that guarded patterns don't guarantee coverage
func TestExhaustiveness_GuardedNotCounted(t *testing.T) {
	// match x { true if y => 1, false => 0 }  -- true with guard doesn't count as covering true
	match := &core.Match{
		Scrutinee: &core.Var{Name: "x"},
		Arms: []core.MatchArm{
			{
				Pattern: &core.LitPattern{Value: true},
				Guard:   &core.Var{Name: "y"}, // Has guard
				Body:    &core.Lit{Kind: core.IntLit, Value: 1},
			},
			{
				Pattern: &core.LitPattern{Value: false},
				Body:    &core.Lit{Kind: core.IntLit, Value: 0},
			},
		},
	}

	checker := NewExhaustivenessChecker()
	exhaustive, missing := checker.CheckExhaustiveness(match, &types.TCon{Name: "Bool"})

	// With our conservative guard handling, guarded patterns don't count
	// So we should see 'true' as missing
	if exhaustive {
		t.Error("Expected non-exhaustive match when guard is present")
	}
	if len(missing) != 1 {
		t.Errorf("Expected 1 missing pattern (true), got: %v", missing)
	}
}

// TestExhaustiveness_InfiniteType tests that infinite types (Int) always exhaustive with wildcard
func TestExhaustiveness_InfiniteType(t *testing.T) {
	// match x { 0 => "zero", _ => "other" }
	match := &core.Match{
		Scrutinee: &core.Var{Name: "x"},
		Arms: []core.MatchArm{
			{
				Pattern: &core.LitPattern{Value: 0},
				Body:    &core.Lit{Kind: core.StringLit, Value: "zero"},
			},
			{
				Pattern: &core.WildcardPattern{},
				Body:    &core.Lit{Kind: core.StringLit, Value: "other"},
			},
		},
	}

	checker := NewExhaustivenessChecker()
	exhaustive, missing := checker.CheckExhaustiveness(match, &types.TCon{Name: "Int"})

	if !exhaustive {
		t.Errorf("Expected exhaustive match for Int with wildcard, but got missing patterns: %v", missing)
	}
}

// TestExhaustiveness_InfiniteTypeIncomplete tests that infinite types without wildcard are incomplete
func TestExhaustiveness_InfiniteTypeIncomplete(t *testing.T) {
	// match x { 0 => "zero", 1 => "one" }  -- missing other integers
	match := &core.Match{
		Scrutinee: &core.Var{Name: "x"},
		Arms: []core.MatchArm{
			{
				Pattern: &core.LitPattern{Value: 0},
				Body:    &core.Lit{Kind: core.StringLit, Value: "zero"},
			},
			{
				Pattern: &core.LitPattern{Value: 1},
				Body:    &core.Lit{Kind: core.StringLit, Value: "one"},
			},
		},
	}

	checker := NewExhaustivenessChecker()
	exhaustive, missing := checker.CheckExhaustiveness(match, &types.TCon{Name: "Int"})

	if exhaustive {
		t.Error("Expected non-exhaustive match for Int without wildcard")
	}
	// Should report wildcard as missing (representing all other integers)
	if len(missing) == 0 {
		t.Error("Expected missing patterns for incomplete Int match")
	}
}
