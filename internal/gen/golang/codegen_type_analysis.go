// Package golang provides Go code generation from AILANG Core AST.
package golang

import (
	"strings"

	"github.com/sunholo-data/ailang/internal/core"
)

// exprProducesInterface checks if an expression produces interface{} type.
// M-DX24: Used to determine if type assertion is needed.
// M-DX25.8: Uses CoreTypeInfo for accurate type checking.
// M-DX25.10: Checks for runtime helper calls FIRST - these always produce interface{}
// regardless of what CoreTypeInfo says about the AILANG type.
// M-DX26: In _impl functions, everything produces interface{} (except literals).
func (g *Generator) exprProducesInterface(expr core.CoreExpr) bool {
	// M-PERF6: Check memoization cache first
	if g.interfaceCache != nil {
		if cached, ok := g.interfaceCache[expr]; ok {
			return cached
		}
	}

	result := g.exprProducesInterfaceUncached(expr)

	// M-PERF6: Store result in cache
	if g.interfaceCache != nil {
		g.interfaceCache[expr] = result
	}
	return result
}

// exprProducesInterfaceUncached is the actual implementation without caching.
func (g *Generator) exprProducesInterfaceUncached(expr core.CoreExpr) bool {
	// M-DX26: In _impl functions, almost everything is interface{}
	// Exceptions: literals and ADT constructor calls (which return typed values)
	if g.expectedReturnType == "interface{}" {
		if _, isLit := expr.(*core.Lit); isLit {
			return false // Literals produce concrete types
		}
		// M-DX27: Local variables bound from typed ADT fields have concrete types
		// e.g., s := _adt.ContentStarfield.Scroll is bool, not interface{}
		if v, isVar := expr.(*core.Var); isVar {
			goVarName := ToGoVarName(v.Name)
			if _, isTyped := g.typedLocalVars[goVarName]; isTyped {
				return false // Variable has concrete type from ADT field extraction
			}
		}
		// M-CODEGEN-ADT-TYPE-ASSERT: ADT constructor calls return typed values (*ADT),
		// not interface{}, even in _impl functions
		if app, isApp := expr.(*core.App); isApp {
			if g.isADTConstructorCall(app) {
				return false // ADT constructors return *ADT, not interface{}
			}
		}
		// M-CODEGEN-ADT-TYPE-ASSERT: Nullary ADT constructors are VarGlobal, not App.
		// Example: `G` in `type SpectralClass = O | B | A | F | G | K | M` becomes NewSpectralClassG()
		if vg, isVG := expr.(*core.VarGlobal); isVG {
			if g.isNullaryADTConstructor(vg) {
				return false // Nullary ADT constructors return *ADT, not interface{}
			}
		}
		return true // Everything else is interface{}
	}

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

// isADTConstructorCall checks if an App is calling an ADT constructor.
// M-CODEGEN-ADT-TYPE-ASSERT: ADT constructor calls return typed values (*ADT), not interface{}.
func (g *Generator) isADTConstructorCall(app *core.App) bool {
	// Check for $adt.make_TypeName_CtorName pattern
	if v, ok := app.Func.(*core.VarGlobal); ok {
		if v.Ref.Module == "$adt" && strings.HasPrefix(v.Ref.Name, "make_") {
			return true
		}
		// Also check direct constructor name
		// M-DX22: Use LookupADTConstructor for backwards-compatible lookup
		if _, ok := g.LookupADTConstructor("", v.Ref.Name); ok {
			return true
		}
	}
	return false
}

// isNullaryADTConstructor checks if a VarGlobal is a nullary ADT constructor.
// M-CODEGEN-ADT-TYPE-ASSERT: Nullary constructors (with no fields) are VarGlobal, not App.
// Example: `G` in `type SpectralClass = O | B | A | F | G | K | M` is a nullary constructor.
// M-CODEGEN-ADT-DOUBLE-PAREN: Fixed to actually check FieldCount, not just pattern.
func (g *Generator) isNullaryADTConstructor(vg *core.VarGlobal) bool {
	// Check for $adt.make_TypeName_CtorName pattern
	if vg.Ref.Module == "$adt" && strings.HasPrefix(vg.Ref.Name, "make_") {
		// M-CODEGEN-ADT-DOUBLE-PAREN: Parse the pattern and look up actual field count.
		// Don't assume ALL $adt.make_* are nullary!
		parts := strings.SplitN(vg.Ref.Name[5:], "_", 2) // Skip "make_"
		if len(parts) == 2 {
			typeName := parts[0]
			ctorName := parts[1]
			// M-DX22: Use qualified lookup with typeName for disambiguation
			if info, ok := g.LookupADTConstructor(typeName, ctorName); ok {
				return len(info.FieldTypes) == 0
			}
		}
		// If not in map, we can't determine - default to false (safer)
		return false
	}
	// Check if it's in the adtConstructors map (nullary constructors have 0 field types)
	// M-DX22: Use LookupADTConstructor for backwards-compatible lookup
	if info, ok := g.LookupADTConstructor("", vg.Ref.Name); ok {
		return len(info.FieldTypes) == 0
	}
	return false
}

// appProducesInterface checks if a function application produces interface{}.
// M-DX24: ADT constructors and typed runtime helpers return concrete types.
func (g *Generator) appProducesInterface(app *core.App) bool {
	// Check if this is a known ADT constructor or runtime helper
	if varGlobal, ok := app.Func.(*core.VarGlobal); ok {
		// M-CODEGEN-ADT-TYPE-ASSERT: Check for $adt.make_TypeName_CtorName pattern
		if varGlobal.Ref.Module == "$adt" && strings.HasPrefix(varGlobal.Ref.Name, "make_") {
			// ADT constructors return *ADT, not interface{}
			return false
		}
		name := varGlobal.Ref.Name
		// M-DX22: Use LookupADTConstructor for backwards-compatible lookup
		if _, ok := g.LookupADTConstructor("", name); ok {
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

// getTypedSignature extracts typed parameter and return types for a function.
// M-DX23: Returns nil/empty if type info is unavailable or not a function type.
// M-CODEGEN-TYPED-PARAMS: First checks funcTypeOverrides for explicit AST-derived types,
// which prevents cross-module type contamination when multiple records share field names.
func (g *Generator) getTypedSignature(name string, lam *core.Lambda) ([]GoType, GoType) {
	// M-CODEGEN-TYPED-PARAMS: Check for explicit override from AST first
	// This ensures declared types (e.g., ArrivalState) are used instead of
	// inferred structural types (TRecord{...}) which can be ambiguous
	if override, ok := g.funcTypeOverrides[name]; ok {
		return override.ParamTypes, override.ReturnType
	}

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

// isUnitType checks if a Go type represents the unit type.
// M-ZERO-ARG: Used to skip unit-typed parameters in function signatures.
// Unit types in Go are: struct{} (from AILANG ())
func isUnitType(t GoType) bool {
	return t == "struct{}" || t == "()" || t == "unit"
}

// isUnitParam checks if a Lambda parameter should be skipped in Go codegen.
// M-ZERO-ARG: Parameters named "_" with unit type are implicit and shouldn't appear in Go.
// This handles the parser's implicit unit param for zero-arg functions: func f() -> T
func (g *Generator) isUnitParam(paramName string, paramType GoType) bool {
	// Check if it's a unit type
	if isUnitType(paramType) {
		return true
	}
	// Also skip if the param name is "_" and type is interface{} (fallback case)
	// This handles when type info isn't available but we know it's implicit
	if paramName == "_" && paramType == "interface{}" {
		return true
	}
	return false
}
