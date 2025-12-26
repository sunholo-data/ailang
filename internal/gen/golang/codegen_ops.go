// Package golang provides Go code generation from AILANG Core AST.
package golang

import (
	"fmt"

	"github.com/sunholo/ailang/internal/core"
)

// generateBinOp generates a Go binary operation.
// M-VERIFY: When in _impl context (interface{} world), comparison operators
// use runtime helpers to avoid type errors with interface{} operands.
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

	// M-VERIFY: In _impl functions (interface{} world), comparison operators
	// need runtime helpers because Go doesn't allow comparing interface{} values
	// directly with typed literals.
	if g.expectedReturnType == "interface{}" {
		helper := g.mapComparisonToHelper(binop.Op)
		if helper != "" {
			g.writef("%s(", helper)
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
	}

	// M-CODEGEN-BOOL-ASSERTIONS: Logical operators need boolean operands
	// Both && and || require bool operands in Go, so add type assertions if needed
	if binop.Op == "&&" || binop.Op == "||" {
		g.write("(")
		if err := g.generateExprWithBoolAssertion(binop.Left); err != nil {
			return err
		}
		g.writef(" %s ", g.mapOperator(binop.Op))
		if err := g.generateExprWithBoolAssertion(binop.Right); err != nil {
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

// mapComparisonToHelper returns the runtime helper for a comparison operator.
// M-VERIFY: Used for contract predicate checks where operands are interface{}.
// Returns empty string if operator is not a comparison.
func (g *Generator) mapComparisonToHelper(op string) string {
	switch op {
	case ">=":
		return "GeInt" // TODO: Need type-aware dispatch
	case ">":
		return "GtInt"
	case "<=":
		return "LeInt"
	case "<":
		return "LtInt"
	case "==":
		return "EqInt"
	case "!=":
		return "NeInt"
	default:
		return ""
	}
}

// generateUnOp generates a Go unary operation.
func (g *Generator) generateUnOp(unop *core.UnOp) error {
	g.writef("%s", g.mapUnaryOperator(unop.Op))
	return g.generateExpr(unop.Operand)
}

// NOTE: Record-related functions (generateRecord, generateTypedRecord, generateTypedRecordValue,
// generateRecordAccess, generateRecordUpdate, helper functions) moved to codegen_record.go

// generateList generates a Go slice literal.
// M-DX25.11: Uses typed slices when element type is known from CoreTypeInfo.
// M-CODEGEN-LIST: Flattens nested Let expressions to reduce IIFE nesting.
func (g *Generator) generateList(list *core.List) error {
	// M-CODEGEN-LIST: Check if any elements need flattening (contain Let expressions)
	needsFlattening := false
	for _, elem := range list.Elements {
		if containsLet(elem) {
			needsFlattening = true
			break
		}
	}

	if needsFlattening {
		// M-CODEGEN-LIST: Wrap in single IIFE that flattens all element computations
		// This reduces O(n) nesting to O(1) nesting
		return g.generateFlattenedList(list)
	}

	// M-DX26: In _impl functions, always generate []interface{}
	inImplFunc := g.expectedReturnType == "interface{}"

	// M-DX25.11: Try to determine element type from CoreTypeInfo
	elemType := ""
	if !inImplFunc {
		elemType = g.getListElementType(list)
	}

	if elemType != "" && elemType != "interface{}" {
		// Generate typed slice (e.g., []int64{1, 2, 3})
		g.writef("[]%s{", elemType)
	} else {
		// Fallback to interface{} slice
		g.write("[]interface{}{")
	}

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

// containsLet checks if an expression contains a Let binding.
// M-CODEGEN-LIST: Used to detect when list elements need flattening.
func containsLet(expr core.CoreExpr) bool {
	switch e := expr.(type) {
	case *core.Let:
		return true
	case *core.LetRec:
		return true
	case *core.App:
		// Check function and all arguments
		if containsLet(e.Func) {
			return true
		}
		for _, arg := range e.Args {
			if containsLet(arg) {
				return true
			}
		}
	case *core.If:
		return containsLet(e.Cond) || containsLet(e.Then) || containsLet(e.Else)
	case *core.Match:
		if containsLet(e.Scrutinee) {
			return true
		}
		for _, arm := range e.Arms {
			if containsLet(arm.Body) {
				return true
			}
		}
	case *core.BinOp:
		return containsLet(e.Left) || containsLet(e.Right)
	case *core.UnOp:
		return containsLet(e.Operand)
	case *core.List:
		for _, elem := range e.Elements {
			if containsLet(elem) {
				return true
			}
		}
	case *core.Array:
		for _, elem := range e.Elements {
			if containsLet(elem) {
				return true
			}
		}
	case *core.Tuple:
		for _, elem := range e.Elements {
			if containsLet(elem) {
				return true
			}
		}
	case *core.Record:
		for _, value := range e.Fields {
			if containsLet(value) {
				return true
			}
		}
	case *core.RecordAccess:
		return containsLet(e.Record)
	case *core.RecordUpdate:
		if containsLet(e.Base) {
			return true
		}
		for _, value := range e.Updates {
			if containsLet(value) {
				return true
			}
		}
	}
	return false
}

// generateFlattenedList generates a list with flattened Let bindings.
// M-CODEGEN-LIST: Wraps list in single IIFE, flattens all element computations.
//
// Instead of:
//
//	func() interface{} {
//	    var tmp1 = f()
//	    return func() interface{} {
//	        var tmp2 = g()
//	        return []interface{}{tmp1, tmp2}
//	    }()
//	}()
//
// We generate:
//
//	func() interface{} {
//	    var _list_e0 = f()
//	    var _list_e1 = g()
//	    return []interface{}{_list_e0, _list_e1}
//	}()
func (g *Generator) generateFlattenedList(list *core.List) error {
	g.write("func() interface{} {\n")
	g.indent++

	// Evaluate all elements into temporary variables with flattened bindings
	varNames := make([]string, len(list.Elements))
	for i, elem := range list.Elements {
		varName := g.uniqueVarName("_list_e")
		varNames[i] = varName

		// Use flattenExprBindings to extract and emit Let bindings
		finalExpr := g.flattenExprBindings(elem)

		// Assign the final expression to our list element var
		g.writef("var %s interface{} = ", varName)
		if err := g.generateExpr(finalExpr); err != nil {
			return err
		}
		g.write("\n")
	}

	// Build the slice with flattened element references
	g.write("return []interface{}{")
	for i, varName := range varNames {
		if i > 0 {
			g.write(", ")
		}
		g.write(varName)
	}
	g.write("}\n")

	g.indent--
	g.write("}()")
	return nil
}

// flattenExprBindings extracts and emits Let bindings from an expression,
// returning the final non-Let expression.
// M-CODEGEN-LIST: Recursively flattens nested Let expressions.
func (g *Generator) flattenExprBindings(expr core.CoreExpr) core.CoreExpr {
	current := expr

	// Extract consecutive Let bindings
	for {
		let, ok := current.(*core.Let)
		if !ok {
			break
		}

		// Recursively flatten the value
		flatValue := g.flattenExprBindings(let.Value)

		// Emit this binding
		goName := ToGoVarName(let.Name)
		g.writef("var %s interface{} = ", goName)
		// Note: we call generateExpr here, which may still have some nesting
		// for complex expressions, but this is much better than before
		if err := g.generateExpr(flatValue); err != nil {
			// On error, just emit the original - fallback behavior
			if err := g.generateExpr(let.Value); err != nil {
				return nil // Still return nil to continue, error already recorded
			}
		}
		g.write("\n")

		// Move to the body for next iteration
		current = let.Body
	}

	return current
}

// uniqueVarName generates a unique variable name with the given prefix.
// M-CODEGEN-LIST: Prevents name collisions in flattened bindings.
func (g *Generator) uniqueVarName(prefix string) string {
	g.varCounter++
	return fmt.Sprintf("%s%d", prefix, g.varCounter)
}

// getListElementType extracts the Go element type for a list from CoreTypeInfo.
// M-DX25.11: Returns empty string if type is unknown or not a list type.
func (g *Generator) getListElementType(list *core.List) string {
	if g.coreTypeInfo == nil {
		return ""
	}

	nodeID := list.NodeID
	if nodeID == 0 {
		return ""
	}

	typ, ok := g.coreTypeInfo[nodeID]
	if !ok {
		return ""
	}

	// Map the type to Go and extract element type
	goType, err := g.TypeMapper.MapType(typ)
	if err != nil {
		return ""
	}

	// Check if it's a slice type and extract element type
	goTypeStr := string(goType)
	if len(goTypeStr) > 2 && goTypeStr[:2] == "[]" {
		return goTypeStr[2:]
	}

	return ""
}

// generateArray generates a Go slice literal for arrays.
// M-TYPE1: Arrays use the same Go representation as lists (slices).
func (g *Generator) generateArray(arr *core.Array) error {
	// M-DX26: In _impl functions, always generate []interface{}
	inImplFunc := g.expectedReturnType == "interface{}"

	// Try to determine element type from CoreTypeInfo
	elemType := ""
	if !inImplFunc {
		elemType = g.getArrayElementType(arr)
	}

	if elemType != "" && elemType != "interface{}" {
		// Generate typed slice (e.g., []int64{1, 2, 3})
		g.writef("[]%s{", elemType)
	} else {
		// Fallback to interface{} slice
		g.write("[]interface{}{")
	}

	for i, elem := range arr.Elements {
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

// getArrayElementType extracts the Go element type for an array from CoreTypeInfo.
// M-TYPE1: Returns empty string if type is unknown or not an array type.
func (g *Generator) getArrayElementType(arr *core.Array) string {
	if g.coreTypeInfo == nil {
		return ""
	}

	nodeID := arr.NodeID
	if nodeID == 0 {
		return ""
	}

	typ, ok := g.coreTypeInfo[nodeID]
	if !ok {
		return ""
	}

	// Map the type to Go and extract element type
	goType, err := g.TypeMapper.MapType(typ)
	if err != nil {
		return ""
	}

	// Check if it's a slice type and extract element type
	goTypeStr := string(goType)
	if len(goTypeStr) > 2 && goTypeStr[:2] == "[]" {
		return goTypeStr[2:]
	}

	return ""
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
// M-VERIFY: In _impl context (interface{}), comparison operators use runtime helpers.
func (g *Generator) generateIntrinsic(intr *core.Intrinsic) error {
	// M-VERIFY: In _impl functions (interface{} world), comparison operators
	// need runtime helpers because Go doesn't allow comparing interface{} values
	// directly with typed literals.
	if g.expectedReturnType == "interface{}" && len(intr.Args) == 2 {
		helper := g.mapIntrinsicToHelper(intr.Op)
		if helper != "" {
			g.writef("%s(", helper)
			if err := g.generateExpr(intr.Args[0]); err != nil {
				return err
			}
			g.write(", ")
			if err := g.generateExpr(intr.Args[1]); err != nil {
				return err
			}
			g.write(")")
			return nil
		}
	}

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

// mapIntrinsicToHelper returns the runtime helper for a comparison intrinsic.
// M-VERIFY: Used for contract predicate checks where operands are interface{}.
// Returns empty string if not a comparison operator.
func (g *Generator) mapIntrinsicToHelper(op core.IntrinsicOp) string {
	switch op {
	case core.OpGe:
		return "GeInt" // TODO: Need type-aware dispatch
	case core.OpGt:
		return "GtInt"
	case core.OpLe:
		return "LeInt"
	case core.OpLt:
		return "LtInt"
	case core.OpEq:
		return "EqInt"
	case core.OpNe:
		return "NeInt"
	default:
		return ""
	}
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
