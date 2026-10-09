package testing

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/eval"
)

// Runner executes tests and properties.
type Runner struct {
	modulePath     string
	executor       *Executor
	config         TestConfig
	moduleIdentity string

	// genForType is an injectable generator seam. It exists so tests can
	// substitute a generator that produces a value kind valueToLiteral
	// refuses, making the refusal branches at the three call sites observable
	// (rule 3j: a guard is not a gate until something reds when you remove it).
	// Defaults to createGeneratorForType and is bound in NewRunnerWithConfig.
	genForType func(ast.Type) (Generator, Shrinker)
}

// NewRunner creates a new test runner using the default derived-seed policy.
//
// It is a convenience wrapper over NewRunnerWithConfig: the workspace root is
// inferred from the module file's own directory (SeedModeDerived, master 0) and
// the module identity is left unresolved. CLI code must not use this — it should
// use RunTestsFromFileWithConfig so the module identity is resolved properly.
func NewRunner(modulePath string) *Runner {
	// WorkspaceRoot is deliberately left empty and the original filepath.Abs is
	// gone: this convenience wrapper does not resolve the module identity
	// (moduleIdentity is empty), and WorkspaceRoot is read only by Validate and
	// ResolveModuleIdentity — neither of which this path ever invokes. Computing
	// it here would be dead state with a silent-fallback shape (CLAUDE.md §2);
	// the CLI must use RunTestsFromFileWithConfig so the identity is resolved.
	cfg := TestConfig{
		SeedMode:   SeedModeDerived,
		MasterSeed: 0,
	}
	return NewRunnerWithConfig(modulePath, cfg, "")
}

// NewRunnerWithConfig creates a test runner with an explicit seed config and a
// pre-resolved module identity. It never fails: callers resolve the identity via
// ResolveModuleIdentity before calling.
func NewRunnerWithConfig(modulePath string, cfg TestConfig, moduleIdentity string) *Runner {
	r := &Runner{
		modulePath:     modulePath,
		executor:       NewExecutor(modulePath),
		config:         cfg,
		moduleIdentity: moduleIdentity,
	}
	r.executor.maxRecursionDepth = cfg.MaxRecursionDepth
	r.executor.bytecode = cfg.Bytecode || cfg.StrictBytecode
	r.executor.strictBytecode = cfg.StrictBytecode
	// Bind the generator seam to the built-in derivation after the Runner value
	// exists, so the method value binds to the right receiver. Tests may
	// override r.genForType afterwards to inject a refusal-producing generator.
	r.genForType = r.createGeneratorForType
	return r
}

// propertySeed derives the deterministic seed for a single property from the
// runner's master seed, module identity, and the property name.
func (r *Runner) propertySeed(name string) int64 {
	return DeriveSeedV1(r.config.MasterSeed, r.moduleIdentity, name)
}

