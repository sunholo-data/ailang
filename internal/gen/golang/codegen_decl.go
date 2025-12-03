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
func (g *Generator) generateFuncFromLambda(name string, lam *core.Lambda, exported bool) error {
	funcName := ToGoFuncName(name, exported)

	// Register this function name mapping for recursive references
	g.topLevelFuncs[name] = funcName

	// Build parameter list
	var params []string
	for _, p := range lam.Params {
		// Type information would come from TypeInfo - for now use interface{}
		params = append(params, fmt.Sprintf("%s interface{}", ToGoVarName(p)))
	}

	g.writef("func %s(%s) interface{} {\n", funcName, strings.Join(params, ", "))
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
