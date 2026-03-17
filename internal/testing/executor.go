package testing

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/loader"
	"github.com/sunholo/ailang/internal/pipeline"
	"github.com/sunholo/ailang/internal/runtime"
)

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
	resolver := runtime.NewBuiltinOnlyResolver(builtinRegistry)
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
