// Package golang provides Go code generation from AILANG Core AST.
package golang

import (
	"github.com/sunholo/ailang/internal/core"
)

// generateIf generates a Go if expression.
// M-DX25.3: Uses typed IIFE return and conditional type assertions.
// M-DX26: In _impl functions, uses interface{} everywhere.
func (g *Generator) generateIf(ifExpr *core.If) error {
	// M-DX26: In _impl functions, everything is interface{}
	inImplFunc := g.expectedReturnType == "interface{}"

	// M-DX25.3: Look up If expression's type for IIFE return type
	returnType := "interface{}"
	// M-DX26: Skip type inference in _impl functions
	if !inImplFunc {
		// M-DX25.10: Special case for Record branches - infer type from fields
		// Check Then branch first (both branches should have same type)
		if rec, isRec := ifExpr.Then.(*core.Record); isRec {
			fieldNames := make(map[string]bool, len(rec.Fields))
			for name := range rec.Fields {
				fieldNames[name] = true
			}
			if recordType := g.GetRecordTypeByFields(fieldNames); recordType != nil {
				returnType = "*" + recordType.Name
			}
		} else if g.coreTypeInfo != nil {
			if typ, ok := g.coreTypeInfo[ifExpr.NodeID]; ok {
				if goType, err := g.TypeMapper.MapType(typ); err == nil {
					returnType = string(goType)
				}
			}
		}
	}

	g.writef("func() %s {\n", returnType)
	g.indent++
	g.writef("if ")
	if err := g.generateExpr(ifExpr.Cond); err != nil {
		return err
	}
	// M-DX25.3: Only add .(bool) if condition produces interface{}
	if g.exprProducesInterface(ifExpr.Cond) {
		g.write(".(bool)")
	}
	g.write(" {\n")
	g.indent++
	g.writef("return ")
	if err := g.generateExpr(ifExpr.Then); err != nil {
		return err
	}
	// M-DX25.3: Add type assertion if Then branch produces interface{} but we need concrete type
	if returnType != "interface{}" && g.exprProducesInterface(ifExpr.Then) {
		g.writef(".(%s)", returnType)
	}
	g.writef("\n")
	g.indent--
	g.writef("}\n")
	g.writef("return ")
	if err := g.generateExpr(ifExpr.Else); err != nil {
		return err
	}
	// M-DX25.3: Add type assertion if Else branch produces interface{} but we need concrete type
	if returnType != "interface{}" && g.exprProducesInterface(ifExpr.Else) {
		g.writef(".(%s)", returnType)
	}
	g.writef("\n")
	g.indent--
	g.write("}()")
	return nil
}

// canEmitNativeOp checks if an App can be emitted as a native Go operator.
// M-DX24.2: Returns true for arithmetic/comparison helpers when operands have known types.
// M-DX26: Returns false in _impl functions where all params are interface{}.
func (g *Generator) canEmitNativeOp(app *core.App) bool {
	// M-DX26: In _impl functions (interface{} world), never emit native ops
	// All params are interface{}, so Go operators won't work
	if g.expectedReturnType == "interface{}" {
		return false
	}

	// Must have exactly 2 arguments for binary ops
	if len(app.Args) != 2 {
		return false
	}

	// Check if function is a known arithmetic/comparison helper
	funcName := g.getAppFuncName(app)
	if funcName == "" {
		return false
	}

	// Check if this is an arithmetic/comparison helper
	op := arithmeticHelperToOp(funcName)
	if op == "" {
		return false
	}

	// Check if both operands have known types
	// For now, we check if operands are:
	// 1. Typed parameters (Var with known type)
	// 2. Literals (always typed)
	// 3. Other expressions that produce concrete types
	return g.operandHasKnownType(app.Args[0]) && g.operandHasKnownType(app.Args[1])
}

// generateNativeOp generates a native Go operator expression.
// M-DX24.2: Emits (a + b) instead of AddInt(a, b).
func (g *Generator) generateNativeOp(app *core.App) error {
	funcName := g.getAppFuncName(app)
	op := arithmeticHelperToOp(funcName)

	g.write("(")
	if err := g.generateExpr(app.Args[0]); err != nil {
		return err
	}
	g.writef(" %s ", op)
	if err := g.generateExpr(app.Args[1]); err != nil {
		return err
	}
	g.write(")")
	return nil
}

// operandHasKnownType checks if an operand has a known concrete type.
// M-DX24.2: Used to determine if we can emit native operators.
func (g *Generator) operandHasKnownType(expr core.CoreExpr) bool {
	switch e := expr.(type) {
	case *core.Lit:
		// Literals always have concrete types
		return true

	case *core.Var:
		// Variables are typed if they're function parameters
		// We can't easily check this at codegen time, so we're conservative
		// and assume local variables are typed (they come from typed function params)
		return true

	case *core.VarGlobal:
		// Global variables might be typed
		return true

	case *core.App:
		// Function calls - check if the function returns a concrete type
		funcName := g.getAppFuncName(e)
		if retType := runtimeHelperReturnType(funcName); retType != "" && retType != "interface{}" {
			return true
		}
		// ADT constructors return concrete types
		if _, isADT := g.adtConstructors[funcName]; isADT {
			return true
		}
		// Top-level functions may return concrete types
		if _, isTopLevel := g.topLevelFuncs[funcName]; isTopLevel {
			return true
		}
		// Arithmetic helpers we're about to emit as native ops
		if arithmeticHelperToOp(funcName) != "" {
			return true
		}
		return false

	default:
		return false
	}
}

// arithmeticHelperToOp maps arithmetic helper function names to Go operators.
// M-DX24.2: Returns empty string if not a known arithmetic helper.
func arithmeticHelperToOp(name string) string {
	switch name {
	// Integer arithmetic
	case "add_Int", "AddInt":
		return "+"
	case "sub_Int", "SubInt":
		return "-"
	case "mul_Int", "MulInt":
		return "*"
	case "div_Int", "DivInt":
		return "/"
	case "mod_Int", "ModInt":
		return "%"

	// Float arithmetic
	case "add_Float", "AddFloat":
		return "+"
	case "sub_Float", "SubFloat":
		return "-"
	case "mul_Float", "MulFloat":
		return "*"
	case "div_Float", "DivFloat":
		return "/"

	// Integer comparisons
	case "eq_Int", "EqInt":
		return "=="
	case "ne_Int", "NeInt":
		return "!="
	case "lt_Int", "LtInt":
		return "<"
	case "le_Int", "LeInt":
		return "<="
	case "gt_Int", "GtInt":
		return ">"
	case "ge_Int", "GeInt":
		return ">="

	// Float comparisons
	case "eq_Float", "EqFloat":
		return "=="
	case "ne_Float", "NeFloat":
		return "!="
	case "lt_Float", "LtFloat":
		return "<"
	case "le_Float", "LeFloat":
		return "<="
	case "gt_Float", "GtFloat":
		return ">"
	case "ge_Float", "GeFloat":
		return ">="

	// Boolean operations
	case "and_Bool", "AndBool":
		return "&&"
	case "or_Bool", "OrBool":
		return "||"

	default:
		return ""
	}
}
