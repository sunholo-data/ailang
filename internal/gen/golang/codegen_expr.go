// Package golang provides Go code generation from AILANG Core AST.
package golang

import (
	"fmt"

	"github.com/sunholo/ailang/internal/core"
)

// generateExpr generates Go code for a Core expression.
// This is the main dispatcher that routes to specialized handlers in:
// - codegen_expr_simple.go: Literals, variables, lambdas, effect mappings
// - codegen_expr_app.go: Function application and type inference
// - codegen_expr_let.go: Let/LetRec bindings
// - codegen_expr_control.go: If expressions and native operators
// - codegen_ops.go: Binary/unary operators, intrinsics, data structures
// - codegen_match.go: Pattern matching
func (g *Generator) generateExpr(expr core.CoreExpr) error {
	switch e := expr.(type) {
	case *core.Lit:
		return g.generateLit(e)

	case *core.Var:
		return g.generateVar(e)

	case *core.VarGlobal:
		return g.generateVarGlobal(e)

	case *core.Lambda:
		return g.generateLambda(e)

	case *core.App:
		return g.generateApp(e)

	case *core.Let:
		return g.generateLet(e)

	case *core.LetRec:
		return g.generateLetRec(e)

	case *core.If:
		return g.generateIf(e)

	case *core.Match:
		return g.generateMatch(e)

	case *core.BinOp:
		return g.generateBinOp(e)

	case *core.UnOp:
		return g.generateUnOp(e)

	case *core.Record:
		return g.generateRecord(e)

	case *core.RecordAccess:
		return g.generateRecordAccess(e)

	case *core.RecordUpdate:
		return g.generateRecordUpdate(e)

	case *core.List:
		return g.generateList(e)

	case *core.Array:
		return g.generateArray(e)

	case *core.Tuple:
		return g.generateTuple(e)

	case *core.Intrinsic:
		return g.generateIntrinsic(e)

	case *core.DictRef:
		// Dictionary references are runtime-resolved
		g.writef("dict_%s_%s", e.ClassName, e.TypeName)
		return nil

	case *core.DictApp:
		return g.generateDictApp(e)

	default:
		return fmt.Errorf("unsupported expression type: %T", expr)
	}
}
