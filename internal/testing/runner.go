package testing

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/eval"
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
		// Non-inline tests (test "name" { ... } blocks)
		// For now, skip these - they're less common
		result.Status = StatusSkip
		result.Error = "Named test blocks not yet implemented"
	}

	result.Duration = time.Since(start)
	return result
}

// runProperty executes a property-based test with generators and shrinking.
func (r *Runner) runProperty(propCase PropertyCase) PropertyResult {
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
