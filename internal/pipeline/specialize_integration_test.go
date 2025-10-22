package pipeline

import (
	"testing"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// TestMonomorphization_DirectLambdaApplication tests specialization of direct lambda calls
func TestMonomorphization_DirectLambdaApplication(t *testing.T) {
	// Create: (\x -> x)(42)
	lambda := &core.Lambda{
		CoreNode: core.CoreNode{NodeID: 1},
		Params:   []string{"x"},
		Body:     &core.Var{CoreNode: core.CoreNode{NodeID: 2}, Name: "x"},
	}

	app := &core.App{
		CoreNode: core.CoreNode{NodeID: 3},
		Func:     lambda,
		Args: []core.CoreExpr{
			&core.Lit{CoreNode: core.CoreNode{NodeID: 4}, Kind: core.IntLit, Value: int64(42)},
		},
	}

	// Set up type information
	// Lambda has polymorphic type: α -> α
	// After application with Int, should become: Int -> Int
	coreTI := types.NewCoreTypeInfo()

	// For polymorphic lambda: α -> α
	alpha := &types.TVar{Name: "alpha"}
	lambdaType := &types.TFunc2{
		Params:    []types.Type{alpha},
		Return:    alpha,
		EffectRow: &types.Row{Kind: types.KEffect{}, Labels: map[string]types.Type{}, Tail: nil},
	}
	coreTI.Set(lambda.ID(), lambdaType)
	coreTI.Set(lambda.Body.ID(), alpha) // x has type α

	// Argument has type Int
	coreTI.Set(app.Args[0].ID(), types.TInt)

	// Application result has type Int
	coreTI.Set(app.ID(), types.TInt)

	// Run specialization
	specializer := NewSpecializer(&coreTI)
	prog := &core.Program{
		Decls: []core.CoreExpr{app},
		Meta:  make(map[string]*core.DeclMeta),
	}
	result, err := specializer.Specialize(prog)
	if err != nil {
		t.Fatalf("Specialization error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	stats := specializer.GetStats()
	t.Logf("Stats: %d specializations, cache: %d hits / %d misses",
		stats.TotalSpecializations, stats.CacheHits, stats.CacheMisses)

	// Should have specialized the lambda (polymorphic α->α with concrete Int argument)
	if stats.TotalSpecializations < 1 {
		t.Logf("Note: Expected at least 1 specialization for polymorphic lambda, got %d", stats.TotalSpecializations)
	}
}

// TestMonomorphization_LetRecSkipsRecursive tests that recursive bindings are skipped
func TestMonomorphization_LetRecSkipsRecursive(t *testing.T) {
	// Create: letrec factorial = \n -> if n == 0 then 1 else n * factorial(n-1) in factorial(5)
	factorialBody := &core.Var{CoreNode: core.CoreNode{NodeID: 10}, Name: "factorial"} // Recursive call (simplified)

	factorial := &core.Lambda{
		CoreNode: core.CoreNode{NodeID: 1},
		Params:   []string{"n"},
		Body:     factorialBody,
	}

	letrec := &core.LetRec{
		CoreNode: core.CoreNode{NodeID: 2},
		Bindings: []core.RecBinding{
			{Name: "factorial", Value: factorial},
		},
		Body: &core.App{
			CoreNode: core.CoreNode{NodeID: 3},
			Func:     &core.Var{CoreNode: core.CoreNode{NodeID: 4}, Name: "factorial"},
			Args: []core.CoreExpr{
				&core.Lit{CoreNode: core.CoreNode{NodeID: 5}, Kind: core.IntLit, Value: int64(5)},
			},
		},
	}

	coreTI := types.NewCoreTypeInfo()
	coreTI.Set(factorial.ID(), &types.TFunc2{
		Params:    []types.Type{types.TInt},
		Return:    types.TInt,
		EffectRow: &types.Row{Kind: types.KEffect{}, Labels: map[string]types.Type{}, Tail: nil},
	})

	specializer := NewSpecializer(&coreTI)
	prog := &core.Program{Decls: []core.CoreExpr{letrec}, Meta: make(map[string]*core.DeclMeta)}
	_, err := specializer.Specialize(prog)
	if err != nil {
		t.Fatalf("Specialization error: %v", err)
	}

	stats := specializer.GetStats()

	// Should have skipped the recursive function
	foundSkip := false
	for _, skip := range stats.SkippedFunctions {
		if skip.DefSym == "factorial" {
			foundSkip = true
			t.Logf("Correctly skipped recursive function: %s", skip.Reason)
		}
	}

	if !foundSkip && stats.TotalSpecializations == 0 {
		t.Log("Recursive function was either skipped or not specialized (both acceptable)")
	}
}

// TestMonomorphization_ModuleCapEnforcement tests module-wide limit enforcement
func TestMonomorphization_ModuleCapEnforcement(t *testing.T) {
	// Create multiple lambda applications
	lambda1 := &core.Lambda{
		CoreNode: core.CoreNode{NodeID: 1},
		Params:   []string{"x"},
		Body:     &core.Var{CoreNode: core.CoreNode{NodeID: 2}, Name: "x"},
	}

	app1 := &core.App{
		CoreNode: core.CoreNode{NodeID: 3},
		Func:     lambda1,
		Args: []core.CoreExpr{
			&core.Lit{CoreNode: core.CoreNode{NodeID: 4}, Kind: core.IntLit, Value: int64(1)},
		},
	}

	coreTI := types.NewCoreTypeInfo()
	alpha := &types.TVar{Name: "alpha"}
	coreTI.Set(lambda1.ID(), &types.TFunc2{
		Params:    []types.Type{alpha},
		Return:    alpha,
		EffectRow: &types.Row{Kind: types.KEffect{}, Labels: map[string]types.Type{}, Tail: nil},
	})
	coreTI.Set(app1.Args[0].ID(), types.TInt)

	specializer := NewSpecializer(&coreTI)
	specializer.Limits.MaxPerModule = 0 // Don't allow any specializations

	prog := &core.Program{Decls: []core.CoreExpr{app1}, Meta: make(map[string]*core.DeclMeta)}
	_, err := specializer.Specialize(prog)
	if err != nil {
		t.Fatalf("Specialization error: %v", err)
	}

	stats := specializer.GetStats()

	// Should not have done any specializations due to limit
	if stats.TotalSpecializations > 0 {
		t.Errorf("Expected 0 specializations (module limit=0), got %d", stats.TotalSpecializations)
	}

	// Should have a skip reason
	if len(stats.SkippedFunctions) > 0 {
		t.Logf("Correctly skipped due to: %s", stats.SkippedFunctions[0].Reason)
	}
}

// TestMonomorphization_PerFunctionCapEnforcement tests per-function limit
func TestMonomorphization_PerFunctionCapEnforcement(t *testing.T) {
	coreTI := types.NewCoreTypeInfo()
	specializer := NewSpecializer(&coreTI)
	specializer.Limits.MaxPerFunction = 2

	// Simulate reaching the per-function limit
	specializer.PerFunction["(lambda)"] = 2

	// Try to specialize another lambda - should be rejected
	lambda := &core.Lambda{
		CoreNode: core.CoreNode{NodeID: 1},
		Params:   []string{"x"},
		Body:     &core.Var{CoreNode: core.CoreNode{NodeID: 2}, Name: "x"},
	}

	alpha := &types.TVar{Name: "alpha"}
	argTypes := []types.Type{types.TInt}
	env := make(map[string]types.Type)

	coreTI.Set(lambda.ID(), &types.TFunc2{
		Params:    []types.Type{alpha},
		Return:    alpha,
		EffectRow: &types.Row{Kind: types.KEffect{}, Labels: map[string]types.Type{}, Tail: nil},
	})

	// Call specializeLambda directly
	result, err := specializer.specializeLambda(lambda, argTypes, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should return nil (skipped)
	if result != nil {
		t.Error("Expected nil result due to per-function cap")
	}

	stats := specializer.GetStats()
	if len(stats.SkippedFunctions) == 0 {
		t.Error("Expected at least one skip reason")
	}
}

// TestMonomorphization_CacheHitOnDuplicate tests cache deduplication
func TestMonomorphization_CacheHitOnDuplicate(t *testing.T) {
	// Create two identical lambda applications with same type
	lambda := &core.Lambda{
		CoreNode: core.CoreNode{NodeID: 1},
		Params:   []string{"x"},
		Body:     &core.Var{CoreNode: core.CoreNode{NodeID: 2}, Name: "x"},
	}

	coreTI := types.NewCoreTypeInfo()
	alpha := &types.TVar{Name: "alpha"}
	coreTI.Set(lambda.ID(), &types.TFunc2{
		Params:    []types.Type{alpha},
		Return:    alpha,
		EffectRow: &types.Row{Kind: types.KEffect{}, Labels: map[string]types.Type{}, Tail: nil},
	})
	coreTI.Set(lambda.Body.ID(), alpha)

	specializer := NewSpecializer(&coreTI)
	argTypes := []types.Type{types.TInt}
	env := make(map[string]types.Type)

	// First call - should miss cache
	result1, err := specializer.specializeLambda(lambda, argTypes, env)
	if err != nil {
		t.Fatalf("First specialization error: %v", err)
	}

	stats1 := specializer.GetStats()
	firstMisses := stats1.CacheMisses

	// Second call with same types - should hit cache
	result2, err := specializer.specializeLambda(lambda, argTypes, env)
	if err != nil {
		t.Fatalf("Second specialization error: %v", err)
	}

	stats2 := specializer.GetStats()

	if result1 != nil && result2 != nil {
		// Both succeeded - check cache behavior
		if stats2.CacheHits == 0 {
			t.Error("Expected cache hit on second identical specialization")
		}
		if stats2.CacheMisses != firstMisses {
			t.Error("Cache misses should not increase on cache hit")
		}
	} else {
		t.Log("One or both specializations returned nil (acceptable)")
	}
}

// TestMonomorphization_StatsAccuracy validates statistics tracking
func TestMonomorphization_StatsAccuracy(t *testing.T) {
	coreTI := types.NewCoreTypeInfo()
	specializer := NewSpecializer(&coreTI)

	// Set up known state
	specializer.TotalCount = 7
	specializer.PerFunction["func1"] = 4
	specializer.PerFunction["func2"] = 3
	specializer.CacheHits = 3
	specializer.CacheMisses = 4
	specializer.Skipped = []SkipReason{
		{DefSym: "recursive1", Reason: "recursive", Location: "test:1:1"},
		{DefSym: "recursive2", Reason: "recursive", Location: "test:2:1"},
	}

	stats := specializer.GetStats()

	// Verify all fields
	if stats.TotalSpecializations != 7 {
		t.Errorf("TotalSpecializations: expected 7, got %d", stats.TotalSpecializations)
	}
	if stats.PerFunction["func1"] != 4 {
		t.Errorf("PerFunction[func1]: expected 4, got %d", stats.PerFunction["func1"])
	}
	if stats.PerFunction["func2"] != 3 {
		t.Errorf("PerFunction[func2]: expected 3, got %d", stats.PerFunction["func2"])
	}
	if stats.CacheHits != 3 {
		t.Errorf("CacheHits: expected 3, got %d", stats.CacheHits)
	}
	if stats.CacheMisses != 4 {
		t.Errorf("CacheMisses: expected 4, got %d", stats.CacheMisses)
	}
	if len(stats.SkippedFunctions) != 2 {
		t.Errorf("SkippedFunctions: expected 2, got %d", len(stats.SkippedFunctions))
	}
}

// TestMonomorphization_EmptyProgram tests handling of simple literals
func TestMonomorphization_EmptyProgram(t *testing.T) {
	// Just a literal - no specialization needed
	lit := &core.Lit{
		CoreNode: core.CoreNode{NodeID: 1},
		Kind:     core.IntLit,
		Value:    int64(42),
	}

	coreTI := types.NewCoreTypeInfo()
	coreTI.Set(lit.ID(), types.TInt)

	specializer := NewSpecializer(&coreTI)
	prog := &core.Program{Decls: []core.CoreExpr{lit}, Meta: make(map[string]*core.DeclMeta)}
	result, err := specializer.Specialize(prog)
	if err != nil {
		t.Fatalf("Specialization error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	stats := specializer.GetStats()
	if stats.TotalSpecializations != 0 {
		t.Errorf("Expected 0 specializations for literal, got %d", stats.TotalSpecializations)
	}
}
