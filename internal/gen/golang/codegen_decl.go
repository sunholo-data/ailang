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
	// M-CODEGEN-MULTIMOD: Apply module prefix to ALL let bindings when moduleName is set.
	varName := ToGoFuncName(let.Name, exported)
	if g.moduleName != "" {
		varName = g.moduleName + "__" + varName
	}
	// M-CODEGEN-LETBIND-FIX: Do NOT register non-function lets in topLevelFuncs.
	// Only Lambda-valued lets belong there (and they're registered in generateFuncFromLambda).
	// Non-function lets in topLevelFuncs cause generateVar to emit "varName_impl" which
	// doesn't exist, silently breaking ALL function codegen in the module.
	// Instead, register in a separate map for variable references.
	g.topLevelVars[let.Name] = varName

	// M-CODEGEN-MULTIMOD: Dedup — skip if this var was already emitted in this module.
	// The Core program can contain duplicate Let nodes for the same binding when
	// the lowering phase inlines references.
	if g.emittedVars[let.Name] {
		return nil
	}
	g.emittedVars[let.Name] = true

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

// generateFuncFromLambda generates Go functions from a Lambda expression.
// M-DX26: Generates BOTH _impl (interface{} world) and typed wrapper.
// The _impl function uses interface{} for all params and returns.
// The typed wrapper calls _impl and handles type conversions.
// M-ZERO-ARG: Unit-typed parameters are skipped in the Go signature.
func (g *Generator) generateFuncFromLambda(name string, lam *core.Lambda, exported bool) error {
	// M-DX27: Clear typed local vars from previous function
	// This map tracks variables bound from ADT fields within a single function.
	// Must be reset per-function to avoid scope contamination.
	g.typedLocalVars = make(map[string]string)

	// M-PERF6: Clear interface cache — result depends on per-function state
	g.interfaceCache = make(map[core.CoreExpr]bool)

	// M-DX26: Get typed signature for the wrapper
	// M-CODEGEN-TYPED-PARAMS: Pass function name to check for AST-derived type overrides
	paramTypes, returnType := g.getTypedSignature(name, lam)

	// Build typed parameter info for wrapper generation
	// M-ZERO-ARG: Skip unit-typed parameters - they don't appear in Go API
	var typedParamTypes []string
	for i, paramName := range lam.Params {
		var paramType GoType
		if i < len(paramTypes) {
			paramType = paramTypes[i]
		} else {
			paramType = "interface{}"
		}

		// M-ZERO-ARG: Skip unit-typed params from the public API types
		if g.isUnitParam(paramName, paramType) {
			continue
		}
		typedParamTypes = append(typedParamTypes, string(paramType))
	}

	// Determine typed return type
	typedRetType := "interface{}"
	if returnType != "" {
		typedRetType = string(returnType)
	}

	// M-DX26: Store metadata for call site type assertions (uses wrapper types)
	// M-ZERO-ARG: This stores only non-unit params, matching the Go wrapper signature
	g.funcParamTypes[name] = typedParamTypes
	g.funcReturnTypes[name] = typedRetType

	// Register function name mapping (uses wrapper name for external references)
	// M-CODEGEN-MULTIMOD: ALL functions (exported and non-exported) are prefixed when
	// moduleName is set. This prevents collisions when multiple modules export the same
	// function name (e.g., parseDocxComments in both docx_parser and docparse_browser).
	// Previously only non-exported functions were prefixed (M-DX18), but exported
	// functions from different modules also collide in a single Go package.
	funcName := ToGoFuncName(name, exported)
	if g.moduleName != "" {
		funcName = g.moduleName + "__" + funcName
	}
	g.topLevelFuncs[name] = funcName

	// M-DX18-FIX: Also store the _impl name since ToGoVarName != ToGoFuncName for exported funcs
	// _impl uses ToGoVarName (camelCase), wrapper uses ToGoFuncName (PascalCase for exported)
	implName := ToGoVarName(name) + "_impl"
	if g.moduleName != "" {
		implName = g.moduleName + "__" + implName
	}
	g.topLevelImplFuncs[name] = implName

	// M-CROSS-MODULE: Set the declared return type for record literal resolution
	// Extract type name from Go type (strip pointer prefix if present)
	g.currentFuncDeclaredReturn = ""
	if typedRetType != "interface{}" {
		g.currentFuncDeclaredReturn = strings.TrimPrefix(typedRetType, "*")
	}

	// M-DX26.1: Generate _impl function (interface{} everywhere)
	if err := g.generateImplFunc(name, lam); err != nil {
		g.currentFuncDeclaredReturn = ""
		return err
	}
	g.currentFuncDeclaredReturn = ""

	// M-DX26.1: Generate typed wrapper
	return g.generateTypedWrapper(name, lam, typedParamTypes, typedRetType, exported)
}

