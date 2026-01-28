package testing

import (
	"fmt"
	"os"
	"strings"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/loader"
	"github.com/sunholo/ailang/internal/pipeline"
	"github.com/sunholo/ailang/internal/runtime"
)

// CombinedResolver resolves both builtin functions and user-defined functions from the environment.
// Used for inline test harness evaluation to support functions that depend on imports.
// It handles:
// - Builtin references (module="$builtin" or name starts with "_")
// - Module-qualified references (module="std/list" name="filter")
// - Local references (module="" or module matches current file)
type CombinedResolver struct {
	Builtins *runtime.BuiltinRegistry
	Env      *eval.Environment               // Environment containing user-defined and imported functions
	Modules  map[string]*loader.LoadedModule // Loaded modules for module-qualified lookup
}

// ResolveValue implements eval.GlobalResolver for combined resolution.
func (r *CombinedResolver) ResolveValue(ref core.GlobalRef) (eval.Value, error) {
	// Case 1: Builtin references (module="$builtin" or name starts with "_")
	if ref.Module == "$builtin" || strings.HasPrefix(ref.Name, "_") {
		if val, ok := r.Builtins.Get(ref.Name); ok {
			return val, nil
		}
		// Not found in builtins - might be in environment
		if val, ok := r.Env.Get(ref.Name); ok {
			return val, nil
		}
		return nil, fmt.Errorf("builtin %s not found", ref.Name)
	}

	// Case 2: Module-qualified reference (e.g., std/list.filter)
	if ref.Module != "" {
		// Look for the function in the specified module
		if mod, ok := r.Modules[ref.Module]; ok && mod != nil {
			// Try to find the function in the module's Core program
			for _, decl := range mod.Core.Decls {
				// Check Let bindings
				if let, ok := decl.(*core.Let); ok {
					if let.Name == ref.Name {
						// Try to get this from the environment (should have been evaluated)
						if val, ok := r.Env.Get(ref.Name); ok {
							return val, nil
						}
						// If not in environment, return error with details
						return nil, fmt.Errorf("function %s.%s not yet evaluated in environment", ref.Module, ref.Name)
					}
				}
				// Check LetRec bindings
				if letRec, ok := decl.(*core.LetRec); ok {
					for _, binding := range letRec.Bindings {
						if binding.Name == ref.Name {
							if val, ok := r.Env.Get(ref.Name); ok {
								return val, nil
							}
							return nil, fmt.Errorf("function %s.%s not yet evaluated in environment", ref.Module, ref.Name)
						}
					}
				}
			}
		}
		return nil, fmt.Errorf("module %s not found or function %s not in module", ref.Module, ref.Name)
	}

	// Case 3: Unqualified reference - look in environment
	// This includes both the test function being tested and any imported functions
	// that were elaborated and bound during pipeline execution.
	if val, ok := r.Env.Get(ref.Name); ok {
		return val, nil
	}

	// Case 4: Not found - return error (will be caught during harness evaluation)
	return nil, fmt.Errorf("undefined reference: %s (module: %s)", ref.Name, ref.Module)
}

// Executor handles evaluation of test expressions through the AILANG pipeline.
type Executor struct {
	modulePath  string
	sourceFile  *ast.File // Full source file for context
	enableDebug bool
	modules     map[string]*loader.LoadedModule // Cached modules from last pipeline run
}

// NewExecutor creates a new test executor.
func NewExecutor(modulePath string) *Executor {
	return &Executor{
		modulePath:  modulePath,
		sourceFile:  nil,
		enableDebug: false,
		modules:     make(map[string]*loader.LoadedModule),
	}
}

// SetSourceFile sets the source file to provide context for test evaluation.
func (e *Executor) SetSourceFile(file *ast.File) {
	e.sourceFile = file
}

// SetDebug enables debug output for test execution.
func (e *Executor) SetDebug(debug bool) {
	e.enableDebug = debug
}

