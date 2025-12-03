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

	// Build parameter list and store parameter types for call site assertions
	var params []string
	var paramTypeStrings []string // M-DX25.10: Store for call site assertions
	for i, p := range lam.Params {
		var paramType string
		if i < len(paramTypes) {
			paramType = string(paramTypes[i])
		} else {
			paramType = "interface{}"
		}
		params = append(params, fmt.Sprintf("%s %s", ToGoVarName(p), paramType))
		paramTypeStrings = append(paramTypeStrings, paramType)
	}

	// M-DX25.10: Store parameter types for call site type assertions
	g.funcParamTypes[name] = paramTypeStrings

	// M-DX25.10: Store current function's param names → Go types for exprProducesInterface
	// Clear any previous function's params and populate with current function's
	g.currentFuncParams = make(map[string]string)
	for i, p := range lam.Params {
		g.currentFuncParams[p] = paramTypeStrings[i]
	}

	// Use typed return type or fall back to interface{}
	retType := "interface{}"
	if returnType != "" {
		retType = string(returnType)
	}

	// M-DX25.10: Store return type for function call result assertions
	g.funcReturnTypes[name] = retType

	// M-DX24: Set expected return type for type assertion generation
	oldExpectedReturn := g.expectedReturnType
	g.expectedReturnType = GoType(retType)

	g.writef("func %s(%s) %s {\n", funcName, strings.Join(params, ", "), retType)
	g.indent++

	// M-DX25.11: Check if we need to wrap return expression in a slice converter
	needsSliceConv := g.needsReturnSliceConversion(lam.Body)
	if needsSliceConv {
		sliceConv := g.getSliceConversion(retType)
		if sliceConv != "" {
			g.writef("return %s(", sliceConv)
		} else {
			// No converter available - generate without conversion
			g.writef("return ")
			needsSliceConv = false
		}
	} else {
		g.writef("return ")
	}

	if err := g.generateExpr(lam.Body); err != nil {
		g.expectedReturnType = oldExpectedReturn
		return err
	}

	// Close the slice conversion function call
	if needsSliceConv {
		g.write(")")
	} else if g.needsReturnTypeAssertion(lam.Body) {
		// M-DX24: Add type assertion if return type is concrete and expression returns interface{}
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

	// M-DX25.11: If return type is a slice, check for slice type mismatch
	// Slice type assertions don't work in Go (slices are invariant), so we use converters
	if strings.HasPrefix(string(g.expectedReturnType), "[]") {
		return false // Handled by needsReturnSliceConversion
	}

	// Check if the expression produces interface{} from runtime helpers
	return g.exprProducesInterface(expr)
}

// needsReturnSliceConversion checks if a return expression needs a slice converter.
// M-DX25.11: Returns true if return type is []ConcreteType but expression produces []interface{}.
func (g *Generator) needsReturnSliceConversion(expr core.CoreExpr) bool {
	// Check if return type is a concrete slice
	if !strings.HasPrefix(string(g.expectedReturnType), "[]") {
		return false
	}

	// Don't convert if return type is already []interface{}
	if g.expectedReturnType == "[]interface{}" {
		return false
	}

	// Check if expression produces []interface{}
	return g.exprProducesInterfaceSlice(expr)
}

// exprProducesInterfaceSlice checks if an expression produces []interface{} type.
// M-DX25.11: Used to determine if slice conversion is needed.
func (g *Generator) exprProducesInterfaceSlice(expr core.CoreExpr) bool {
	switch e := expr.(type) {
	case *core.List:
		// Lists generate []interface{}{} unless typed
		// Check if we can determine the element type
		elemType := g.getListElementType(e)
		return elemType == "" || elemType == "interface{}"

	case *core.App:
		// Check if it's a Cons call (:: operator)
		if v, ok := e.Func.(*core.Var); ok && v.Name == "::" {
			return true // Cons always returns []interface{}
		}
		if v, ok := e.Func.(*core.VarGlobal); ok {
			if v.Ref.Name == "::" || v.Ref.Name == "Cons" {
				return true
			}
		}
		// Also check for VarGlobal with Cons name
		if v, ok := e.Func.(*core.VarGlobal); ok {
			if runtimeHelperReturnType(v.Ref.Name) == "[]interface{}" {
				return true
			}
		}
		// Check for Var with Cons
		if v, ok := e.Func.(*core.Var); ok {
			if runtimeHelperReturnType(v.Name) == "[]interface{}" {
				return true
			}
		}

	case *core.If:
		// If expression result depends on branches - check both
		return g.exprProducesInterfaceSlice(e.Then) || g.exprProducesInterfaceSlice(e.Else)

	case *core.Match:
		// Match expression - check first arm
		if len(e.Arms) > 0 {
			return g.exprProducesInterfaceSlice(e.Arms[0].Body)
		}

	case *core.Let:
		// Let expression result is the body
		return g.exprProducesInterfaceSlice(e.Body)
	}

	return false
}

