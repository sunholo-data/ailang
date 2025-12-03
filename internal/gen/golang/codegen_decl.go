// Package golang provides Go code generation from AILANG Core AST.
package golang

import (
	"fmt"
	"strings"

	"github.com/sunholo/ailang/internal/core"
)

// generateDecl generates Go code for a Core declaration.
func (g *Generator) generateDecl(decl core.CoreExpr) error {
	switch d := decl.(type) {
	case *core.Let:
		return g.generateTopLevelLet(d)
	case *core.LetRec:
		return g.generateTopLevelLetRec(d)
	default:
		// Skip other declaration types for now
		return nil
	}
}

// isExported checks if a declaration name is marked for export.
func (g *Generator) isExported(name string) bool {
	if g.prog == nil || g.prog.Meta == nil {
		return false
	}
	if meta, ok := g.prog.Meta[name]; ok {
		return meta.IsExport
	}
	return false
}

// generateTopLevelLet generates a top-level let binding as a Go var or func.
func (g *Generator) generateTopLevelLet(let *core.Let) error {
	exported := g.isExported(let.Name)

	// Check if it's a function (Lambda body)
	if lam, ok := let.Value.(*core.Lambda); ok {
		return g.generateFuncFromLambda(let.Name, lam, exported)
	}

	// Otherwise generate as a variable
	varName := ToGoFuncName(let.Name, exported) // Use same naming for consistency
	g.writef("var %s = ", varName)
	if err := g.generateExpr(let.Value); err != nil {
		return err
	}
	g.writef("\n\n")
	return nil
}

// generateTopLevelLetRec generates recursive function bindings.
func (g *Generator) generateTopLevelLetRec(letrec *core.LetRec) error {
	for _, bind := range letrec.Bindings {
		if lam, ok := bind.Value.(*core.Lambda); ok {
			exported := g.isExported(bind.Name)
			if err := g.generateFuncFromLambda(bind.Name, lam, exported); err != nil {
				return err
			}
		}
	}
	return nil
}

// generateFuncFromLambda generates a Go function from a Lambda expression.
// M-DX23: When CoreTypeInfo is available, generates typed signatures instead of interface{}.
// M-DX24: Wraps return expression with type assertion when needed.
func (g *Generator) generateFuncFromLambda(name string, lam *core.Lambda, exported bool) error {
	funcName := ToGoFuncName(name, exported)

	// Register this function name mapping for recursive references
	g.topLevelFuncs[name] = funcName

	// M-DX23: Try to get typed signature from CoreTypeInfo
	paramTypes, returnType := g.getTypedSignature(lam)

	// Build parameter list
	var params []string
	for i, p := range lam.Params {
		var paramType string
		if i < len(paramTypes) {
			paramType = string(paramTypes[i])
		} else {
			paramType = "interface{}"
		}
		params = append(params, fmt.Sprintf("%s %s", ToGoVarName(p), paramType))
	}

	// Use typed return type or fall back to interface{}
	retType := "interface{}"
	if returnType != "" {
		retType = string(returnType)
	}

	// M-DX24: Set expected return type for type assertion generation
	oldExpectedReturn := g.expectedReturnType
	g.expectedReturnType = GoType(retType)

	g.writef("func %s(%s) %s {\n", funcName, strings.Join(params, ", "), retType)
	g.indent++
	g.writef("return ")
	if err := g.generateExpr(lam.Body); err != nil {
		g.expectedReturnType = oldExpectedReturn
		return err
	}

	// M-DX24: Add type assertion if return type is concrete and expression returns interface{}
	if g.needsReturnTypeAssertion(lam.Body) {
		g.writef(".(%s)", retType)
	}

	g.writef("\n")
	g.indent--
	g.writef("}\n\n")

	// Restore previous expected return type
	g.expectedReturnType = oldExpectedReturn

	return nil
}

// needsReturnTypeAssertion checks if a return expression needs a type assertion.
// M-DX24: Returns true if expectedReturnType is concrete and expression produces interface{}.
func (g *Generator) needsReturnTypeAssertion(expr core.CoreExpr) bool {
	// No assertion needed if expected type is interface{}
	if g.expectedReturnType == "" || g.expectedReturnType == "interface{}" {
		return false
	}

	// Check if the expression produces interface{} from runtime helpers
	return g.exprProducesInterface(expr)
}