// generateImplFunc generates the _impl function with interface{} everywhere.
// M-DX26: This is the internal implementation that uses runtime helpers.
// M-CODEGEN-V2: Uses Block IR to generate flat function bodies instead of nested IIFEs.
// M-VERIFY: When verifyContracts is enabled, generates runtime requires checks at entry.
// M-DX18: Non-exported functions are namespaced to prevent collisions across modules.
func (g *Generator) generateImplFunc(name string, lam *core.Lambda) error {
	implName := ToGoVarName(name) + "_impl"
	// M-CODEGEN-MULTIMOD: Namespace ALL _impl functions when moduleName is set
	if g.moduleName != "" {
		implName = g.moduleName + "__" + implName
	}

	// M-VERIFY: Track current function name for contract lookup
	g.currentFuncName = name
	defer func() { g.currentFuncName = "" }()

	// Build parameter list - all interface{}
	// M-BUGFIX: Handle blank identifiers - replace _ with _unused{i}
	var params []string
	paramNameMap := make(map[string]string) // Maps original name to Go name
	for i, p := range lam.Params {
		paramName := ToGoVarName(p)
		if paramName == "_" {
			paramName = fmt.Sprintf("_unused%d", i)
		}
		paramNameMap[p] = paramName
		params = append(params, fmt.Sprintf("%s interface{}", paramName))
	}

	// M-DX26: Set current function params as interface{} for expr generation
	// Use the mapped parameter names (handles _ → _unused{i})
	g.currentFuncParams = make(map[string]string)
	for p, goName := range paramNameMap {
		g.currentFuncParams[p] = "interface{}"
		// Also register the Go name so variable lookups work
		if p != goName {
			g.currentFuncParams[goName] = "interface{}"
		}
	}

	// M-DX26: No expected return type for _impl - everything is interface{}
	oldExpectedReturn := g.expectedReturnType
	g.expectedReturnType = "interface{}"

	g.writef("func %s(%s) interface{} {\n", implName, strings.Join(params, ", "))
	g.indent++

	// M-VERIFY Phase 0.5: Generate requires contract checks at function entry
	if err := g.generateContractRequiresChecks(); err != nil {
		g.expectedReturnType = oldExpectedReturn
		return err
	}

	// M-CODEGEN-V2: Use flat body generation instead of return <expr>
	// This eliminates nested IIFEs by flattening let chains to flat statements
	if err := g.generateFlatBody(lam.Body); err != nil {
		g.expectedReturnType = oldExpectedReturn
		return err
	}

	g.indent--
	g.writef("}\n\n")

	g.expectedReturnType = oldExpectedReturn
	return nil
}