// getListElementType is defined in codegen_ops.go

// exprProducesInterface checks if an expression produces interface{} type.
// M-DX24: Used to determine if type assertion is needed.
// M-DX25.8: Uses CoreTypeInfo for accurate type checking.
// M-DX25.10: Checks for runtime helper calls FIRST - these always produce interface{}
// regardless of what CoreTypeInfo says about the AILANG type.
func (g *Generator) exprProducesInterface(expr core.CoreExpr) bool {
	// M-DX25.10: Check for expressions that ALWAYS produce interface{} at runtime
	// These generate calls to runtime helpers that return interface{}, even though
	// the AILANG type may be concrete. Check BEFORE CoreTypeInfo lookup.
	switch e := expr.(type) {
	case *core.RecordAccess:
		// RecordAccess generates FieldGet() which always returns interface{}
		return true

	case *core.DictApp:
		// DictApp generates dict.Method() which always returns interface{}
		// This includes NegInt, FieldGet, and other type class methods
		return true

	case *core.App:
		// M-DX25.10: App calls to runtime builtins return interface{}
		// Op-lowering transforms Intrinsic{OpNeg} to App{VarGlobal{"neg_Int"}}
		// The generated NegInt(), AddInt(), etc. all return interface{}
		if g.appCallsRuntimeBuiltin(e) {
			return true
		}
		// M-DX25.10: Check if calling a user-defined function that returns interface{}
		// This must be checked BEFORE CoreTypeInfo because the AILANG type might be
		// concrete but the Go function was generated with interface{} return.
		if g.appCallsInterfaceReturningFunc(e) {
			return true
		}
		// M-DX25.10: Check if calling a function stored in a variable (generates CallFunc)
		// CallFunc returns interface{} regardless of the function's actual return type
		if g.appUsesCallFunc(e) {
			return true
		}
		// M-DX25.10: Check if calling a runtime helper that returns a concrete (non-interface{}) type
		// This must be checked BEFORE CoreTypeInfo to handle builtins like Cons that return []interface{}
		if g.appCallsConcreteReturningHelper(e) {
			return false
		}

	case *core.Var:
		// M-DX25.10: Check if this Var is a function parameter generated as interface{}
		// This must be checked BEFORE CoreTypeInfo because the AILANG type might be
		// concrete but the Go parameter was generated as interface{}.
		if goType, ok := g.currentFuncParams[e.Name]; ok {
			return goType == "interface{}"
		}
	}

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
		// M-DX25.10 FIX: If we reach this fallback, CoreTypeInfo didn't have the type.
		// This means the variable might have been declared as interface{} in Go.
		// Return true to be safe - better to have unnecessary type assertions
		// than compile errors from missing assertions.
		return true

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
		// M-DX25.10: Check if we know the function's actual return type
		if retType, found := g.funcReturnTypes[name]; found {
			return retType == "interface{}"
		}
		// Check if it's a known top-level function (user-defined with typed signature)
		if _, isTopLevel := g.topLevelFuncs[name]; isTopLevel {
			// User functions may return concrete types (fallback if no return type stored)
			return false
		}
		// M-DX24: Check if it's a runtime helper that returns a concrete type
		if retType := runtimeHelperReturnType(name); retType != "" && retType != "interface{}" {
			return false
		}
	}

	// Also check for Var (local function or builtin reference like ::)
	if v, ok := app.Func.(*core.Var); ok {
		// Check if it's a runtime helper that returns a concrete type
		if retType := runtimeHelperReturnType(v.Name); retType != "" && retType != "interface{}" {
			return false
		}
	}

	// Unknown function - assume returns interface{}
	return true
}

