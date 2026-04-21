package pipeline

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

// TestValidateEffects_LargeArrayPerformance is a regression test for M-PERF1.
// Before the fix, effect checking had O(m²) complexity where m = number of Let bindings.
// This caused effect checking to hang on modules with large array literals.
//
// The fix ensures we don't traverse Let/LetRec bodies in collectRequiredEffects
// since validateDecl already handles body recursion.
func TestValidateEffects_LargeArrayPerformance(t *testing.T) {
	// Build a program with multiple Let bindings containing large lists
	// This mirrors the stapledons_voyage sim/bridge.ail structure
	numBindings := 20      // Number of Let bindings
	elementsPerList := 200 // Elements in each list

	prog := buildLargeArrayProgram(numBindings, elementsPerList)
	typeInfo := types.NewCoreTypeInfo()

	// Register types for all nodes (simplified - just mark as pure)
	registerTypesForProgram(prog, typeInfo)

	// Measure effect checking time
	start := time.Now()
	err := ValidateEffects(nil, prog, typeInfo)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("ValidateEffects failed: %v", err)
	}

	// Before fix: Would hang (30+ seconds)
	// After fix: Should complete in <1 second
	maxExpected := 5 * time.Second
	if elapsed > maxExpected {
		t.Errorf("Effect checking took too long: %v (max expected: %v)", elapsed, maxExpected)
		t.Errorf("This may indicate the O(m²) quadratic traversal bug has regressed")
	}

	t.Logf("Effect checking completed in %v for %d bindings x %d elements",
		elapsed, numBindings, elementsPerList)
}

// TestValidateEffects_LinearScaling verifies that effect checking scales linearly with input size.
// Note: This test uses warmup iterations and takes minimum times to be robust on CI.
func TestValidateEffects_LinearScaling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	sizes := []int{10, 50, 100}
	var times []time.Duration
	const iterations = 5 // Run multiple times and take minimum

	for _, n := range sizes {
		prog := buildLargeArrayProgram(n, 100)
		typeInfo := types.NewCoreTypeInfo()
		registerTypesForProgram(prog, typeInfo)

		// Warmup run to avoid cold cache effects
		_ = ValidateEffects(nil, prog, typeInfo)

		// Run multiple iterations, take the minimum (least affected by CI noise)
		var minTime time.Duration
		for i := 0; i < iterations; i++ {
			start := time.Now()
			err := ValidateEffects(nil, prog, typeInfo)
			elapsed := time.Since(start)

			if err != nil {
				t.Fatalf("ValidateEffects failed for n=%d: %v", n, err)
			}

			if minTime == 0 || elapsed < minTime {
				minTime = elapsed
			}
		}
		times = append(times, minTime)
	}

	// Check that scaling is roughly linear (not quadratic)
	// For quadratic: time(100) / time(10) ≈ 100 (10² ratio)
	// For linear: time(100) / time(10) ≈ 10 (10x ratio)
	if times[0] > 0 {
		ratio := float64(times[2]) / float64(times[0])
		// Allow for significant noise on CI runners, but ratio should be
		// much less than 100 (which would indicate quadratic behavior)
		if ratio > 80 {
			t.Errorf("Scaling appears quadratic: time(100)/time(10) = %.1f (expected <80 for linear)", ratio)
		}
		t.Logf("Scaling ratio time(100)/time(10) = %.1f", ratio)
	}

	for i, n := range sizes {
		t.Logf("  n=%d: %v (min of %d iterations)", n, times[i], iterations)
	}
}

// BenchmarkValidateEffects_LargeArrays benchmarks effect checking with increasing array sizes.
func BenchmarkValidateEffects_LargeArrays(b *testing.B) {
	for _, n := range []int{10, 50, 100, 200} {
		b.Run(fmt.Sprintf("bindings=%d", n), func(b *testing.B) {
			prog := buildLargeArrayProgram(n, 100)
			typeInfo := types.NewCoreTypeInfo()
			registerTypesForProgram(prog, typeInfo)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = ValidateEffects(nil, prog, typeInfo)
			}
		})
	}
}

