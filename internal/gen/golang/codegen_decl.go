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
	// M-DX18: Non-exported functions are namespaced to prevent collisions across modules
	funcName := ToGoFuncName(name, exported)
	if !exported && g.moduleName != "" {
		// Prefix non-exported functions with module name to avoid collisions
		funcName = g.moduleName + "__" + funcName
	}
	g.topLevelFuncs[name] = funcName

	// M-DX18-FIX: Also store the _impl name since ToGoVarName != ToGoFuncName for exported funcs
	// _impl uses ToGoVarName (camelCase), wrapper uses ToGoFuncName (PascalCase for exported)
	implName := ToGoVarName(name) + "_impl"
	if !exported && g.moduleName != "" {
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
	// M-DX18: Namespace non-exported _impl functions
	if !g.isExported(name) && g.moduleName != "" {
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
	// M-DX18: Namespace non-exported wrapper and _impl functions
	if !exported && g.moduleName != "" {
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
		resultExpr := fmt.Sprintf("%s(%s)", implName, strings.Join(callArgs, ", "))
		if retType == "interface{}" {
			g.writef("_result := %s\n", resultExpr)
		} else if strings.HasPrefix(retType, "[]") {
			sliceConv := g.getSliceConversion(retType)
			if sliceConv != "" {
				g.writef("_result := %s(%s)\n", sliceConv, resultExpr)
			} else {
				g.writef("_result := %s.(%s)\n", resultExpr, retType)
			}
		} else {
			g.writef("_result := %s.(%s)\n", resultExpr, retType)
		}

		// Generate ensures checks
		if err := g.generateContractEnsuresChecks(name, "_result", retType); err != nil {
			return err
		}

		g.writef("return _result\n")
	} else {
		// No ensures - generate direct return
		if retType == "interface{}" {
			g.writef("return %s(%s)\n", implName, strings.Join(callArgs, ", "))
		} else if strings.HasPrefix(retType, "[]") {
			sliceConv := g.getSliceConversion(retType)
			if sliceConv != "" {
				g.writef("return %s(%s(%s))\n", sliceConv, implName, strings.Join(callArgs, ", "))
			} else {
				g.writef("return %s(%s).(%s)\n", implName, strings.Join(callArgs, ", "), retType)
			}
		} else {
			g.writef("return %s(%s).(%s)\n", implName, strings.Join(callArgs, ", "), retType)
		}
	}

	g.indent--
	g.writef("}\n\n")

	return nil
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
