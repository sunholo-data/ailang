package testing

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/loader"
	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/runtime"
)

// Executor handles evaluation of test expressions through the AILANG pipeline.
type Executor struct {
	modulePath  string
	sourceFile  *ast.File // Full source file for context
	enableDebug bool
	modules     map[string]*loader.LoadedModule // Cached modules from last pipeline run
	lastMeta    map[string]*core.DeclMeta       // Cached Core.Meta from last pipeline run (lowered contracts)
}

// NewExecutor creates a new test executor.
func NewExecutor(modulePath string) *Executor {
	return &Executor{
		modulePath:  modulePath,
		sourceFile:  nil,
		enableDebug: false,
		modules:     make(map[string]*loader.LoadedModule),
		lastMeta:    nil,
	}
}

// LastDeclMeta returns the cached DeclMeta for the given function from the most
// recent pipeline run (populated by ExtractFunctionBinding). Nil if not present.
//
// M-DX26 Phase 5.1: used to access the *already-lowered* ensures predicate
// (i.e. with arithmetic ops resolved to typed dictionary calls), avoiding the
// need to re-implement OpLowering on the surface AST in the test harness.
func (e *Executor) LastDeclMeta(funcName string) *core.DeclMeta {
	if e.lastMeta == nil {
		return nil
	}
	return e.lastMeta[funcName]
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
	cfg := pipeline.Config{
		Mode: pipeline.ModeEval,
	}
	src := pipeline.Source{
		Code:     source,
		Filename: "_test.ail",
		IsREPL:   true,
	}

	result, err := pipeline.Run(cfg, src)
	if err != nil {
		return nil, err
	}

	return result.Value, nil
}

// EvaluateEnsuresHarness evaluates a single iteration of an ensures-clause property test.
// Returns a *eval.BoolValue: true if the ensures predicate held for the given inputs, false if violated.
//
// This is the M-DX26 Phase 5 entry point — it routes around the broken
// EvaluateExpression source-synthesis path by evaluating Core directly,
// the same way EvaluateInlineTestsWithHarness does for inline tests blocks.
func (e *Executor) EvaluateEnsuresHarness(binding core.RecBinding, params []EnsuresParam, predicate ast.Expr) (eval.Value, error) {
	return e.evaluateEnsuresHarnessCore(BuildEnsuresPropertyHarness(binding, params, predicate))
}

// EvaluateEnsuresHarnessFromCore is the lowered-Core variant — accepts a predicate
// that has already been through OpLowering (typically pulled from
// `result.Artifacts.Core.Meta[funcName].Contracts[i].Expr`).
//
// M-DX26 Phase 5.1: this is the path the runner uses so arithmetic operators
// in ensures predicates work without re-implementing OpLowering on the AST side.
func (e *Executor) EvaluateEnsuresHarnessFromCore(binding core.RecBinding, params []EnsuresParam, predicateCore core.CoreExpr) (eval.Value, error) {
	return e.evaluateEnsuresHarnessCore(BuildEnsuresPropertyHarnessFromCore(binding, params, predicateCore))
}

// EvaluateRequiresHarnessFromCore evaluates a requires-clause predicate against
// generated parameter values. M-DX26 Phase 5.2.
//
// `requires` does NOT need a function binding — the predicate runs before the
// function would be called and references parameters only. Returns the predicate's
// BoolValue: true = requires holds (inputs are in-contract), false = requires
// violated (test discarded).
func (e *Executor) EvaluateRequiresHarnessFromCore(params []EnsuresParam, predicateCore core.CoreExpr) (eval.Value, error) {
	return e.evaluateEnsuresHarnessCore(BuildRequiresPropertyHarnessFromCore(params, predicateCore))
}

