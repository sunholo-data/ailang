package testing

import (
	"fmt"

	"github.com/sunholo/ailang/internal/ast"
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
