// Package golang provides Go code generation from AILANG Core AST.
package golang

import (
	"github.com/sunholo/ailang/internal/core"
)

// generateBinOp generates a Go binary operation.
func (g *Generator) generateBinOp(binop *core.BinOp) error {
	// Handle cons operator specially - it's not a simple binary op in Go
	if binop.Op == "::" {
		g.write("Cons(")
		if err := g.generateExpr(binop.Left); err != nil {
			return err
		}
		g.write(", ")
		if err := g.generateExpr(binop.Right); err != nil {
			return err
		}
		g.write(")")
		return nil
	}

	g.write("(")
	if err := g.generateExpr(binop.Left); err != nil {
		return err
	}
	g.writef(" %s ", g.mapOperator(binop.Op))
	if err := g.generateExpr(binop.Right); err != nil {
		return err
	}
	g.write(")")
	return nil
}

// generateUnOp generates a Go unary operation.
func (g *Generator) generateUnOp(unop *core.UnOp) error {
	g.writef("%s", g.mapUnaryOperator(unop.Op))
	return g.generateExpr(unop.Operand)
}

// generateRecord generates a Go struct literal.
func (g *Generator) generateRecord(rec *core.Record) error {
	g.write("map[string]interface{}{")
	first := true
	for name, value := range rec.Fields {
		if !first {
			g.write(", ")
		}
		first = false
		g.writef("%q: ", name)
		if err := g.generateExpr(value); err != nil {
			return err
		}
	}
	g.write("}")
	return nil
}

// generateRecordAccess generates field access.
func (g *Generator) generateRecordAccess(ra *core.RecordAccess) error {
	if err := g.generateExpr(ra.Record); err != nil {
		return err
	}
	g.writef(".(map[string]interface{})[%q]", ra.Field)
	return nil
}

// generateRecordUpdate generates Go code for a record update expression.
// AILANG: { base | field1: val1, field2: val2 }
// Go: RecordUpdate(base, map[string]interface{}{"field1": val1, ...})
func (g *Generator) generateRecordUpdate(ru *core.RecordUpdate) error {
	g.write("RecordUpdate(")
	if err := g.generateExpr(ru.Base); err != nil {
		return err
	}
	g.write(", map[string]interface{}{")

	first := true
	for field, val := range ru.Updates {
		if !first {
			g.write(", ")
		}
		first = false
		g.writef("%q: ", field)
		if err := g.generateExpr(val); err != nil {
			return err
		}
	}

	g.write("})")
	return nil
}

// generateList generates a Go slice literal.
func (g *Generator) generateList(list *core.List) error {
	g.write("[]interface{}{")
	for i, elem := range list.Elements {
		if i > 0 {
			g.write(", ")
		}
		if err := g.generateExpr(elem); err != nil {
			return err
		}
	}
	g.write("}")
	return nil
}

// generateTuple generates a Go struct for tuple.
func (g *Generator) generateTuple(tuple *core.Tuple) error {
	g.write("[]interface{}{")
	for i, elem := range tuple.Elements {
		if i > 0 {
			g.write(", ")
		}
		if err := g.generateExpr(elem); err != nil {
			return err
		}
	}
	g.write("}")
	return nil
}

// generateIntrinsic generates code for intrinsic operations.
func (g *Generator) generateIntrinsic(intr *core.Intrinsic) error {
	op := g.mapIntrinsicOp(intr.Op)
	if len(intr.Args) == 1 {
		// Unary
		g.writef("(%s", op)
		if err := g.generateExpr(intr.Args[0]); err != nil {
			return err
		}
		g.write(")")
	} else if len(intr.Args) == 2 {
		// Binary
		g.write("(")
		if err := g.generateExpr(intr.Args[0]); err != nil {
			return err
		}
		g.writef(" %s ", op)
		if err := g.generateExpr(intr.Args[1]); err != nil {
			return err
		}
		g.write(")")
	}
	return nil
}

// generateDictApp generates dictionary method application.
func (g *Generator) generateDictApp(da *core.DictApp) error {
	if err := g.generateExpr(da.Dict); err != nil {
		return err
	}
	g.writef(".%s(", ToPascalCase(da.Method))
	for i, arg := range da.Args {
		if i > 0 {
			g.write(", ")
		}
		if err := g.generateExpr(arg); err != nil {
			return err
		}
	}
	g.write(")")
	return nil
}

// mapOperator maps AILANG binary operators to Go operators.
func (g *Generator) mapOperator(op string) string {
	switch op {
	case "+", "-", "*", "/", "%":
		return op
	case "==", "!=", "<", ">", "<=", ">=":
		return op
	case "&&", "||":
		return op
	case "++":
		return "+" // Works for strings
	default:
		return op
	}
}

// mapUnaryOperator maps AILANG unary operators to Go operators.
func (g *Generator) mapUnaryOperator(op string) string {
	switch op {
	case "-":
		return "-"
	case "!":
		return "!"
	default:
		return op
	}
}

// mapIntrinsicOp maps intrinsic operations to Go operators.
func (g *Generator) mapIntrinsicOp(op core.IntrinsicOp) string {
	switch op {
	case core.OpAdd:
		return "+"
	case core.OpSub:
		return "-"
	case core.OpMul:
		return "*"
	case core.OpDiv:
		return "/"
	case core.OpMod:
		return "%"
	case core.OpEq:
		return "=="
	case core.OpNe:
		return "!="
	case core.OpLt:
		return "<"
	case core.OpLe:
		return "<="
	case core.OpGt:
		return ">"
	case core.OpGe:
		return ">="
	case core.OpConcat:
		return "+"
	case core.OpAnd:
		return "&&"
	case core.OpOr:
		return "||"
	case core.OpNot:
		return "!"
	case core.OpNeg:
		return "-"
	default:
		return "?"
	}
}
