// Package golang provides Go code generation from AILANG Core AST.
package golang

import (
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/gen/block"
)

// generateFlatBody generates a function body using Block IR for flat output.
// M-CODEGEN-V2: This is the key method that eliminates nested IIFEs.
//
// Instead of generating:
//
//	return func() interface{} {
//	    var x = 1
//	    return func() interface{} {
//	        var y = 2
//	        return x + y
//	    }()
//	}()
//
// We generate:
//
//	var x interface{} = 1
//	var y interface{} = 2
//	return x + y
//
// This produces O(n) lines instead of O(n) nesting depth.
func (g *Generator) generateFlatBody(body core.CoreExpr) error {
	// Special case: LetRec at top level needs forward declarations
	if letrec, ok := body.(*core.LetRec); ok {
		return g.generateFlatBodyLetRec(letrec)
	}

	// Lower the body to extract top-level let chains
	blk := block.Lower(body)

	// Generate flat variable declarations for each binding
	// M-CODEGEN-V2.M4: Only emit suppress unused for _ placeholder bindings (effects).
	// AILANG semantics guarantee all other bindings are used in subsequent code.
	for _, stmt := range blk.Stmts {
		goName := ToGoVarName(stmt.Name)
		g.writef("var %s interface{} = ", goName)
		if err := g.generateExpr(stmt.Value); err != nil {
			return err
		}
		g.writef("\n")
		// Only suppress for _ placeholder bindings (explicit ignore for effects)
		if stmt.Name == "_" || stmt.Name[0] == '_' && len(stmt.Name) > 1 && stmt.Name[1] >= '0' && stmt.Name[1] <= '9' {
			g.writef("_ = %s // suppress unused\n", goName)
		}
	}

	// Generate the final expression as the return value
	g.writef("return ")
	if err := g.generateExpr(blk.FinalExpr); err != nil {
		return err
	}
	g.writef("\n")

	return nil
}

// generateFlatBodyLetRec generates a function body for LetRec using Block IR.
// M-CODEGEN-V2: LetRec requires forward declarations for mutual recursion.
//
// Generated pattern:
//
//	var even func(...interface{}) interface{}
//	var odd func(...interface{}) interface{}
//	even = func(args ...interface{}) interface{} { ... }
//	odd = func(args ...interface{}) interface{} { ... }
//	return body
func (g *Generator) generateFlatBodyLetRec(letrec *core.LetRec) error {
	blk := block.LowerLetRec(letrec)

	// Forward declarations for all bindings
	for _, stmt := range blk.Stmts {
		g.writef("var %s func(...interface{}) interface{}\n", ToGoVarName(stmt.Name))
	}

	// Now assign values (which can reference each other via forward declarations)
	for _, stmt := range blk.Stmts {
		g.writef("%s = func(args ...interface{}) interface{} {\n", ToGoVarName(stmt.Name))
		g.indent++
		g.writef("return ")
		if err := g.generateExpr(stmt.Value); err != nil {
			return err
		}
		g.writef("\n")
		g.indent--
		g.writef("}\n")
	}

	// Generate the final expression
	g.writef("return ")
	if err := g.generateExpr(blk.FinalExpr); err != nil {
		return err
	}
	g.writef("\n")

	return nil
}