// generateTypedWrapper generates a typed wrapper that calls the _impl function.
// M-DX26: This provides the typed Go API that external code uses.
// M-ZERO-ARG: Unit-typed parameters are skipped in the Go signature but passed to _impl.
// M-VERIFY Phase 0.5: Generates ensures contract checks before return.
// M-DX18: Non-exported functions are namespaced to prevent collisions across modules.
func (g *Generator) generateTypedWrapper(name string, lam *core.Lambda, paramTypes []string, retType string, exported bool) error {
	funcName := ToGoFuncName(name, exported)
	implName := ToGoVarName(name) + "_impl"
	// M-CODEGEN-MULTIMOD: Namespace ALL functions when moduleName is set
	if g.moduleName != "" {
		funcName = g.moduleName + "__" + funcName
		implName = g.moduleName + "__" + implName
	}

	// Build typed parameter list
	// M-BUGFIX: Handle blank identifiers - in Go, _ can be a parameter name
	// (to discard the value) but cannot be used as a value in function calls.
	// Replace _ with _unused0, _unused1, etc.
	// M-ZERO-ARG: Skip unit-typed parameters in the Go signature.
	var params []string
	var callArgs []string
	for i, p := range lam.Params {
		pType := GoType("interface{}")
		if i < len(paramTypes) {
			pType = GoType(paramTypes[i])
		}
		paramName := ToGoVarName(p)
		if paramName == "_" {
			// Generate a usable placeholder name for blank identifiers
			paramName = fmt.Sprintf("_unused%d", i)
		}

		// M-ZERO-ARG: Skip unit-typed params in Go signature, but pass dummy to _impl
		if g.isUnitParam(p, pType) {
			// Unit param: don't add to Go signature, pass struct{}{} to _impl
			callArgs = append(callArgs, "struct{}{}")
			continue
		}

		params = append(params, fmt.Sprintf("%s %s", paramName, pType))
		callArgs = append(callArgs, paramName)
	}

	g.writef("func %s(%s) %s {\n", funcName, strings.Join(params, ", "), retType)
	g.indent++

	// M-VERIFY Phase 0.5: Check if function has ensures contracts
	hasEnsures := g.hasEnsuresContracts(name)

	// Call _impl and convert result
	if hasEnsures {
		// M-VERIFY: Capture result in variable for ensures checks
		implCall := fmt.Sprintf("%s(%s)", implName, strings.Join(callArgs, ", "))
		g.writef("_result := %s\n", g.coerceReturnExpr(implCall, retType))

		// Generate ensures checks
		if err := g.generateContractEnsuresChecks(name, "_result", retType); err != nil {
			return err
		}

		g.writef("return _result\n")
	} else {
		// No ensures - generate direct return with appropriate type coercion
		implCall := fmt.Sprintf("%s(%s)", implName, strings.Join(callArgs, ", "))
		g.writef("return %s\n", g.coerceReturnExpr(implCall, retType))
	}

	g.indent--
	g.writef("}\n\n")

	return nil
}

// coerceReturnExpr generates Go code to convert an interface{} expression to a concrete return type.
// M-CODEGEN-COMPILE-GATE: Handles the _impl → typed wrapper boundary safely.
// _impl functions return interface{} but typed wrappers need concrete Go types.
// Direct .(type) assertion panics when the runtime representation differs from the Go type
// (e.g., _impl returns []interface{} but wrapper expects []string).
func (g *Generator) coerceReturnExpr(expr, retType string) string {
	// interface{} — no conversion needed
	if retType == "interface{}" {
		return expr
	}

	// Slice types — use converter functions
	if strings.HasPrefix(retType, "[]") {
		if conv := g.getSliceConversion(retType); conv != "" {
			return fmt.Sprintf("%s(%s)", conv, expr)
		}
		// Fallback for unknown slice types: direct assertion
		return fmt.Sprintf("%s.(%s)", expr, retType)
	}

	// Pointer types to ADTs/records — direct assertion works (runtime stores as *Type)
	if strings.HasPrefix(retType, "*") {
		return fmt.Sprintf("%s.(%s)", expr, retType)
	}

	// Value types (small records) — may be stored as pointer, use As<Type> converter if available
	if g.hasValueTypeConverter(retType) {
		return fmt.Sprintf("As%s(%s)", retType, expr)
	}

	// Primitive types — direct assertion
	// string, int64, float64, bool are stored as themselves in interface{}
	return fmt.Sprintf("%s.(%s)", expr, retType)
}

// hasValueTypeConverter checks if a value-type converter (AsTypeName) exists for this type.
func (g *Generator) hasValueTypeConverter(typeName string) bool {
	if g.recordTypes == nil {
		return false
	}
	info, ok := g.recordTypes[typeName]
	return ok && info.Category == TypeCategoryValue
}

// hasEnsuresContracts checks if a function has any ensures contracts.
// M-VERIFY Phase 0.5: Used to decide whether to capture result in a variable.
func (g *Generator) hasEnsuresContracts(name string) bool {
	if g.prog == nil || g.prog.Meta == nil {
		return false
	}
	meta, ok := g.prog.Meta[name]
	if !ok {
		return false
	}
	for _, c := range meta.Contracts {
		if c.Kind == core.EnsuresKind {
			return true
		}
	}
	return false
}