// EvaluateExpression evaluates a Surface AST expression through the pipeline.
// Uses ModeEval to properly handle function definitions and expression evaluation.
func (e *Executor) EvaluateExpression(expr ast.Expr) (eval.Value, error) {
	// Build synthetic source with pure functions + test expression
	// NOTE: No module declaration - this triggers ModeEval for direct evaluation
	var sourceParts []string

	// Include pure function definitions (not main() with effects)
	if e.sourceFile != nil {
		for _, f := range e.sourceFile.Funcs {
			if f.IsPure {
				// Reconstruct function source from AST
				// This is a simplified reconstruction - full version would preserve exact source
				funcSrc := fmt.Sprintf("pure func %s(", f.Name)
				for i, param := range f.Params {
					if i > 0 {
						funcSrc += ", "
					}
					funcSrc += fmt.Sprintf("%s: %v", param.Name, param.Type)
				}
				funcSrc += fmt.Sprintf(") -> %v {\n", f.ReturnType)
				funcSrc += "  " + fmt.Sprintf("%v", f.Body) + "\n}\n\n"
				sourceParts = append(sourceParts, funcSrc)
			}
		}
	}

	// Add test expression
	sourceParts = append(sourceParts, fmt.Sprintf("%v", expr))

	source := ""
	for _, part := range sourceParts {
		source += part
	}

	// Use pipeline with ModeEval (non-module evaluation)
	// Set IsREPL: true to prevent the pipeline from treating this as a module
	// and attempting to resolve imports (fixes "module not found: _test" error)
	cfg := pipeline.Config{
		Mode: pipeline.ModeEval,
	}
	src := pipeline.Source{
		Code:     source,
		Filename: "_test.ail",
		IsREPL:   true, // Treat as REPL-like evaluation, not a module
	}

	result, err := pipeline.Run(cfg, src)
	if err != nil {
		return nil, err
	}

	return result.Value, nil
}

// EvaluateInlineTestsWithHarness evaluates inline tests using the test harness builder.
// This is the PREFERRED method for inline tests (fixes scoping issues in EvaluateExpression).
//
// Given:
//   - binding: Core LetRec binding for the function being tested
//   - tests: List of inline test cases with (input, expected) tuples
//
// Returns:
//   - Tuple of actual result values
//   - Error if harness evaluation fails
//
// Example:
//
//	binding = LetRec("factorial", λn. if n <= 1 then 1 else n * factorial(n-1))
//	tests = [(0, 1), (1, 1), (5, 120)]
//	result = EvaluateInlineTestsWithHarness(binding, tests)
//	→ TupleValue([IntValue(1), IntValue(1), IntValue(120)])
func (e *Executor) EvaluateInlineTestsWithHarness(binding core.RecBinding, tests []TestCase) (*eval.TupleValue, error) {
	// Build test harness using the harness builder
	harnessExpr := BuildInlineTestHarness(binding, tests)

	// Wrap harness in a Core program for evaluation
	coreProg := &core.Program{
		Decls: []core.CoreExpr{harnessExpr},
	}

	// Evaluate the harness
	// Note: We use the eval package directly since we already have Core
	evaluator := eval.NewCoreEvaluator()

	// Set up builtin registry and combined resolver
	// The combined resolver handles both builtins and user-defined/imported functions
	builtinRegistry := runtime.NewBuiltinRegistry(evaluator)

	// Get the evaluator's environment for the combined resolver
	env := evaluator.Env()

	// Inject elaborated module functions into the environment
	// This binds functions that were imported and elaborated during pipeline execution
	e.injectModuleBindings(evaluator, env)

	// Create combined resolver that can handle both builtins and module functions
	resolver := &CombinedResolver{
		Builtins: builtinRegistry,
		Env:      env,
		Modules:  e.modules,
	}
	evaluator.SetGlobalResolver(resolver)

	// Inject ADT constructor bindings from source file so test inputs like (North, 0) work
	e.injectADTConstructors(evaluator)

	result, err := evaluator.EvalCoreProgram(coreProg)
	if err != nil {
		return nil, fmt.Errorf("harness evaluation failed: %w", err)
	}

	// Result should be a tuple of actual values
	tupleResult, ok := result.(*eval.TupleValue)
	if !ok {
		return nil, fmt.Errorf("harness should return tuple, got %T", result)
	}

	return tupleResult, nil
}