// exprProducesInterface checks if an expression produces interface{} type.
// M-DX24: Used to determine if type assertion is needed.
// M-DX25.8: Now uses CoreTypeInfo to accurately determine expression types.
func (g *Generator) exprProducesInterface(expr core.CoreExpr) bool {
	// M-DX25.8: Use CoreTypeInfo if available for accurate type checking
	if g.coreTypeInfo != nil {
		nodeID := g.getExprNodeID(expr)
		if typ, ok := g.coreTypeInfo[nodeID]; ok {
			if goType, err := g.TypeMapper.MapType(typ); err == nil {
				// If we can map to a concrete Go type, it doesn't produce interface{}
				return string(goType) == "interface{}"
			}
		}
	}

	// Fallback to heuristic-based checking when no type info available
	switch e := expr.(type) {
	case *core.Lit:
		// Literals produce concrete types, no assertion needed
		return false

	case *core.Var, *core.VarGlobal:
		// Variables could be either - for now assume they match expected type
		// since we generate typed parameters
		return false

	case *core.BinOp:
		// Without type info, assume binary ops might need assertion
		return true

	case *core.UnOp:
		// Without type info, assume unary ops might need assertion
		return true

	case *core.If:
		// If expression result depends on branches
		// Both branches should have same type, check the then branch
		return g.exprProducesInterface(e.Then)

	case *core.App:
		// M-DX24.2: If this will be emitted as a native op, it produces concrete type
		if g.canEmitNativeOp(e) {
			return false
		}
		// Function applications could return interface{} or concrete
		// For now, assume ADT constructors and runtime helpers return interface{}
		// User functions with typed signatures return concrete types
		return g.appProducesInterface(e)

	case *core.Match:
		// Match expression result depends on branches - check first arm
		if len(e.Arms) > 0 {
			return g.exprProducesInterface(e.Arms[0].Body)
		}
		return true

	case *core.Let:
		// Let expression result is the body
		return g.exprProducesInterface(e.Body)

	case *core.Lambda:
		// Lambda itself is a function value - no assertion needed
		return false

	case *core.Intrinsic:
		// Without type info, assume intrinsics might need assertion
		return true

	case *core.Record:
		// Records produce concrete struct types (if typed) or map[string]interface{}
		return false

	case *core.List:
		// Without type info, assume lists produce []interface{}
		return true

	case *core.Tuple:
		// Tuples produce concrete tuple types
		return false

	default:
		// Default: assume interface{} to be safe
		return true
	}
}

// appProducesInterface checks if a function application produces interface{}.
// M-DX24: ADT constructors and typed runtime helpers return concrete types.
func (g *Generator) appProducesInterface(app *core.App) bool {
	// Check if this is a known ADT constructor or runtime helper
	if varGlobal, ok := app.Func.(*core.VarGlobal); ok {
		name := varGlobal.Ref.Name
		if _, isADT := g.adtConstructors[name]; isADT {
			// ADT constructors return *ADT, not interface{}
			return false
		}
		// Check if it's a known top-level function (user-defined with typed signature)
		if _, isTopLevel := g.topLevelFuncs[name]; isTopLevel {
			// User functions may return concrete types
			return false
		}
		// M-DX24: Check if it's a runtime helper that returns a concrete type
		if retType := runtimeHelperReturnType(name); retType != "" && retType != "interface{}" {
			return false
		}
	}

	// Unknown function - assume returns interface{}
	return true
}

// runtimeHelperReturnType returns the Go return type for known runtime helpers.
// M-DX24: Used to determine if type assertion is needed at return boundaries.
// Returns empty string if the helper is unknown or returns interface{}.
func runtimeHelperReturnType(name string) string {
	// Map builtin/runtime helper names to their concrete return types
	// These are the internal names used in Core AST (before ToPascalCase)
	switch name {
	// String operations - return string
	case "concat_String", "_str_concat":
		return "string"
	case "show", "_show":
		return "string"
	case "_str_len", "str_len":
		return "int64"
	case "_str_slice", "str_slice":
		return "string"
	case "_str_at", "str_at":
		return "string"
	case "_str_indexOf", "str_indexOf":
		return "int64"
	case "_str_contains", "str_contains":
		return "bool"
	case "_str_startsWith", "str_startsWith":
		return "bool"
	case "_str_endsWith", "str_endsWith":
		return "bool"
	case "_str_toUpper", "str_toUpper":
		return "string"
	case "_str_toLower", "str_toLower":
		return "string"
	case "_str_trim", "str_trim":
		return "string"
	case "_str_split", "str_split":
		return "[]string"

	// Conversion operations
	case "_stringToInt", "stringToInt":
		return "int64"
	case "_stringToFloat", "stringToFloat":
		return "float64"
	case "_intToString", "intToString":
		return "string"
	case "_floatToString", "floatToString":
		return "string"

	// List operations with concrete return types
	case "ListLen", "_list_len", "list_len":
		return "int"

	// Type conversion helpers
	case "toInt64":
		return "int64"
	case "toFloat64":
		return "float64"

	default:
		// Unknown helper or returns interface{} (e.g., AddInt, SubInt, etc.)
		return ""
	}
}

// getTypedSignature extracts typed parameter and return types from CoreTypeInfo.
// M-DX23: Returns nil/empty if type info is unavailable or not a function type.
func (g *Generator) getTypedSignature(lam *core.Lambda) ([]GoType, GoType) {
	if g.coreTypeInfo == nil {
		return nil, ""
	}

	// Look up the Lambda's type by NodeID
	lamType, ok := g.coreTypeInfo[lam.NodeID]
	if !ok {
		return nil, ""
	}

	// Extract function signature using TypeMapper
	paramTypes, returnType, ok := g.TypeMapper.ExtractFuncSignature(lamType)
	if !ok {
		return nil, ""
	}

	return paramTypes, returnType
}
