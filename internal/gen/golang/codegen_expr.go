// Package golang provides Go code generation from AILANG Core AST.
package golang

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/core"
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

// needsBoolAssertion returns true if the expression returns interface{} and
// needs a .(bool) type assertion when used in a boolean context.
// M-CODEGEN-BOOL-ASSERTIONS: Detects DictApp calls and interface{}-typed variables.
func (g *Generator) needsBoolAssertion(expr core.CoreExpr) bool {
	switch e := expr.(type) {
	case *core.DictApp:
		// Dictionary method calls return interface{}
		// Check if this is a comparison method (Eq, Neq, Lt, Gt, Lte, Gte)
		switch e.Method {
		case "eq", "neq", "lt", "gt", "lte", "gte":
			return true
		}
		return false

	case *core.Var:
		// Check if this variable is known to hold an interface{} value
		// Variables bound to dictionary results need assertions
		goVarName := ToGoVarName(e.Name)

		// If variable is in typedLocalVars, it has concrete type from ADT field extraction
		if _, isTyped := g.typedLocalVars[goVarName]; isTyped {
			return false
		}

		// Check if it's a typed function parameter with concrete type
		if goType, ok := g.currentFuncParams[e.Name]; ok {
			// Only need assertion if param is interface{}
			return goType == "interface{}"
		}

		// Not in typed locals or typed params - might be interface{} from Let binding
		// Conservative: assume it needs assertion in boolean context
		return true

	case *core.BinOp:
		// Comparison operators that go through dictionaries
		switch e.Op {
		case "==", "!=", "<", ">", "<=", ">=":
			// These might be lowered to DictApp, check the type
			return false // The lowered form will be caught by DictApp case
		}
		return false

	default:
		return false
	}
}

// generateExprWithBoolAssertion generates an expression and appends .(bool)
// if the expression returns interface{} in a boolean context.
// M-CODEGEN-BOOL-ASSERTIONS: Used for if conditions, logical operators, bool returns.
func (g *Generator) generateExprWithBoolAssertion(expr core.CoreExpr) error {
	needsAssertion := g.needsBoolAssertion(expr)

	if err := g.generateExpr(expr); err != nil {
		return err
	}

	if needsAssertion {
		g.write(".(bool)")
	}

	return nil
}
