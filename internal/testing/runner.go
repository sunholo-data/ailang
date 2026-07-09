package testing

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/eval"
)

// Runner executes tests and properties.
type Runner struct {
	modulePath string
	executor   *Executor
}

// NewRunner creates a new test runner.
func NewRunner(modulePath string) *Runner {
	return &Runner{
		modulePath: modulePath,
		executor:   NewExecutor(modulePath),
	}
}

// RunSuite executes all tests in a test suite and returns aggregated results.
func (r *Runner) RunSuite(suite *TestSuite) *SuiteResult {
	result := NewSuiteResult(suite.ModulePath)

	// Run all tests
	for _, testCase := range suite.Tests {
		testResult := r.runTest(testCase)
		result.AddTestResult(testResult)
	}

	// Run all properties (basic implementation - full property testing in Days 6-8)
	for _, propCase := range suite.Properties {
		propResult := r.runProperty(propCase)
		result.AddPropertyResult(propResult)
	}

	return result
}

// runTest executes a single test case.
func (r *Runner) runTest(testCase TestCase) TestResult {
	start := time.Now()

	result := TestResult{
		Name:     testCase.Name,
		Location: testCase.Location.String(),
	}

	// For inline tests, use the harness-based approach
	if testCase.IsInline {
		// Get source file from executor (must be set via SetSourceFile)
		if r.executor.sourceFile == nil {
			result.Status = StatusFail
			result.Error = "source file not set on executor (call SetSourceFile first)"
			result.Duration = time.Since(start)
			return result
		}

		// Extract function binding from source
		binding, err := r.executor.ExtractFunctionBinding(testCase.FunctionCtx, r.executor.sourceFile)
		if err != nil {
			result.Status = StatusFail
			result.Error = fmt.Sprintf("failed to extract function binding: %v", err)
			result.Duration = time.Since(start)
			return result
		}

		// Check if function has cross-function dependencies
		// If so, use cluster evaluation to include all dependencies
		var actualsTuple *eval.TupleValue
		cluster, coreProg, clusterErr := r.executor.ExtractPureClusterForFunction(testCase.FunctionCtx, r.executor.sourceFile)
		if clusterErr == nil && cluster != nil && cluster.HasDependencies() {
			// Function has dependencies - use cluster harness
			actualsTuple, err = r.executor.EvaluateInlineTestsWithCluster(testCase.FunctionCtx, []TestCase{testCase}, coreProg)
			if err != nil {
				result.Status = StatusFail
				result.Error = fmt.Sprintf("cluster harness evaluation failed: %v", err)
				result.Duration = time.Since(start)
				return result
			}
		} else {
			// No dependencies or cluster extraction failed - use single-binding harness
			actualsTuple, err = r.executor.EvaluateInlineTestsWithHarness(*binding, []TestCase{testCase})
			if err != nil {
				result.Status = StatusFail
				result.Error = fmt.Sprintf("harness evaluation failed: %v", err)
				result.Duration = time.Since(start)
				return result
			}
		}

		// Compare each actual to expected
		for i, expr := range testCase.Body {
			// Expected format: Tuple with 2 elements (input, expected)
			tuple, ok := expr.(*ast.Tuple)
			if !ok || len(tuple.Elements) != 2 {
				result.Status = StatusFail
				result.Error = fmt.Sprintf("inline test %d: expected (input, expected) tuple, got %T", i, expr)
				result.Duration = time.Since(start)
				return result
			}

			expected := tuple.Elements[1]

			// Get actual value from tuple
			if i >= len(actualsTuple.Elements) {
				result.Status = StatusFail
				result.Error = fmt.Sprintf("test %d: harness returned %d values, expected at least %d", i, len(actualsTuple.Elements), i+1)
				result.Duration = time.Since(start)
				return result
			}
			actualValue := actualsTuple.Elements[i]

			// Evaluate expected expression (should be a simple literal)
			expectedValue, err := r.executor.EvaluateLiteral(expected)
			if err != nil {
				result.Status = StatusFail
				result.Error = fmt.Sprintf("test %d: failed to evaluate expected: %v", i, err)
				result.Duration = time.Since(start)
				return result
			}

			// Compare values
			if !r.executor.CompareValues(actualValue, expectedValue) {
				result.Status = StatusFail
				result.Error = fmt.Sprintf("test %d: expected %v, got %v", i, expectedValue, actualValue)
				result.Duration = time.Since(start)
				return result
			}
		}

		// All tests passed
		result.Status = StatusPass
	} else {
		// Named test blocks: test "name" { <expr> }
		// Each expression in the body must evaluate to bool true.
		// false → FAIL; runtime error → FAIL with error text.
		result = r.runNamedTest(testCase, start)
	}

	result.Duration = time.Since(start)
	return result
}

