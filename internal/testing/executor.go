package testing

import (
	"fmt"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/pipeline"
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
	// Build source code string from AST
	// We need to reconstruct the source to run through the pipeline
	source := e.reconstructSource(sourceFile)

	// Run through pipeline to get Core
	cfg := pipeline.Config{
		Mode: pipeline.ModeEval,
	}
	src := pipeline.Source{
		Code:     source,
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

// reconstructSource rebuilds source code from AST (simplified version).
// This is a temporary solution - ideally we'd preserve original source.
func (e *Executor) reconstructSource(file *ast.File) string {
	var source string

	// Add module declaration if present
	if file.Module != nil {
		source += fmt.Sprintf("module %s\n\n", file.Module.Path)
	}

	// Add function definitions
	for _, f := range file.Funcs {
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