// evaluateEnsuresHarnessCore is the shared evaluation step for both AST and Core entry points.
func (e *Executor) evaluateEnsuresHarnessCore(harnessExpr core.CoreExpr) (eval.Value, error) {

	coreProg := &core.Program{
		Decls: []core.CoreExpr{harnessExpr},
	}

	evaluator := eval.NewCoreEvaluator()
	builtinRegistry := runtime.NewBuiltinRegistry(evaluator)
	env := evaluator.Env()
	e.injectModuleBindings(evaluator, env)
	resolver := &CombinedResolver{
		Builtins: builtinRegistry,
		Env:      env,
		Modules:  e.modules,
	}
	evaluator.SetGlobalResolver(resolver)
	e.injectADTConstructors(evaluator)

	result, err := evaluator.EvalCoreProgram(coreProg)
	if err != nil {
		return nil, fmt.Errorf("ensures harness evaluation failed: %w", err)
	}
	return result, nil
}

// EvaluateNamedTestBodyExprs evaluates the body expressions of a named test block.
//
// This is the execution path for `test "name" { <exprs> }` blocks.
// It reuses the module-scope elaboration approach from the inline-test harness (v0.4.7):
//
//  1. Reads the source file, strips non-pure functions, appends the body expressions
//     (using their AST string representations) to the source text.
//  2. Runs the combined source through the full pipeline (including OpLowering) to
//     produce a Core program where the body expressions are the final decls.
//  3. Evaluates the Core program with EvalCoreProgram (returns last value).
//
// This ensures arithmetic operators (via OpLowering) and all module-scope bindings
// work correctly — exactly the same pipeline path used by inline-test evaluation.
//
// Returns the final evaluated value so the caller can check the bool pass/fail contract.
func (e *Executor) EvaluateNamedTestBodyExprs(bodyExprs []ast.Expr) (eval.Value, error) {
	if len(bodyExprs) == 0 {
		return nil, fmt.Errorf("named test block has no body expressions")
	}

	// Build source: stripped file content + body expressions appended.
	// Try to read the source file; fall back to empty if unavailable (unit tests with fake paths).
	var baseSource string
	var hasModule bool
	if e.sourceFile != nil {
		hasModule = e.sourceFile.Module != nil
		src, err := os.ReadFile(e.modulePath)
		if err == nil {
			baseSource = e.stripNonPureFunctions(string(src), e.sourceFile)
		}
		// If read fails, baseSource stays empty — we'll wrap in a synthetic module below.
	}

	// Append each body expression's source text.
	// Use PrintAILANGSource (not expr.String()) because String() uses prefix
	// notation for FuncCall which is not valid AILANG syntax.
	var sb strings.Builder
	sb.WriteString(baseSource)
	sb.WriteString("\n")
	for _, expr := range bodyExprs {
		sb.WriteString(PrintAILANGSource(expr))
		sb.WriteString("\n")
	}
	combinedSource := sb.String()

	// Determine the pipeline filename. The module loader requires either:
	//   (a) a file with an explicit module declaration on disk, or
	//   (b) a temp file with a synthetic module header.
	// We always write to a temp file to avoid mutating the original.
	var pipelineFilename string
	{
		tmpDir, err := os.MkdirTemp("", "ailang-namedtest-*")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp dir: %w", err)
		}
		defer os.RemoveAll(tmpDir)

		var baseName string
		if e.modulePath != "" {
			baseName = strings.TrimSuffix(filepath.Base(e.modulePath), ".ail")
		} else {
			baseName = "body"
		}
		tmpFile := filepath.Join(tmpDir, baseName+".ail")

		if !hasModule {
			// No module declaration — prepend a synthetic one.
			combinedSource = fmt.Sprintf("module _test/%s\n\n%s", baseName, combinedSource)
		}

		if err := os.WriteFile(tmpFile, []byte(combinedSource), 0644); err != nil {
			return nil, fmt.Errorf("failed to write temp file: %w", err)
		}
		pipelineFilename = tmpFile
	}

	cfg := pipeline.Config{
		Mode:         pipeline.ModeEval,
		RelaxModules: true,
	}
	pipelineSrc := pipeline.Source{
		Code:     combinedSource,
		Filename: pipelineFilename,
		IsREPL:   false,
	}

	pipelineResult, err := pipeline.Run(cfg, pipelineSrc)
	if err != nil {
		return nil, fmt.Errorf("pipeline error: %w", err)
	}

	if pipelineResult.Artifacts.Core == nil {
		return nil, fmt.Errorf("pipeline produced no Core program")
	}

	coreProg := pipelineResult.Artifacts.Core
	if len(coreProg.Decls) == 0 {
		return nil, fmt.Errorf("Core program has no declarations")
	}

	// Cache modules for future use.
	e.modules = pipelineResult.Modules

	// Evaluate all decls; EvalCoreProgram returns the last value.
	// This ensures function bindings are in scope when the body expression is evaluated.
	evaluator := eval.NewCoreEvaluator()
	builtinRegistry := runtime.NewBuiltinRegistry(evaluator)
	env := evaluator.Env()
	e.injectModuleBindings(evaluator, env)
	resolver := &CombinedResolver{
		Builtins: builtinRegistry,
		Env:      env,
		Modules:  e.modules,
	}
	evaluator.SetGlobalResolver(resolver)
	e.injectADTConstructors(evaluator)

	val, err := evaluator.EvalCoreProgram(coreProg)
	if err != nil {
		return nil, fmt.Errorf("evaluation error: %w", err)
	}
	return val, nil
}