// runNamedTest executes a named test block: test "name" { <expr> }
//
// Reuses the module-scope evaluation path (EvaluateExpression) from the
// inline-test core-evaluation machinery (v0.4.7). The body is treated as a
// sequence of expressions; the LAST expression must evaluate to bool true for
// the test to pass. Earlier expressions are evaluated for side-effects only
// (they are discarded — named test bodies are pure by contract).
//
// Pass contract: final expression evaluates to *eval.BoolValue{true}.
// false      → StatusFail, error = "expected true, got false"
// non-bool   → StatusFail, error = "expected bool result, got <T>"
// eval error → StatusFail, error = <error text>
func (r *Runner) runNamedTest(testCase TestCase, start time.Time) TestResult {
	result := TestResult{
		Name:     testCase.Name,
		Location: testCase.Location.String(),
	}

	if len(testCase.Body) == 0 {
		result.Status = StatusFail
		result.Error = "named test block has empty body"
		result.Duration = time.Since(start)
		return result
	}

	// Evaluate the body expressions via the module-scope elaboration path.
	// EvaluateNamedTestBodyExprs returns the value of the last expression.
	// recover() ensures a panic in the printer or evaluator fails THIS test
	// instead of crashing the whole runner (defence-in-depth).
	var val eval.Value
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("internal panic evaluating named test body: %v", r)
			}
		}()
		val, err = r.executor.EvaluateNamedTestBodyExprs(testCase.Body)
	}()
	if err != nil {
		result.Status = StatusFail
		result.Error = err.Error()
		result.Duration = time.Since(start)
		return result
	}

	// Pass contract: last expression must evaluate to bool true.
	boolVal, ok := val.(*eval.BoolValue)
	if !ok {
		result.Status = StatusFail
		result.Error = fmt.Sprintf("expected bool result, got %T", val)
		result.Duration = time.Since(start)
		return result
	}
	if !boolVal.Value {
		result.Status = StatusFail
		result.Error = "expected true, got false"
		result.Duration = time.Since(start)
		return result
	}

	result.Status = StatusPass
	result.Duration = time.Since(start)
	return result
}

// runProperty executes a property-based test with generators and shrinking.
func (r *Runner) runProperty(propCase PropertyCase) PropertyResult {
	// M-DX26 Phase 5: ensures/requires clauses route through dedicated harnesses
	// that pull the already-lowered Core predicate from Meta.Contracts.
	// forall-style properties remain on the broken EvaluateExpression
	// source-synthesis path until the broader Option A refactor.
	switch propCase.Property.Kind {
	case ast.EnsuresKind:
		return r.runEnsuresProperty(propCase)
	case ast.RequiresKind:
		return r.runRequiresProperty(propCase)
	}

	start := time.Now()

	result := PropertyResult{
		Name:     propCase.Name,
		Location: propCase.Location.String(),
		TestsRun: 0,
	}

	// Number of test cases to generate per property
	const numTests = 100

	// Create generators for each binder based on type
	generators := make([]Generator, len(propCase.Property.Binders))
	shrinkers := make([]Shrinker, len(propCase.Property.Binders))

	for i, binder := range propCase.Property.Binders {
		gen, shrink := r.createGeneratorForType(binder.Type)
		if gen == nil {
			result.Status = StatusSkip
			result.Error = fmt.Sprintf("no generator for type %v", binder.Type)
			result.Duration = time.Since(start)
			return result
		}
		generators[i] = gen
		shrinkers[i] = shrink
	}

	// Run property tests
	config := DefaultConfig()
	rng := newRNG(config.Seed)

	for testNum := 0; testNum < numTests; testNum++ {
		// Generate values for all forall parameters
		generatedValues := make([]eval.Value, len(generators))
		for i, gen := range generators {
			generatedValues[i] = gen.Generate(rng)
		}

		// Bind generated values to property expression
		boundExpr := r.bindPropertyValues(propCase.Property, generatedValues)

		// Evaluate the property expression (should return bool)
		resultValue, err := r.executor.EvaluateExpression(boundExpr)
		if err != nil {
			result.Status = StatusFail
			result.Error = fmt.Sprintf("test %d: evaluation failed: %v", testNum, err)
			result.TestsRun = testNum + 1
			result.Duration = time.Since(start)
			return result
		}

		// Check if result is a boolean
		boolVal, ok := resultValue.(*eval.BoolValue)
		if !ok {
			result.Status = StatusFail
			result.Error = fmt.Sprintf("test %d: property must return bool, got %T", testNum, resultValue)
			result.TestsRun = testNum + 1
			result.Duration = time.Since(start)
			return result
		}

		// If property fails, try to shrink to minimal counterexample
		if !boolVal.Value {
			counterexample := r.shrinkCounterexample(propCase.Property, generatedValues, shrinkers)
			result.Status = StatusFail
			result.Error = fmt.Sprintf("property failed on input: %v", counterexample)
			result.TestsRun = testNum + 1
			result.Duration = time.Since(start)
			return result
		}

		result.TestsRun++
	}

	// All tests passed
	result.Status = StatusPass
	result.Duration = time.Since(start)
	return result
}

