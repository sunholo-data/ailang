package testing

import (
	"fmt"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/elaborate"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/pipeline"
	"github.com/sunholo/ailang/internal/types"
)

// Executor handles evaluation of test expressions through the AILANG pipeline.
type Executor struct {
	typeEnv      *types.TypeEnv
	instEnv      *types.InstanceEnv
	dictReg      *types.DictionaryRegistry
	evalEnv      *eval.Environment
	coreEval     *eval.CoreEvaluator
	modulePath   string
	enableDebug  bool
}

// NewExecutor creates a new test executor with default environments.
func NewExecutor(modulePath string) *Executor {
	return &Executor{
		typeEnv:     types.NewTypeEnvWithBuiltins(),
		instEnv:     types.LoadBuiltinInstances(),
		dictReg:     types.NewDictionaryRegistry(),
		evalEnv:     eval.NewEnvironment(),
		coreEval:    eval.NewCoreEvaluator(),
		modulePath:  modulePath,
		enableDebug: false,
	}
}

// SetDebug enables debug output for test execution.
func (e *Executor) SetDebug(debug bool) {
	e.enableDebug = debug
}

// EvaluateExpression evaluates a Surface AST expression through the full pipeline.
// This follows the same pattern as REPL evaluation (pipeline_single.go).
func (e *Executor) EvaluateExpression(expr ast.Expr) (eval.Value, error) {
	// Phase 1: For simple evaluation, just wrap the expression directly
	// The pipeline will handle the rest
	syntheticFile := &ast.File{
		Module: &ast.ModuleDecl{
			Path: "_test/expr",
			Pos:  ast.Pos{Line: 1, Column: 1},
		},
		Statements: []ast.Node{expr},
	}

	// Phase 2: Elaborate to Core
	elaborator := elaborate.NewElaboratorWithPath(e.modulePath)
	elaborator.AddBuiltinsToGlobalEnv()
	coreProg, err := elaborator.ElaborateFile(syntheticFile)
	if err != nil {
		return nil, fmt.Errorf("elaboration error: %w", err)
	}

	if len(coreProg.Decls) == 0 {
		return nil, fmt.Errorf("elaboration produced no declarations")
	}

	// Phase 3: Type Check
	typeChecker := types.NewCoreTypeCheckerWithInstances(e.instEnv)
	coreExpr := coreProg.Decls[0]
	typedNode, _, _, constraints, err := typeChecker.InferWithConstraints(coreExpr, e.typeEnv)
	if err != nil {
		return nil, fmt.Errorf("type error: %w", err)
	}

	// Phase 3.4: Dictionary Elaboration
	resolved := typeChecker.GetResolvedConstraints()
	elaboratedProg, err := elaborate.ElaborateWithDictionaries(coreProg, resolved)
	if err != nil {
		return nil, fmt.Errorf("dictionary elaboration error: %w", err)
	}

	// Phase 3.5: Validate CoreTypeInfo before monomorphization
	if err := pipeline.ValidateCoreTypeInfo(elaboratedProg, typeChecker.CoreTI); err != nil {
		return nil, fmt.Errorf("CoreTypeInfo validation failed: %w", err)
	}

	// Phase 3.6: Monomorphization
	specializer := pipeline.NewSpecializer(&typeChecker.CoreTI)
	specializedProg, err := specializer.Specialize(elaboratedProg)
	if err != nil {
		return nil, fmt.Errorf("monomorphization error: %w", err)
	}

	// Phase 4: Var Type Resolution (modifies in place)
	resolver := pipeline.NewVarResolver(typeChecker.CoreTI)
	resolver.Resolve(specializedProg)

	// Phase 5: Operator Lowering
	lowerer := pipeline.NewOpLowerer(e.typeEnv, typeChecker.CoreTI)
	loweredProg, err := lowerer.Lower(specializedProg)
	if err != nil {
		return nil, fmt.Errorf("operator lowering error: %w", err)
	}

	// Phase 6: Evaluate
	if len(loweredProg.Decls) == 0 {
		return nil, fmt.Errorf("no declarations to evaluate")
	}

	value, err := e.coreEval.Eval(loweredProg.Decls[0])
	if err != nil {
		return nil, fmt.Errorf("runtime error: %w", err)
	}

	_ = typedNode
	_ = constraints

	return value, nil
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