// EvaluateInlineTestsWithHarness evaluates inline tests using the test harness builder.
// This is the PREFERRED method for inline tests (fixes scoping issues in EvaluateExpression).
func (e *Executor) EvaluateInlineTestsWithHarness(binding core.RecBinding, tests []TestCase) (*eval.TupleValue, error) {
	// Build test harness using the harness builder
	harnessExpr := BuildInlineTestHarness(binding, tests)

	// Wrap harness in a Core program for evaluation
	coreProg := &core.Program{
		Decls: []core.CoreExpr{harnessExpr},
	}

	// Evaluate the harness
	evaluator := eval.NewCoreEvaluator()

	// Set up builtin registry and combined resolver
	builtinRegistry := runtime.NewBuiltinRegistry(evaluator)

	// Get the evaluator's environment for the combined resolver
	env := evaluator.Env()

	// Inject elaborated module functions into the environment
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

	// Strip out non-pure functions
	strippedSource := e.stripNonPureFunctions(string(sourceCode), sourceFile)

	pipelineFilename := e.modulePath
	if sourceFile.Module == nil {
		// Write temp file with synthetic module so the module pipeline can load it
		tmpDir, tmpErr := os.MkdirTemp("", "ailang-test-*")
		if tmpErr != nil {
			return nil, fmt.Errorf("failed to create temp dir for module-less test: %w", tmpErr)
		}
		defer os.RemoveAll(tmpDir)
		baseName := strings.TrimSuffix(filepath.Base(e.modulePath), ".ail")
		tmpFile := filepath.Join(tmpDir, filepath.Base(e.modulePath))
		syntheticSource := fmt.Sprintf("module _test/%s\n\n%s", baseName, strippedSource)
		if writeErr := os.WriteFile(tmpFile, []byte(syntheticSource), 0644); writeErr != nil {
			return nil, fmt.Errorf("failed to write temp file for module-less test: %w", writeErr)
		}
		pipelineFilename = tmpFile
	}
	cfg := pipeline.Config{
		Mode:         pipeline.ModeEval,
		RelaxModules: true,
	}
	src := pipeline.Source{
		Code:     strippedSource,
		Filename: pipelineFilename,
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

	// M-DX26 Phase 5.1: Cache the elaborated + lowered Meta so runEnsuresProperty
	// can pull the already-lowered ensures predicate (arithmetic ops resolved).
	e.lastMeta = result.Artifacts.Core.Meta

	// Search for the function binding in Core.Decls
	for _, decl := range result.Artifacts.Core.Decls {
		if letRec, ok := decl.(*core.LetRec); ok {
			for _, binding := range letRec.Bindings {
				if binding.Name == functionName {
					return &binding, nil
				}
			}
		}

		if let, ok := decl.(*core.Let); ok {
			if let.Name == functionName {
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
func (e *Executor) stripNonPureFunctions(source string, file *ast.File) string {
	var nonPureFunctions []string
	for _, f := range file.Funcs {
		if !f.IsPure {
			nonPureFunctions = append(nonPureFunctions, f.Name)
		}
	}

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

// EvaluateLiteral converts an AST literal expression to an eval.Value.
func (e *Executor) EvaluateLiteral(expr ast.Expr) (eval.Value, error) {
	// Handle UnaryOp (e.g., -3)
	if unop, ok := expr.(*ast.UnaryOp); ok {
		if unop.Op == "-" {
			operand, err := e.EvaluateLiteral(unop.Expr)
			if err != nil {
				return nil, err
			}
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
func (e *Executor) CompareValues(actual, expected eval.Value) bool {
	return equalValues(actual, expected)
}

// EvaluateInlineTestsWithCluster evaluates inline tests for a function with cross-function dependencies.
func (e *Executor) EvaluateInlineTestsWithCluster(
	functionName string,
	tests []TestCase,
	coreProg *core.Program,
) (*eval.TupleValue, error) {
	g := BuildCallGraph(coreProg)
	sccs := ComputeSCCs(g)
	closure := GetDependencyClosure(g, sccs, functionName)
	if closure == nil {
		return nil, fmt.Errorf("function '%s' not found in call graph", functionName)
	}

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

	harnessExpr := BuildClusterTestHarness(cluster, tests)
	harnessProgram := &core.Program{
		Decls: []core.CoreExpr{harnessExpr},
	}

	evaluator := eval.NewCoreEvaluator()
	builtinRegistry := runtime.NewBuiltinRegistry(evaluator)
	env := evaluator.Env()
	e.injectModuleBindings(evaluator, env)
	resolver := &CombinedResolver{
		Builtins: builtinRegistry,
		Env:      env,
		Modules:  e.modules,
	}
	evaluator.SetGlobalResolver(resolver)
	e.injectADTConstructors(evaluator)

	result, err := evaluator.EvalCoreProgram(harnessProgram)
	if err != nil {
		return nil, fmt.Errorf("cluster harness evaluation failed: %w", err)
	}

	tupleResult, ok := result.(*eval.TupleValue)
	if !ok {
		return nil, fmt.Errorf("harness should return tuple, got %T", result)
	}

	return tupleResult, nil
}

// ExtractPureClusterForFunction extracts the pure dependency cluster for a function from source code.
func (e *Executor) ExtractPureClusterForFunction(
	functionName string,
	sourceFile *ast.File,
) (*PureCluster, *core.Program, error) {
	sourceCode, err := os.ReadFile(e.modulePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read source file: %w", err)
	}

	strippedSource := e.stripNonPureFunctions(string(sourceCode), sourceFile)

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

	e.modules = result.Modules

	coreProg := result.Artifacts.Core
	g := BuildCallGraph(coreProg)
	sccs := ComputeSCCs(g)
	closure := GetDependencyClosure(g, sccs, functionName)
	if closure == nil {
		return nil, nil, fmt.Errorf("function '%s' not found in call graph", functionName)
	}

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
func (e *Executor) HasCrossFunctionDependencies(
	functionName string,
	coreProg *core.Program,
) bool {
	g := BuildCallGraph(coreProg)
	sccs := ComputeSCCs(g)
	closure := GetDependencyClosure(g, sccs, functionName)
	return len(closure) > 1
}