// runEnsuresProperty executes an ensures-clause property test.
//
// For each iteration:
//  1. Generate a value for each function parameter (using the existing per-type generators).
//  2. Build a Core harness that calls the function with those values, binds `result`,
//     and evaluates the predicate.
//  3. If the predicate evaluates to false, report a counterexample and stop.
//
// Out of scope here: shrinking the counterexample (existing shrinkCounterexample plumbing
// is wired for forall-binders, not function parameters; follow-up work).
func (r *Runner) runEnsuresProperty(propCase PropertyCase) PropertyResult {
	start := time.Now()

	result := PropertyResult{
		Name:     propCase.Name,
		Location: propCase.Location.String(),
		TestsRun: 0,
	}

	if propCase.Function == nil {
		result.Status = StatusSkip
		result.Error = "ensures property has no function context (top-level ensures not supported)"
		result.Duration = time.Since(start)
		return result
	}

	if r.executor.sourceFile == nil {
		result.Status = StatusFail
		result.Error = "source file not set on executor (call SetSourceFile first)"
		result.Duration = time.Since(start)
		return result
	}

	// Extract the Core function binding (re-uses the inline-tests path).
	// This also caches the elaborated + lowered DeclMeta on the executor.
	binding, err := r.executor.ExtractFunctionBinding(propCase.FunctionCtx, r.executor.sourceFile)
	if err != nil {
		result.Status = StatusFail
		result.Error = fmt.Sprintf("failed to extract function binding for %s: %v", propCase.FunctionCtx, err)
		result.Duration = time.Since(start)
		return result
	}

	// M-DX26 Phase 5.1: Pull the *already-lowered* ensures predicate from Core.Meta
	// instead of converting the surface AST predicate ourselves. This lets arithmetic
	// operators (`+`, `*`) in predicates work — they get rewritten to typed dictionary
	// calls during the standard OpLowering pass that runs over Meta.Contracts.
	loweredPredicate := r.findLoweredContractPredicate(propCase, ast.EnsuresKind, core.EnsuresKind)
	if loweredPredicate == nil {
		result.Status = StatusFail
		result.Error = fmt.Sprintf("could not locate lowered ensures predicate for %s", propCase.FunctionCtx)
		result.Duration = time.Since(start)
		return result
	}

	// Build generators per parameter type. We do NOT use Property.Binders here —
	// ensures has no forall binders; the values flow into the function call.
	params := propCase.Function.Params
	generators := make([]Generator, len(params))
	for i, p := range params {
		gen, _ := r.createGeneratorForType(p.Type)
		if gen == nil {
			result.Status = StatusSkip
			result.Error = fmt.Sprintf("no generator for parameter %s: %v", p.Name, p.Type)
			result.Duration = time.Since(start)
			return result
		}
		generators[i] = gen
	}

	const numTests = 100
	config := DefaultConfig()
	rng := newRNG(config.Seed)

	for testNum := 0; testNum < numTests; testNum++ {
		generatedValues := make([]eval.Value, len(generators))
		ensuresParams := make([]EnsuresParam, len(generators))
		for i, gen := range generators {
			v := gen.Generate(rng)
			generatedValues[i] = v
			ensuresParams[i] = EnsuresParam{
				Name:  params[i].Name,
				Value: astExprToCore(r.valueToLiteral(v)),
			}
		}

		boolValueRaw, err := r.executor.EvaluateEnsuresHarnessFromCore(*binding, ensuresParams, loweredPredicate)
		if err != nil {
			result.Status = StatusFail
			result.Error = fmt.Sprintf("test %d: %v", testNum, err)
			result.TestsRun = testNum + 1
			result.Duration = time.Since(start)
			return result
		}

		boolVal, ok := boolValueRaw.(*eval.BoolValue)
		if !ok {
			result.Status = StatusFail
			result.Error = fmt.Sprintf("test %d: ensures predicate must return bool, got %T", testNum, boolValueRaw)
			result.TestsRun = testNum + 1
			result.Duration = time.Since(start)
			return result
		}

		if !boolVal.Value {
			result.Status = StatusFail
			result.Error = fmt.Sprintf("ensures violated for input: %s", formatEnsuresInputs(params, generatedValues))
			result.TestsRun = testNum + 1
			result.Duration = time.Since(start)
			return result
		}

		result.TestsRun++
	}

	result.Status = StatusPass
	result.Duration = time.Since(start)
	return result
}