// ExtractFunctionBinding extracts a Core LetRec binding for a function from source code.
// This runs the source through elaboration to get the Core program, then extracts the binding.
//
// Given:
//   - functionName: Name of the function to extract
//   - sourceFile: The full source AST (for function definitions context)
//
// Returns:
//   - Core LetRec binding for the function
//   - Error if function not found or elaboration fails
func (e *Executor) ExtractFunctionBinding(functionName string, sourceFile *ast.File) (*core.RecBinding, error) {
	// Find the function declaration in the AST
	var funcDecl *ast.FuncDecl
	for _, f := range sourceFile.Funcs {
		if f.Name == functionName {
			funcDecl = f
			break
		}
	}

	if funcDecl == nil {
		return nil, fmt.Errorf("function '%s' not found in source file", functionName)
	}

	// Read original source file to get actual source code
	sourceCode, err := os.ReadFile(e.modulePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read source file: %w", err)
	}

	// Strip out non-pure functions (export func, func with effects)
	// to avoid type-checking errors in test mode
	strippedSource := e.stripNonPureFunctions(string(sourceCode), sourceFile)

	// Run through pipeline to get Core for pure functions only
	cfg := pipeline.Config{
		Mode: pipeline.ModeEval,
	}
	src := pipeline.Source{
		Code:     strippedSource,
		Filename: e.modulePath,
		IsREPL:   false,
	}

	result, err := pipeline.Run(cfg, src)
	if err != nil {
		return nil, fmt.Errorf("failed to elaborate source: %w", err)
	}

	// Cache the modules from the pipeline result for use in test harness evaluation
	e.modules = result.Modules

	// Extract the LetRec binding from the Core program
	if result.Artifacts.Core == nil {
		return nil, fmt.Errorf("pipeline did not produce Core program")
	}

	// Search for the function binding in Core.Decls
	// Handle both LetRec (recursive functions) and Let (non-recursive functions)
	for _, decl := range result.Artifacts.Core.Decls {
		// Try LetRec first (recursive functions like factorial)
		if letRec, ok := decl.(*core.LetRec); ok {
			for _, binding := range letRec.Bindings {
				if binding.Name == functionName {
					return &binding, nil
				}
			}
		}

		// Also try Let (non-recursive functions like identity, add, max)
		if let, ok := decl.(*core.Let); ok {
			if let.Name == functionName {
				// Convert Let to RecBinding format
				binding := core.RecBinding{
					Name:  let.Name,
					Value: let.Value,
				}
				return &binding, nil
			}
		}
	}

	return nil, fmt.Errorf("function '%s' not found in Core program", functionName)
}

// stripNonPureFunctions removes functions with effects from source code.
// This prevents type-checking errors for functions that require runtime context (println, etc.)
// Note: Exported pure functions are KEPT (they're testable via inline tests).
func (e *Executor) stripNonPureFunctions(source string, file *ast.File) string {
	// Build a list of non-pure function names to remove
	var nonPureFunctions []string
	for _, f := range file.Funcs {
		if !f.IsPure {
			nonPureFunctions = append(nonPureFunctions, f.Name)
		}
	}

	// Simple approach: For now, just comment out the main() function
	// since that's the only non-pure function in factorial.ail
	// A proper implementation would use AST positions to precisely remove declarations
	lines := []string{}
	for _, line := range splitLines(source) {
		skip := false
		for _, funcName := range nonPureFunctions {
			if containsPattern(line, "export func "+funcName) || containsPattern(line, "func "+funcName) {
				skip = true
				break
			}
		}
		if !skip {
			lines = append(lines, line)
		}
	}

	return joinLines(lines)
}