// RunSuite executes all tests in a test suite and returns aggregated results.
func (r *Runner) RunSuite(suite *TestSuite) *SuiteResult {
	result := NewSuiteResult(suite.ModulePath)

	// Run all tests. Named tests share one compile (named_batch.go).
	r.executor.prepareNamedTests(suite.Tests, suite.Properties)
	for _, testCase := range suite.Tests {
		testResult := r.runTest(testCase)
		result.AddTestResult(testResult)
	}

	// Run all properties (basic implementation - full property testing in Days 6-8)
	for _, propCase := range suite.Properties {
		propResult := r.runProperty(propCase)
		result.AddPropertyResult(propResult)
	}
	// After the properties: a forall property compiled alone also counts
	// toward deciding whether a batch failure was the harness's fault.
	if f := r.executor.finishNamedTests(); f != nil {
		result.NamedBatchFailures = append(result.NamedBatchFailures, *f)
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
		for _, expr := range testCase.Body {
			if problem := ast.RowExprSupported(expr); problem != nil {
				result.Status = StatusFail
				result.Error = fmt.Sprintf("%s at %s: %s", testCase.Name, problem.Pos, problem)
				result.Duration = time.Since(start)
				return result
			}
		}
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

			// Evaluate expected through the same module-scoped machinery as inputs.
			expectedValue, err := r.executor.EvaluateExpectedExpr(expected)
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
// The body runs as an entry of the file's one batched compile
// (named_batch.go), or alone when the batch is unavailable. The body is a
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
	// forall properties run as a compiled function of their binders
	// (property_batch.go, #624).
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
		Seed:     r.propertySeed(propCase.Name),
	}

	// Number of test cases to generate per property
	const numTests = 100

	// Create generators for each binder based on type
	generators := make([]Generator, len(propCase.Property.Binders))
	shrinkers := make([]Shrinker, len(propCase.Property.Binders))

	for i, binder := range propCase.Property.Binders {
		gen, shrink := r.genForTypeSeam(binder.Type)
		if gen == nil {
			result.Status = StatusSkip
			result.SkipKind = SkipKindNoGenerator
			result.Error = fmt.Sprintf("no generator for type %v", binder.Type)
			result.Duration = time.Since(start)
			return result
		}
		generators[i] = gen
		shrinkers[i] = shrink
	}

	// The predicate is one compiled function of the binders, called once per
	// generated case with the values as arguments (property_batch.go, #624).
	call, err := r.executor.forallCaller(propCase.Property)
	if err != nil {
		result.Status = StatusFail
		result.Error = fmt.Sprintf("property does not compile: %v", err)
		result.Duration = time.Since(start)
		return result
	}
	rng := newRNG(r.propertySeed(propCase.Name))

	for testNum := 0; testNum < numTests; testNum++ {
		result.GeneratedInputs = testNum + 1
		// Generate values for all forall parameters
		generatedValues := make([]eval.Value, len(generators))
		for i, gen := range generators {
			generatedValues[i] = gen.Generate(rng)
		}

		// Evaluate the predicate on this case (should return bool)
		resultValue, err := call(generatedValues)
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
			counterexample := shrinkCounterexample(call, generatedValues, shrinkers)
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
		Seed:     r.propertySeed(propCase.Name),
	}

	if propCase.Function == nil {
		result.Status = StatusSkip
		result.SkipKind = SkipKindUnsupported
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
		gen, _ := r.genForTypeSeam(p.Type)
		if gen == nil {
			result.Status = StatusSkip
			result.SkipKind = SkipKindNoGenerator
			result.Error = fmt.Sprintf("no generator for parameter %s: %v", p.Name, p.Type)
			result.Duration = time.Since(start)
			return result
		}
		generators[i] = gen
	}

	const numTests = 100
	rng := newRNG(r.propertySeed(propCase.Name))

	for testNum := 0; testNum < numTests; testNum++ {
		result.GeneratedInputs = testNum + 1
		generatedValues := make([]eval.Value, len(generators))
		harnessParams := make([]EnsuresParam, len(generators))
		for i, gen := range generators {
			v := gen.Generate(rng)
			generatedValues[i] = v
			lit, err := r.valueToLiteral(v)
			if err != nil {
				result.Status = StatusFail
				result.Error = fmt.Sprintf("test %d: %v", testNum, err)
				result.TestsRun = testNum + 1
				result.Duration = time.Since(start)
				return result
			}
			valueCore, err := astExprToCore(lit)
			if err != nil {
				result.Status = StatusFail
				result.Error = err.Error()
				result.Duration = time.Since(start)
				return result
			}
			harnessParams[i] = EnsuresParam{
				Name:  params[i].Name,
				Value: valueCore,
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
			result.SkipKind = SkipKindOutOfContract
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

// RunTestsFromFileWithConfig parses, collects, and runs tests from a file with an
// explicit seed config. It validates the config, resolves the module identity
// (module-bearing files are path-independent; module-less files are made
// workspace-relative), and stamps the seed metadata onto the result.
func RunTestsFromFileWithConfig(filePath string, file *ast.File, cfg TestConfig) (*SuiteResult, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// Resolve the stable module identity before the Runner exists (D4: one
	// consumption point for WorkspaceRoot).
	declared := ""
	if file.Module != nil {
		declared = file.Module.Path
	}
	identity, err := ResolveModuleIdentity(cfg.WorkspaceRoot, filePath, declared)
	if err != nil {
		return nil, err
	}

	// Collect tests from AST
	collector := NewCollector(filePath)
	suite := collector.Collect(file)

	// Run tests
	runner := NewRunnerWithConfig(filePath, cfg, identity)
	// Provide source file context to executor
	runner.executor.SetSourceFile(file)
	result := runner.RunSuite(suite)
	result.SetSeedMetadata(cfg)
	result.Engine = runner.executor.engine

	return result, nil
}

// RunTestsFromFile is a convenience function that parses, collects, and runs
// tests from a file with the default derived-seed policy (workspace root is the
// module file's own directory). It is a wrapper over RunTestsFromFileWithConfig.
func RunTestsFromFile(filePath string, ast *ast.File) (*SuiteResult, error) {
	abs, err := filepath.Abs(filePath)
	if err != nil {
		// No silent fallback to a non-absolute root: this value feeds the module
		// identity, which feeds the seed (CLAUDE.md §2). Surface the error.
		return nil, err
	}
	cfg := TestConfig{
		WorkspaceRoot: filepath.Dir(abs),
		SeedMode:      SeedModeDerived,
		MasterSeed:    0,
	}
	return RunTestsFromFileWithConfig(filePath, ast, cfg)
}

// genForTypeSeam returns the generator and shrinker for typ, routing through the
// injectable genForType field so tests can substitute a generator that yields a
// value kind valueToLiteral refuses. It falls back to the built-in
// createGeneratorForType for any Runner not constructed via NewRunnerWithConfig.
func (r *Runner) genForTypeSeam(typ ast.Type) (Generator, Shrinker) {
	if r.genForType != nil {
		return r.genForType(typ)
	}
	return r.createGeneratorForType(typ)
}

// maxShrinkCalls bounds shrinking so a pathological shrinker cannot hang a run.
const maxShrinkCalls = 1000

// shrinkCounterexample finds a minimal counterexample: it repeatedly replaces
// a binder with the first shrunk value on which the predicate still returns
// false, until no binder shrinks further (or the call budget is spent). A
// candidate the predicate cannot evaluate (a runtime error, a non-bool) is
// skipped. Deterministic: shrinkers are, and the order is fixed.
func shrinkCounterexample(call propertyCall, failingValues []eval.Value, shrinkers []Shrinker) []eval.Value {
	minimal := make([]eval.Value, len(failingValues))
	copy(minimal, failingValues)

	calls := 0
	for progress := true; progress && calls < maxShrinkCalls; {
		progress = false
		for i, shrinker := range shrinkers {
			if shrinker == nil {
				continue
			}
			for _, shrunk := range shrinker.Shrink(minimal[i]) {
				if calls >= maxShrinkCalls {
					break
				}
				calls++
				testValues := make([]eval.Value, len(minimal))
				copy(testValues, minimal)
				testValues[i] = shrunk

				result, err := call(testValues)
				if err != nil {
					continue
				}
				if boolVal, ok := result.(*eval.BoolValue); ok && !boolVal.Value {
					minimal[i] = shrunk
					progress = true
					break
				}
			}
		}
	}

	return minimal
}

// newRNG creates a random number generator from a fixed seed. The seed is always
// used verbatim — there is deliberately no wall-clock fallback, so property
// streams are deterministic and replayable.
func newRNG(seed int64) *rand.Rand {
	return rand.New(rand.NewSource(seed))
}