// runRequiresProperty executes a requires-clause property test.
//
// Unlike `ensures`, `requires` runs *before* the function would be called and
// references parameters only (no `result`). For each iteration we generate
// parameter values and evaluate the lowered predicate.
//
// A `requires` clause that evaluates to false on a generated input is **not** a
// function bug — the test is "out of contract" and would normally be discarded
// by a property tester. For now we report it as a Skip with the offending input
// so users can refine their generators. The function being verified is never
// called from this path.
//
// M-DX26 Phase 5.2.
func (r *Runner) runRequiresProperty(propCase PropertyCase) PropertyResult {
	start := time.Now()

	result := PropertyResult{
		Name:     propCase.Name,
		Location: propCase.Location.String(),
		TestsRun: 0,
	}

	if propCase.Function == nil {
		result.Status = StatusSkip
		result.Error = "requires property has no function context (top-level requires not supported)"
		result.Duration = time.Since(start)
		return result
	}

	if r.executor.sourceFile == nil {
		result.Status = StatusFail
		result.Error = "source file not set on executor (call SetSourceFile first)"
		result.Duration = time.Since(start)
		return result
	}

	// We still need ExtractFunctionBinding to populate LastDeclMeta with the
	// elaborated + lowered Contracts. The returned binding itself is unused for
	// requires (we never call the function).
	if _, err := r.executor.ExtractFunctionBinding(propCase.FunctionCtx, r.executor.sourceFile); err != nil {
		result.Status = StatusFail
		result.Error = fmt.Sprintf("failed to extract function binding for %s: %v", propCase.FunctionCtx, err)
		result.Duration = time.Since(start)
		return result
	}

	loweredPredicate := r.findLoweredContractPredicate(propCase, ast.RequiresKind, core.RequiresKind)
	if loweredPredicate == nil {
		result.Status = StatusFail
		result.Error = fmt.Sprintf("could not locate lowered requires predicate for %s", propCase.FunctionCtx)
		result.Duration = time.Since(start)
		return result
	}

	params := propCase.Function.Params
	generators := make([]Generator, len(params))
	for i, p := range params {
		gen, _ := r.createGeneratorForType(p.Type)
		if gen == nil {
			result.Status = StatusSkip
			result.Error = fmt.Sprintf("no generator for parameter %s: %v", p.Name, p.Type)
			result.Duration = time.Since(start)
			return result
		}
		generators[i] = gen
	}

	const numTests = 100
	config := DefaultConfig()
	rng := newRNG(config.Seed)

	for testNum := 0; testNum < numTests; testNum++ {
		generatedValues := make([]eval.Value, len(generators))
		harnessParams := make([]EnsuresParam, len(generators))
		for i, gen := range generators {
			v := gen.Generate(rng)
			generatedValues[i] = v
			harnessParams[i] = EnsuresParam{
				Name:  params[i].Name,
				Value: astExprToCore(r.valueToLiteral(v)),
			}
		}

		boolValueRaw, err := r.executor.EvaluateRequiresHarnessFromCore(harnessParams, loweredPredicate)
		if err != nil {
			result.Status = StatusFail
			result.Error = fmt.Sprintf("test %d: %v", testNum, err)
			result.TestsRun = testNum + 1
			result.Duration = time.Since(start)
			return result
		}

		boolVal, ok := boolValueRaw.(*eval.BoolValue)
		if !ok {
			result.Status = StatusFail
			result.Error = fmt.Sprintf("test %d: requires predicate must return bool, got %T", testNum, boolValueRaw)
			result.TestsRun = testNum + 1
			result.Duration = time.Since(start)
			return result
		}

		if !boolVal.Value {
			// Out-of-contract input — surface it so users can refine generators.
			// We mark the property Skipped (not Fail) because random inputs that
			// violate `requires` aren't a function bug.
			result.Status = StatusSkip
			result.Error = fmt.Sprintf("requires not satisfied by random input (consider tighter generators): %s", formatEnsuresInputs(params, generatedValues))
			result.TestsRun = testNum + 1
			result.Duration = time.Since(start)
			return result
		}

		result.TestsRun++
	}

	result.Status = StatusPass
	result.Duration = time.Since(start)
	return result
}

