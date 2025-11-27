package testing

import (
	"fmt"
	"os"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/pipeline"
	"github.com/sunholo/ailang/internal/runtime"
)

// Executor handles evaluation of test expressions through the AILANG pipeline.
type Executor struct {
	modulePath  string
	sourceFile  *ast.File // Full source file for context
	enableDebug bool
}

// NewExecutor creates a new test executor.
func NewExecutor(modulePath string) *Executor {
	return &Executor{
		modulePath:  modulePath,
		sourceFile:  nil,
		enableDebug: false,
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
	cfg := pipeline.Config{
		Mode: pipeline.ModeEval,
	}
	src := pipeline.Source{
		Code:     source,
		Filename: "_test.ail",
		IsREPL:   false,
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
//   binding = LetRec("factorial", λn. if n <= 1 then 1 else n * factorial(n-1))
//   tests = [(0, 1), (1, 1), (5, 120)]
//   result = EvaluateInlineTestsWithHarness(binding, tests)
//   → TupleValue([IntValue(1), IntValue(1), IntValue(120)])
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

	// Set up builtin resolver so arithmetic/comparison operators work
	builtinRegistry := runtime.NewBuiltinRegistry(evaluator)
	resolver := runtime.NewBuiltinOnlyResolver(builtinRegistry)
	evaluator.SetGlobalResolver(resolver)

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
	}

	return nil, fmt.Errorf("function '%s' not found in Core program", functionName)
}

// stripNonPureFunctions removes export functions and functions with effects from source code.
// This prevents type-checking errors for functions that require runtime context (println, etc.)
func (e *Executor) stripNonPureFunctions(source string, file *ast.File) string {
	// Build a list of non-pure function names to remove
	var nonPureFunctions []string
	for _, f := range file.Funcs {
		if !f.IsPure || f.IsExport {
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

// reconstructSource rebuilds source code from AST (simplified version).
// This is a temporary solution - ideally we'd preserve original source.
// If functionName is provided, only that function is included.
func (e *Executor) reconstructSource(file *ast.File, functionName string) string {
	var source string

	// NOTE: DO NOT include module declaration here!
	// Including "module X" triggers module loading in the pipeline,
	// which causes "module not found" errors for test files.
	// ModeEval works without module declarations.

	// Add function definitions (only the specified function if name provided)
	for _, f := range file.Funcs {
		// Skip if we're looking for a specific function and this isn't it
		if functionName != "" && f.Name != functionName {
			continue
		}

		// Only include pure functions
		if f.IsPure {
			source += fmt.Sprintf("pure func %s(", f.Name)
			for i, param := range f.Params {
				if i > 0 {
					source += ", "
				}
				source += fmt.Sprintf("%s: %v", param.Name, param.Type)
			}
			source += fmt.Sprintf(") -> %v {\n", f.ReturnType)
			source += "  " + fmt.Sprintf("%v", f.Body) + "\n}\n\n"
		}
	}

	return source
}

// EvaluateLiteral converts an AST literal expression to an eval.Value.
// This is a simplified version that only handles basic literals (int, float, bool, string).
func (e *Executor) EvaluateLiteral(expr ast.Expr) (eval.Value, error) {
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