// Helper: split source into lines
func splitLines(s string) []string {
	result := []string{}
	current := ""
	for _, ch := range s {
		current += string(ch)
		if ch == '\n' {
			result = append(result, current)
			current = ""
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

// Helper: join lines back
func joinLines(lines []string) string {
	result := ""
	for _, line := range lines {
		result += line
	}
	return result
}

// Helper: check if line contains pattern
func containsPattern(line, pattern string) bool {
	return len(line) >= len(pattern) && findSubstring(line, pattern)
}

// Helper: find substring
func findSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// EvaluateLiteral converts an AST literal expression to an eval.Value.
// This is a simplified version that only handles basic literals (int, float, bool, string).
// Also handles UnaryOp for negative numbers.
func (e *Executor) EvaluateLiteral(expr ast.Expr) (eval.Value, error) {
	// Handle UnaryOp (e.g., -3)
	if unop, ok := expr.(*ast.UnaryOp); ok {
		if unop.Op == "-" {
			// Get the operand value
			operand, err := e.EvaluateLiteral(unop.Expr)
			if err != nil {
				return nil, err
			}
			// Negate it
			if intVal, ok := operand.(*eval.IntValue); ok {
				return &eval.IntValue{Value: -intVal.Value}, nil
			}
			if floatVal, ok := operand.(*eval.FloatValue); ok {
				return &eval.FloatValue{Value: -floatVal.Value}, nil
			}
			return nil, fmt.Errorf("cannot negate non-numeric value: %T", operand)
		}
		return nil, fmt.Errorf("unsupported unary operator: %s", unop.Op)
	}

	lit, ok := expr.(*ast.Literal)
	if !ok {
		return nil, fmt.Errorf("expected literal expression, got %T", expr)
	}

	switch lit.Kind {
	case ast.IntLit:
		if v, ok := lit.Value.(int64); ok {
			return &eval.IntValue{Value: int(v)}, nil
		}
		return nil, fmt.Errorf("invalid int literal value: %T", lit.Value)
	case ast.FloatLit:
		if v, ok := lit.Value.(float64); ok {
			return &eval.FloatValue{Value: v}, nil
		}
		return nil, fmt.Errorf("invalid float literal value: %T", lit.Value)
	case ast.BoolLit:
		if v, ok := lit.Value.(bool); ok {
			return &eval.BoolValue{Value: v}, nil
		}
		return nil, fmt.Errorf("invalid bool literal value: %T", lit.Value)
	case ast.StringLit:
		if v, ok := lit.Value.(string); ok {
			return &eval.StringValue{Value: v}, nil
		}
		return nil, fmt.Errorf("invalid string literal value: %T", lit.Value)
	case ast.UnitLit:
		return &eval.UnitValue{}, nil
	default:
		return nil, fmt.Errorf("unsupported literal kind: %v", lit.Kind)
	}
}

// CompareValues checks if two values are equal.
// Used for test assertions (actual == expected).
func (e *Executor) CompareValues(actual, expected eval.Value) bool {
	return equalValues(actual, expected)
}

// equalValues performs deep equality check on eval values.
func equalValues(a, b eval.Value) bool {
	switch av := a.(type) {
	case *eval.IntValue:
		if bv, ok := b.(*eval.IntValue); ok {
			return av.Value == bv.Value
		}
	case *eval.FloatValue:
		if bv, ok := b.(*eval.FloatValue); ok {
			// Float comparison with tolerance for testing
			diff := av.Value - bv.Value
			if diff < 0 {
				diff = -diff
			}
			return diff < 1e-9
		}
	case *eval.BoolValue:
		if bv, ok := b.(*eval.BoolValue); ok {
			return av.Value == bv.Value
		}
	case *eval.StringValue:
		if bv, ok := b.(*eval.StringValue); ok {
			return av.Value == bv.Value
		}
	case *eval.ListValue:
		if bv, ok := b.(*eval.ListValue); ok {
			if len(av.Elements) != len(bv.Elements) {
				return false
			}
			for i := range av.Elements {
				if !equalValues(av.Elements[i], bv.Elements[i]) {
					return false
				}
			}
			return true
		}
	case *eval.TupleValue:
		if bv, ok := b.(*eval.TupleValue); ok {
			if len(av.Elements) != len(bv.Elements) {
				return false
			}
			for i := range av.Elements {
				if !equalValues(av.Elements[i], bv.Elements[i]) {
					return false
				}
			}
			return true
		}
	case *eval.RecordValue:
		if bv, ok := b.(*eval.RecordValue); ok {
			if len(av.Fields) != len(bv.Fields) {
				return false
			}
			for k, av := range av.Fields {
				bv, exists := bv.Fields[k]
				if !exists {
					return false
				}
				if !equalValues(av, bv) {
					return false
				}
			}
			return true
		}
	case *eval.TaggedValue:
		if bv, ok := b.(*eval.TaggedValue); ok {
			// Compare constructor names and fields
			if av.CtorName != bv.CtorName {
				return false
			}
			if len(av.Fields) != len(bv.Fields) {
				return false
			}
			for i := range av.Fields {
				if !equalValues(av.Fields[i], bv.Fields[i]) {
					return false
				}
			}
			return true
		}
	case *eval.UnitValue:
		_, ok := b.(*eval.UnitValue)
		return ok
	}
	return false
}

// EvaluateInlineTestsWithCluster evaluates inline tests for a function with cross-function dependencies.
// This method automatically detects dependencies and includes them in the test harness.
//
// Given:
//   - functionName: Name of the function being tested
//   - tests: List of inline test cases
//   - coreProg: The Core program containing all function definitions (should be pure-only)
//
// Returns:
//   - Tuple of actual result values
//   - Error if evaluation fails
//
// Note: This assumes coreProg contains only pure functions (e.g., after stripNonPureFunctions).
// Purity checking is skipped since the input is pre-filtered.
func (e *Executor) EvaluateInlineTestsWithCluster(
	functionName string,
	tests []TestCase,
	coreProg *core.Program,
) (*eval.TupleValue, error) {
	// Build call graph from Core program
	g := BuildCallGraph(coreProg)

	// Compute SCCs
	sccs := ComputeSCCs(g)

	// Get the dependency closure (all functions reachable from functionName)
	closure := GetDependencyClosure(g, sccs, functionName)
	if closure == nil {
		return nil, fmt.Errorf("function '%s' not found in call graph", functionName)
	}

	// Build pure cluster directly from call graph bindings
	// Since coreProg was pre-filtered to pure functions, no purity check needed
	bindings := make([]core.RecBinding, 0, len(closure))
	names := make(map[string]bool)
	for _, name := range closure {
		if binding, ok := g.Bindings[name]; ok {
			bindings = append(bindings, *binding)
			names[name] = true
		}
	}

	cluster := &PureCluster{
		FuncName: functionName,
		Bindings: bindings,
		Names:    names,
	}

	// Build cluster test harness
	harnessExpr := BuildClusterTestHarness(cluster, tests)

	// Wrap harness in a Core program for evaluation
	harnessProgram := &core.Program{
		Decls: []core.CoreExpr{harnessExpr},
	}

	// Evaluate the harness
	evaluator := eval.NewCoreEvaluator()

	// Set up builtin resolver so arithmetic/comparison operators work
	builtinRegistry := runtime.NewBuiltinRegistry(evaluator)
	resolver := runtime.NewBuiltinOnlyResolver(builtinRegistry)
	evaluator.SetGlobalResolver(resolver)

	// Inject ADT constructor bindings from source file so test inputs like (North, 0) work
	e.injectADTConstructors(evaluator)

	result, err := evaluator.EvalCoreProgram(harnessProgram)
	if err != nil {
		return nil, fmt.Errorf("cluster harness evaluation failed: %w", err)
	}

	// Result should be a tuple of actual values
	tupleResult, ok := result.(*eval.TupleValue)
	if !ok {
		return nil, fmt.Errorf("harness should return tuple, got %T", result)
	}

	return tupleResult, nil
}

// ExtractPureClusterForFunction extracts the pure dependency cluster for a function from source code.
// This runs the source through the pipeline, builds call graph, computes SCCs, and returns the cluster.
//
// Returns:
//   - PureCluster containing all pure functions needed to test the target function
//   - Core program (pre-filtered to pure functions)
//   - Error if function not found or pipeline fails
func (e *Executor) ExtractPureClusterForFunction(
	functionName string,
	sourceFile *ast.File,
) (*PureCluster, *core.Program, error) {
	// Read original source file
	sourceCode, err := os.ReadFile(e.modulePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read source file: %w", err)
	}

	// Strip out non-pure functions (ensures all remaining functions are pure)
	strippedSource := e.stripNonPureFunctions(string(sourceCode), sourceFile)

	// Run through pipeline to get Core
	cfg := pipeline.Config{
		Mode: pipeline.ModeEval,
	}
	src := pipeline.Source{
		Code:     strippedSource,
		Filename: e.modulePath,
		IsREPL:   false,
	}

	result, err := pipeline.Run(cfg, src)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to elaborate source: %w", err)
	}

	if result.Artifacts.Core == nil {
		return nil, nil, fmt.Errorf("pipeline did not produce Core program")
	}

	coreProg := result.Artifacts.Core

	// Build call graph
	g := BuildCallGraph(coreProg)

	// Compute SCCs
	sccs := ComputeSCCs(g)

	// Get dependency closure
	closure := GetDependencyClosure(g, sccs, functionName)
	if closure == nil {
		return nil, nil, fmt.Errorf("function '%s' not found in call graph", functionName)
	}

	// Build cluster directly (purity already ensured by stripping non-pure functions)
	bindings := make([]core.RecBinding, 0, len(closure))
	names := make(map[string]bool)
	for _, name := range closure {
		if binding, ok := g.Bindings[name]; ok {
			bindings = append(bindings, *binding)
			names[name] = true
		}
	}

	cluster := &PureCluster{
		FuncName: functionName,
		Bindings: bindings,
		Names:    names,
	}

	return cluster, coreProg, nil
}

// HasCrossFunctionDependencies checks if a function has dependencies on other user-defined functions.
// This is useful to determine whether to use the simple harness or cluster harness.
func (e *Executor) HasCrossFunctionDependencies(
	functionName string,
	coreProg *core.Program,
) bool {
	g := BuildCallGraph(coreProg)
	sccs := ComputeSCCs(g)

	// Get the dependency closure
	closure := GetDependencyClosure(g, sccs, functionName)

	// If closure has more than one function, there are dependencies
	return len(closure) > 1
}

// injectModuleBindings evaluates all module Core programs and injects their bindings
// into the evaluator's environment. This allows the test harness to reference functions
// that were imported and elaborated (like functions from std/fs, std/net, etc.).
//
// CRITICAL BUG FIX (M-DX25):
// The issue was that FunctionValues were capturing `env` at injection time, before all
// module bindings were added to `env`. When a function's body references another imported
// function, that reference might not be in the captured environment snapshot.
//
// Example of the bug:
//  1. Module std/fs has function `read` that internally calls `_io_read`
//  2. We inject `read` with Env: env (capturing current env, which doesn't have `_io_read` yet)
//  3. Later, we inject `_io_read` into env
//  4. When evaluating test harness that calls `read`:
//     - read's body references `_io_read`
//     - But `_io_read` is not in read's captured env (it was added after the capture!)
//     - Resolution fails with "cannot apply non-function value: <nil>"
//
// Solution: Use a two-pass approach:
//
//	Pass 1: Collect all lambda bindings to inject, but don't create FunctionValues yet
//	Pass 2: After env is populated with all names, create FunctionValues that capture
//	        the now-complete environment
func (e *Executor) injectModuleBindings(evaluator *eval.CoreEvaluator, env *eval.Environment) {
	if len(e.modules) == 0 {
		return
	}

	// PASS 1: Collect pending bindings (lambdas) and inject non-lambda values
	type PendingLambdaBinding struct {
		name   string
		lambda *core.Lambda
	}
	var pendingLambdas []PendingLambdaBinding

	for _, mod := range e.modules {
		if mod.Core == nil {
			continue
		}

		// Process each declaration in the module
		for _, decl := range mod.Core.Decls {
			switch d := decl.(type) {
			case *core.Let:
				// For pure functions, the value is a Lambda - queue it for Pass 2
				if lambda, ok := d.Value.(*core.Lambda); ok {
					pendingLambdas = append(pendingLambdas, PendingLambdaBinding{
						name:   d.Name,
						lambda: lambda,
					})
				} else if _, ok := d.Value.(*core.VarGlobal); ok {
					// This is a re-export of another module's function
					// We need to evaluate it to get the actual function value
					val, err := evaluator.Eval(d.Value)
					if err == nil && val != nil {
						env.Set(d.Name, val)
					}
				}

			case *core.LetRec:
				// For recursive bindings, queue the lambdas for Pass 2
				for _, binding := range d.Bindings {
					if lambda, ok := binding.Value.(*core.Lambda); ok {
						pendingLambdas = append(pendingLambdas, PendingLambdaBinding{
							name:   binding.Name,
							lambda: lambda,
						})
					}
				}
			}
		}
	}

	// PASS 2: Now create FunctionValues with the fully-populated environment
	// This ensures all function dependencies can be resolved from env.
	for _, pending := range pendingLambdas {
		funcVal := &eval.FunctionValue{
			Params: extractLambdaParams(pending.lambda),
			Body:   pending.lambda.Body,
			Env:    env, // Capture the fully-populated environment
			Typed:  true,
		}
		env.Set(pending.name, funcVal)
	}
}

// extractLambdaParams extracts parameter names from a Core Lambda
func extractLambdaParams(lambda *core.Lambda) []string {
	return lambda.Params
}

func (e *Executor) injectADTConstructors(evaluator *eval.CoreEvaluator) {
	if e.sourceFile == nil {
		return
	}

	env := evaluator.Env()

	// Type declarations are in Decls ([]Node)
	for _, decl := range e.sourceFile.Decls {
		typeDecl, ok := decl.(*ast.TypeDecl)
		if !ok {
			continue
		}

		// Only process ADTs (algebraic types)
		if adt, ok := typeDecl.Definition.(*ast.AlgebraicType); ok {
			typeName := typeDecl.Name
			for _, ctor := range adt.Constructors {
				ctorName := ctor.Name
				arity := len(ctor.Fields)

				if arity == 0 {
					// Nullary constructor - bind directly to TaggedValue
					env.Set(ctorName, &eval.TaggedValue{
						TypeName: typeName,
						CtorName: ctorName,
						Fields:   []eval.Value{},
					})
				} else {
					// Constructor with data - bind to ConstructorClosure
					env.Set(ctorName, &eval.ConstructorClosure{
						TypeName: typeName,
						CtorName: ctorName,
						Arity:    arity,
					})
				}
			}
		}
	}
}