// findLoweredContractPredicate locates the lowered Core predicate for a
// requires- or ensures-PropertyCase by counting its position among same-kind
// properties on the function and indexing into the cached DeclMeta's same-kind
// Contracts.
//
// The elaborator skips forall properties when emitting Contracts (file_funcs.go),
// so requires/ensures contracts are emitted in their original source order.
// Within same-kind entries, the i-th one in propCase.Function.Properties matches
// the i-th same-kind Contract in DeclMeta.Contracts.
func (r *Runner) findLoweredContractPredicate(propCase PropertyCase, astKind ast.ContractKind, coreKind core.ContractKind) core.CoreExpr {
	if propCase.Function == nil {
		return nil
	}
	meta := r.executor.LastDeclMeta(propCase.FunctionCtx)
	if meta == nil {
		return nil
	}

	// Count which same-kind-index this PropertyCase has among the function's properties.
	contractIndex := -1
	target := propCase.Property
	count := 0
	for _, p := range propCase.Function.Properties {
		if p.Kind != astKind {
			continue
		}
		if p == target {
			contractIndex = count
			break
		}
		count++
	}
	if contractIndex < 0 {
		return nil
	}

	// Walk DeclMeta.Contracts taking the n-th same-kind one.
	count = 0
	for _, c := range meta.Contracts {
		if c.Kind != coreKind {
			continue
		}
		if count == contractIndex {
			return c.Expr
		}
		count++
	}
	return nil
}

// formatEnsuresInputs renders generated parameter values as `name=value, ...` for counterexample reporting.
func formatEnsuresInputs(params []*ast.Param, values []eval.Value) string {
	parts := make([]string, len(values))
	for i, v := range values {
		name := fmt.Sprintf("arg%d", i)
		if i < len(params) && params[i] != nil {
			name = params[i].Name
		}
		parts[i] = fmt.Sprintf("%s=%v", name, v)
	}
	return strings.Join(parts, ", ")
}

// RunTestsFromFile is a convenience function that parses, collects, and runs tests from a file.
func RunTestsFromFile(filePath string, ast *ast.File) (*SuiteResult, error) {
	// Collect tests from AST
	collector := NewCollector(filePath)
	suite := collector.Collect(ast)

	// Run tests
	runner := NewRunner(filePath)
	// Provide source file context to executor
	runner.executor.SetSourceFile(ast)
	result := runner.RunSuite(suite)

	return result, nil
}