// appCallsRuntimeBuiltin checks if an App calls a runtime builtin that returns interface{}.
// M-DX25.10: Op-lowering transforms intrinsics like OpNeg to App{VarGlobal{"neg_Int"}}.
// These generate calls to NegInt(), AddInt(), etc. which all return interface{}.
// BUT: Binary ops that can be emitted as native Go operators return concrete types.
func (g *Generator) appCallsRuntimeBuiltin(app *core.App) bool {
	// M-DX24.2: If this will be emitted as a native Go operator, it produces concrete type
	// Binary ops like (x + y), (x >= y) return concrete types, not interface{}
	if g.canEmitNativeOp(app) {
		return false
	}

	varGlobal, ok := app.Func.(*core.VarGlobal)
	if !ok {
		return false
	}

	name := varGlobal.Ref.Name

	// Check for $builtin module (from op-lowering)
	if varGlobal.Ref.Module == "$builtin" {
		// M-DX25.10: Some builtins like :: (Cons) return concrete types, not interface{}
		if retType := runtimeHelperReturnType(name); retType != "" && retType != "interface{}" {
			return false // Returns concrete type, not interface{}
		}
		// All other op-lowered builtins return interface{}
		// Examples: neg_Int (unary), and binary ops when operands lack known types
		return true
	}

	// Check for common runtime helper patterns that return interface{}
	// These are the names after op-lowering (before ToPascalCase)
	switch {
	case strings.HasPrefix(name, "neg_"),
		strings.HasPrefix(name, "add_"),
		strings.HasPrefix(name, "sub_"),
		strings.HasPrefix(name, "mul_"),
		strings.HasPrefix(name, "div_"),
		strings.HasPrefix(name, "mod_"),
		strings.HasPrefix(name, "eq_"),
		strings.HasPrefix(name, "ne_"),
		strings.HasPrefix(name, "lt_"),
		strings.HasPrefix(name, "le_"),
		strings.HasPrefix(name, "gt_"),
		strings.HasPrefix(name, "ge_"):
		return true
	}

	return false
}

// appUsesCallFunc checks if an App will be generated using CallFunc (function stored in variable).
// M-DX25.10: CallFunc returns interface{} regardless of the function's actual return type.
func (g *Generator) appUsesCallFunc(app *core.App) bool {
	// Check if function is a Var that's NOT a known top-level function
	if v, ok := app.Func.(*core.Var); ok {
		if _, isTopLevel := g.topLevelFuncs[v.Name]; !isTopLevel {
			return true
		}
	}
	return false
}

// appCallsConcreteReturningHelper checks if an App calls a runtime helper that returns a concrete type.
// M-DX25.10: Cons (::) returns []interface{}, not interface{}.
func (g *Generator) appCallsConcreteReturningHelper(app *core.App) bool {
	// Check for VarGlobal
	if varGlobal, ok := app.Func.(*core.VarGlobal); ok {
		if retType := runtimeHelperReturnType(varGlobal.Ref.Name); retType != "" && retType != "interface{}" {
			return true
		}
	}
	// Check for Var (local reference like ::)
	if v, ok := app.Func.(*core.Var); ok {
		if retType := runtimeHelperReturnType(v.Name); retType != "" && retType != "interface{}" {
			return true
		}
	}
	return false
}

// appCallsInterfaceReturningFunc checks if an App calls a user-defined function that returns interface{}.
// M-DX25.10: Used to determine if function call results need type assertions.
func (g *Generator) appCallsInterfaceReturningFunc(app *core.App) bool {
	// Check if function is a VarGlobal referencing a known function
	if varGlobal, ok := app.Func.(*core.VarGlobal); ok {
		name := varGlobal.Ref.Name
		// Check if we know this function's return type
		if retType, found := g.funcReturnTypes[name]; found {
			return retType == "interface{}"
		}
	}
	// Also check for Var (local function reference)
	if v, ok := app.Func.(*core.Var); ok {
		if retType, found := g.funcReturnTypes[v.Name]; found {
			return retType == "interface{}"
		}
	}
	return false
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
	case "::", "Cons":
		// M-DX25.10: Cons returns []interface{}, not interface{}
		return "[]interface{}"
	case "CallFunc":
		// CallFunc is used for calling functions stored in variables, returns interface{}
		return "" // Empty string = returns interface{}

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
