// Package golang provides Go code generation from AILANG Core AST.
package golang

import (
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/gen/block"
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

	// M-CODEGEN-LIST: Special case: Let-Bool-Match chain should be flattened
	// Pattern: Let $cmp = <comparison> in Match $cmp { true => A, false => ... }
	// This generates a flat if-else chain instead of nested switch/IIFE.
	if let, ok := body.(*core.Let); ok && isLetBoolMatchChain(let) {
		return g.generateFlatBodyBoolMatchChain(let)
	}

	// Lower the body to extract top-level let chains
	blk := block.Lower(body)

	// Generate flat variable declarations for each binding
	// M-CODEGEN-STDLIB-BUILTINS: Always suppress unused for all flat bindings.
	// Effectful sequencing (e.g., let _ = println("...") in ...) produces bindings
	// whose return values are never referenced. Go requires _ = var for these.
	for _, stmt := range blk.Stmts {
		goName := ToGoVarName(stmt.Name)
		g.writef("var %s interface{} = ", goName)
		if err := g.generateExpr(stmt.Value); err != nil {
			return err
		}
		g.writef("\n")
		g.writeSuppressUnused(goName)
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

// generateFlatBodyBoolMatchChain generates a function body for Let-Bool-Match chains.
// M-CODEGEN-LIST: This eliminates nested IIFEs for spectralFromRoll-style patterns.
//
// Instead of generating:
//
//	var tmp29 interface{} = LtFloat(roll, 0.76)
//	return func() interface{} {
//	    switch tmp29 {
//	    case true: return M
//	    case false: return func() interface{} { ... nested ... }()
//	    }
//	}()
//
// We generate:
//
//	if LtFloat(roll, 0.76).(bool) {
//	    return M
//	} else if LtFloat(roll, 0.88).(bool) {
//	    return K
//	} else {
//	    return O
//	}
func (g *Generator) generateFlatBodyBoolMatchChain(let *core.Let) error {
	entries, elseBody := collectLetBoolMatchChain(let)

	for i, entry := range entries {
		if i == 0 {
			g.writef("if ")
		} else {
			g.writef("} else if ")
		}
		if err := g.generateExpr(entry.Condition); err != nil {
			return err
		}
		if g.exprProducesInterface(entry.Condition) {
			g.write(".(bool)")
		}
		g.write(" {\n")
		g.indent++
		g.writef("return ")
		if err := g.generateExpr(entry.TrueBody); err != nil {
			return err
		}
		g.writef("\n")
		g.indent--
	}

	// Final else
	g.writef("} else {\n")
	g.indent++
	g.writef("return ")
	if err := g.generateExpr(elseBody); err != nil {
		return err
	}
	g.writef("\n")
	g.indent--
	g.writef("}\n")

	return nil
}