// createGeneratorForType creates a generator and shrinker for the given type.
func (r *Runner) createGeneratorForType(typ ast.Type) (Generator, Shrinker) {
	// Check for simple types
	if simpleType, ok := typ.(*ast.SimpleType); ok {
		switch simpleType.Name {
		case "int":
			config := DefaultConfig()
			return NewIntGenerator(config.MinInt, config.MaxInt), NewIntShrinker()
		case "float":
			config := DefaultConfig()
			return NewFloatGenerator(config.MinFloat, config.MaxFloat), NewFloatShrinker()
		case "bool":
			return NewBoolGenerator(), NewNoOpShrinker()
		case "string":
			config := DefaultConfig()
			return NewStringGenerator(0, config.MaxSize, ""), NewStringShrinker()
		}
	}

	// Check for list types [a]
	if listType, ok := typ.(*ast.ListType); ok {
		// Create generator for element type
		elemGen, elemShrink := r.createGeneratorForType(listType.Element)
		if elemGen == nil {
			return nil, nil
		}
		config := DefaultConfig()
		return NewListGenerator(elemGen, 0, config.MaxSize), NewListShrinker(elemShrink)
	}

	// Unsupported type
	return nil, nil
}

// bindPropertyValues binds generated values to forall parameters in a property expression.
// For: forall(x: int, y: int) => x + y == y + x
// With values: [5, 10]
// Returns: let x = 5 in let y = 10 in (x + y == y + x)
func (r *Runner) bindPropertyValues(property *ast.Property, values []eval.Value) ast.Expr {
	expr := property.Expr

	// Bind in reverse order (innermost first)
	for i := len(property.Binders) - 1; i >= 0; i-- {
		binder := property.Binders[i]
		value := values[i]

		// Convert eval.Value to ast.Expr (literal)
		valueLit := r.valueToLiteral(value)

		// Wrap in let binding
		expr = &ast.Let{
			Name:  binder.Name,
			Value: valueLit,
			Body:  expr,
			Pos:   property.Pos,
		}
	}

	return expr
}

// valueToLiteral converts an eval.Value to an ast.Literal expression.
func (r *Runner) valueToLiteral(value eval.Value) ast.Expr {
	switch v := value.(type) {
	case *eval.IntValue:
		return &ast.Literal{
			Kind:  ast.IntLit,
			Value: v.Value,
		}
	case *eval.FloatValue:
		return &ast.Literal{
			Kind:  ast.FloatLit,
			Value: v.Value,
		}
	case *eval.BoolValue:
		return &ast.Literal{
			Kind:  ast.BoolLit,
			Value: v.Value,
		}
	case *eval.StringValue:
		return &ast.Literal{
			Kind:  ast.StringLit,
			Value: v.Value,
		}
	case *eval.ListValue:
		// Convert list elements
		elements := make([]ast.Expr, len(v.Elements))
		for i, elem := range v.Elements {
			elements[i] = r.valueToLiteral(elem)
		}
		return &ast.List{Elements: elements}
	default:
		// For unsupported types, return a unit literal
		return &ast.Literal{
			Kind:  ast.UnitLit,
			Value: struct{}{},
		}
	}
}

// shrinkCounterexample finds the minimal counterexample using shrinking.
func (r *Runner) shrinkCounterexample(property *ast.Property, failingValues []eval.Value, shrinkers []Shrinker) []eval.Value {
	// Try shrinking each parameter independently
	minimal := make([]eval.Value, len(failingValues))
	copy(minimal, failingValues)

	for i, shrinker := range shrinkers {
		if shrinker == nil {
			continue
		}

		// Try all shrunk values for this parameter
		shrunkValues := shrinker.Shrink(failingValues[i])
		for _, shrunk := range shrunkValues {
			// Replace this parameter with shrunk value
			testValues := make([]eval.Value, len(minimal))
			copy(testValues, minimal)
			testValues[i] = shrunk

			// Test if property still fails with shrunk value
			boundExpr := r.bindPropertyValues(property, testValues)
			result, err := r.executor.EvaluateExpression(boundExpr)
			if err != nil {
				continue // Skip if evaluation fails
			}

			boolVal, ok := result.(*eval.BoolValue)
			if ok && !boolVal.Value {
				// Property still fails with shrunk value - use this
				minimal[i] = shrunk
				// Continue shrinking this parameter
				break
			}
		}
	}

	return minimal
}

// newRNG creates a random number generator.
func newRNG(seed int64) *rand.Rand {
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	return rand.New(rand.NewSource(seed))
}