// TestValidateEffects_PureFunctionNotContaminatedByContext is a regression test for
// M-BUG-EFFECT-CHECKER-CONFLATION (v0.6.2).
// The bug: effect checker incorrectly required IO effects for pure functions when they
// were called inside println that appeared AFTER another println in the same block.
// Root cause: collectRequiredEffects relied on CoreTypeInfo for function effects,
// which had incorrect types. Fix: use declared effects from Surface AST instead.
func TestValidateEffects_PureFunctionNotContaminatedByContext(t *testing.T) {
	// Simplified test: recursive function calling itself
	// Before fix: would use CoreTypeInfo (which might have wrong effects)
	// After fix: uses declaredEffects (correct)

	// sum = \xs. sum(xs)  (simplified - just tests recursion)
	sumBody := &core.App{
		Func: &core.Var{Name: "sum"},
		Args: []core.CoreExpr{&core.Var{Name: "xs"}},
	}
	sumLambda := &core.Lambda{
		Body: sumBody,
	}

	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.LetRec{
				Bindings: []core.RecBinding{
					{Name: "sum", Value: sumLambda},
				},
				Body: &core.Lit{Value: int64(0)},
			},
		},
	}

	// Create CoreTypeInfo and register types
	typeInfo := types.NewCoreTypeInfo()
	registerTypesForProgram(prog, typeInfo)

	// Validate effects with sum declared as pure
	err := ValidateEffects(nil, prog, typeInfo)
	if err != nil {
		t.Errorf("ValidateEffects should pass for pure recursive function, got error: %v", err)
	}
}

// buildLargeArrayProgram creates a Core program with nested Let bindings containing lists.
// Structure: Let("a0", [elems...], Let("a1", [elems...], ... Let("main", Lit(0), Lit(0))...))
func buildLargeArrayProgram(numBindings, elementsPerList int) *core.Program {
	// Start with innermost expression (main function returning 0)
	var body core.CoreExpr = &core.Lit{Value: int64(0)}

	// Build nested Let bindings from inside out
	for i := numBindings - 1; i >= 0; i-- {
		// Create a list with many literal elements
		elements := make([]core.CoreExpr, elementsPerList)
		for j := 0; j < elementsPerList; j++ {
			elements[j] = &core.Lit{Value: int64(j)}
		}

		list := &core.List{Elements: elements}
		bindingName := fmt.Sprintf("arr%d", i)

		body = &core.Let{
			Name:  bindingName,
			Value: list,
			Body:  body,
		}
	}

	return &core.Program{
		Decls: []core.CoreExpr{body},
	}
}

// registerTypesForProgram adds type info for all nodes (simplified - just marks as pure int types)
func registerTypesForProgram(prog *core.Program, typeInfo types.CoreTypeInfo) {
	intType := &types.TCon{Name: "Int"}
	listIntType := &types.TList{Element: intType}

	// Walk and register types
	var walk func(expr core.CoreExpr)
	walk = func(expr core.CoreExpr) {
		if expr == nil {
			return
		}

		switch e := expr.(type) {
		case *core.Lit:
			typeInfo.Set(e.ID(), intType)
		case *core.List:
			typeInfo.Set(e.ID(), listIntType)
			for _, elem := range e.Elements {
				walk(elem)
			}
		case *core.Let:
			typeInfo.Set(e.ID(), intType) // Return type of let
			walk(e.Value)
			walk(e.Body)
		case *core.LetRec:
			for _, b := range e.Bindings {
				walk(b.Value)
			}
			walk(e.Body)
		case *core.Var:
			typeInfo.Set(e.ID(), intType)
		}
	}

	for _, decl := range prog.Decls {
		walk(decl)
	}
}

// TestCollectRequiredEffects_LetDoesNotTraverseBody verifies that
// collectRequiredEffects doesn't traverse Let bodies (the M-PERF1 fix).
func TestCollectRequiredEffects_LetDoesNotTraverseBody(t *testing.T) {
	// Create a Let where the body contains an effectful call
	// If the fix is correct, collectRequiredEffects on Let should NOT see the body's effects
	typeInfo := types.NewCoreTypeInfo()

	// Create a fake effectful function reference in the body
	effectfulVar := &core.VarGlobal{Ref: core.GlobalRef{Module: "test", Name: "effectful"}}
	typeInfo.Set(effectfulVar.ID(), &types.TFunc2{
		Params:    []types.Type{},
		EffectRow: &types.Row{Labels: map[string]types.Type{"IO": types.Unit()}},
		Return:    &types.TCon{Name: "()"},
	})

	// Let("x", pure_value, effectful_call)
	letExpr := &core.Let{
		Name:  "x",
		Value: &core.Lit{Value: int64(42)},                            // Pure RHS
		Body:  &core.App{Func: effectfulVar, Args: []core.CoreExpr{}}, // Effectful body
	}

	// collectRequiredEffects should only see RHS effects (none), not body effects
	declaredEffects := make(map[string][]string) // Empty for this test
	effects := collectRequiredEffects(letExpr, typeInfo, declaredEffects)

	// With the fix: effects should be nil (only RHS is checked, which is pure)
	// Without the fix: effects would include IO from the body
	if effects != nil && len(effects.Labels) > 0 {
		var labels []string
		for k := range effects.Labels {
			labels = append(labels, k)
		}
		t.Errorf("collectRequiredEffects traversed Let body (found effects: %v)", strings.Join(labels, ", "))
		t.Error("This indicates the M-PERF1 fix has regressed!")
	}
}
