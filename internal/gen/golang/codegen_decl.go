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

	g.writef("func %s(%s) %s {\n", funcName, strings.Join(params, ", "), retType)
	g.indent++
	g.writef("return ")
	if err := g.generateExpr(lam.Body); err != nil {
		return err
	}
	g.writef("\n")
	g.indent--
	g.writef("}\n\n")

	return nil
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
