package elaborate

import (
	"fmt"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/core"
)

// Expression normalization entry point and dispatcher
// Individual normalizers are in separate files:
//   - expr_simple.go: literals, lambdas, operators
//   - expr_control.go: if, let, letrec, block
//   - expr_calls.go: function calls
//   - expr_data.go: records, lists, arrays, tuples

// elaborateExpr transforms surface expression to Core ANF
func (e *Elaborator) elaborateExpr(expr ast.Expr) (core.CoreExpr, error) {
	// First pass: desugar surface constructs
	desugared := e.desugar(expr)

	// Second pass: normalize to ANF
	return e.normalize(desugared)
}

// desugar handles surface syntax sugar
func (e *Elaborator) desugar(expr ast.Expr) ast.Expr {
	// For now, pass through - will add ? operator desugaring etc
	return expr
}

// normalize converts expression to A-Normal Form
func (e *Elaborator) normalize(expr ast.Expr) (core.CoreExpr, error) {
	switch ex := expr.(type) {
	case *ast.Literal:
		return e.normalizeLiteral(ex)

	case *ast.Identifier:
		// Check if this is a nullary constructor (e.g., None, True, False)
		if ctorInfo, isConstructor := e.constructors[ex.Name]; isConstructor && ctorInfo.Arity == 0 {
			// Nullary constructor: transform None → $adt.make_Option_None
			// Note: No App node needed - factory is a value, not a function call
			factoryName := fmt.Sprintf("make_%s_%s", ctorInfo.TypeName, ctorInfo.CtorName)
			return &core.VarGlobal{
				CoreNode: e.makeNode(ex.Position()),
				Ref: core.GlobalRef{
					Module: "$adt",
					Name:   factoryName,
				},
			}, nil
		}
		// Check if this is an imported symbol
		if ref, ok := e.globalEnv[ex.Name]; ok {
			return &core.VarGlobal{
				CoreNode: e.makeNode(ex.Position()),
				Ref:      ref,
			}, nil
		}
		// Otherwise it's a local variable
		return &core.Var{
			CoreNode: e.makeNode(ex.Position()),
			Name:     ex.Name,
		}, nil

	case *ast.Lambda:
		return e.normalizeLambda(ex)

	case *ast.FuncLit:
		return e.normalizeFuncLit(ex)

	case *ast.BinaryOp:
		return e.normalizeBinaryOp(ex)

	case *ast.UnaryOp:
		return e.normalizeUnaryOp(ex)

	case *ast.If:
		return e.normalizeIf(ex)

	case *ast.Let:
		return e.normalizeLet(ex)

	case *ast.LetRec:
		return e.normalizeLetRec(ex)

	case *ast.Block:
		return e.normalizeBlock(ex)

	case *ast.FuncCall:
		return e.normalizeFuncCall(ex)

	case *ast.Record:
		return e.normalizeRecord(ex)

	case *ast.RecordAccess:
		return e.normalizeRecordAccess(ex)

	case *ast.RecordUpdate:
		return e.normalizeRecordUpdate(ex)

	case *ast.List:
		return e.normalizeList(ex)

	case *ast.Array:
		return e.normalizeArray(ex)

	case *ast.Tuple:
		return e.normalizeTuple(ex)

	case *ast.Match:
		return e.normalizeMatch(ex)

	case *ast.ForallExpr:
		return e.normalizeForall(ex)

	default:
		if expr == nil {
			return nil, fmt.Errorf("normalization received nil expression")
		}
		return nil, fmt.Errorf("normalization not implemented for %T", expr)
	}
}
