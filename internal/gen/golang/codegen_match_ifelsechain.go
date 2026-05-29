// Package golang provides Go code generation from AILANG Core AST.
package golang

import "github.com/sunholo-data/ailang/internal/core"

// generateMatchIfElse generates if-else chains for complex pattern matching.
// M-PATTERN-GUARDS: When guards are present, uses flat if structure (not else-if)
// so that failed guards can fall through to the next arm.
func (g *Generator) generateMatchIfElse(match *core.Match) error {
	// M-PATTERN-GUARDS: Check if any arm has a guard
	hasAnyGuard := false
	for _, arm := range match.Arms {
		if arm.Guard != nil {
			hasAnyGuard = true
			break
		}
	}

	// M-PATTERN-GUARDS: If guards present, use flat structure for correct fallthrough
	if hasAnyGuard {
		return g.generateMatchIfElseWithGuards(match)
	}

	// Original logic for non-guarded patterns (uses else-if for efficiency)
	first := true
	for _, arm := range match.Arms {
		if isWildcardOrVarPattern(arm.Pattern) {
			// Wildcard/var is the else case
			if first {
				g.writef("// default case\n")
			} else {
				g.writef("} else {\n")
			}
			g.indent++

			// If it's a VarPattern, bind the variable
			if vp, ok := arm.Pattern.(*core.VarPattern); ok {
				g.writef("%s := _scrutinee\n", ToGoVarName(vp.Name))
				g.writeSuppressUnused(ToGoVarName(vp.Name))
			}

			g.writef("return ")
			if err := g.generateExpr(arm.Body); err != nil {
				return err
			}
			// M-DX25.5: Add type assertion if return type is concrete and body produces interface{}
			if g.matchReturnType != "" && g.matchReturnType != "interface{}" && g.exprProducesInterface(arm.Body) {
				g.writef(".(%s)", g.matchReturnType)
			}
			g.writef("\n")
			g.indent--
			if !first {
				g.writef("}\n")
			}
			g.indent--
			g.write("}()")
			return nil
		}

		cond, bindings, err := g.generatePatternCondition(arm.Pattern, "_scrutinee")
		if err != nil {
			return err
		}

		if first {
			g.writef("if %s {\n", cond)
			first = false
		} else {
			g.writef("} else if %s {\n", cond)
		}
		g.indent++

		// Generate bindings
		for _, binding := range bindings {
			g.writef("%s\n", binding)
		}

		g.writef("return ")
		if err := g.generateExpr(arm.Body); err != nil {
			return err
		}
		// M-DX25.5: Add type assertion if return type is concrete and body produces interface{}
		if g.matchReturnType != "" && g.matchReturnType != "interface{}" && g.exprProducesInterface(arm.Body) {
			g.writef(".(%s)", g.matchReturnType)
		}
		g.writef("\n")
		g.indent--
	}

	// No wildcard - add panic
	if first {
		g.writef("panic(\"non-exhaustive match\")\n")
	} else {
		g.writef("} else {\n")
		g.indent++
		g.writef("panic(\"non-exhaustive match\")\n")
		g.indent--
		g.writef("}\n")
	}

	g.indent--
	g.write("}()")
	return nil
}

// generateMatchIfElseWithGuards generates match with guards using flat if structure.
// M-PATTERN-GUARDS: Each arm is a separate if block so failed guards fall through.
func (g *Generator) generateMatchIfElseWithGuards(match *core.Match) error {
	for _, arm := range match.Arms {
		// Handle default case (wildcard/var without guard)
		if isWildcardOrVarPattern(arm.Pattern) && arm.Guard == nil {
			// If it's a VarPattern, bind the variable
			if vp, ok := arm.Pattern.(*core.VarPattern); ok {
				g.writef("%s := _scrutinee\n", ToGoVarName(vp.Name))
				g.writeSuppressUnused(ToGoVarName(vp.Name))
			}

			g.writef("return ")
			if err := g.generateExpr(arm.Body); err != nil {
				return err
			}
			if g.matchReturnType != "" && g.matchReturnType != "interface{}" && g.exprProducesInterface(arm.Body) {
				g.writef(".(%s)", g.matchReturnType)
			}
			g.writef("\n")
			g.indent--
			g.write("}()")
			return nil
		}

		// Get pattern condition and bindings
		cond, bindings, err := g.generatePatternCondition(arm.Pattern, "_scrutinee")
		if err != nil {
			return err
		}

		// M-PATTERN-GUARDS: If arm has a guard, we need nested structure
		if arm.Guard != nil {
			g.writef("if %s {\n", cond)
			g.indent++

			// Generate bindings first (guard may reference them)
			for _, binding := range bindings {
				g.writef("%s\n", binding)
			}

			// Generate guard check - cast to bool since guard expressions return interface{}
			g.writef("if ")
			if err := g.generateExpr(arm.Guard); err != nil {
				return err
			}
			g.writef(".(bool) {\n")
			g.indent++

			// Return body
			g.writef("return ")
			if err := g.generateExpr(arm.Body); err != nil {
				return err
			}
			if g.matchReturnType != "" && g.matchReturnType != "interface{}" && g.exprProducesInterface(arm.Body) {
				g.writef(".(%s)", g.matchReturnType)
			}
			g.writef("\n")

			g.indent--
			g.writef("}\n") // close guard if
			g.indent--
			g.writef("}\n") // close pattern if
		} else {
			// No guard - simple if with return
			g.writef("if %s {\n", cond)
			g.indent++

			for _, binding := range bindings {
				g.writef("%s\n", binding)
			}

			g.writef("return ")
			if err := g.generateExpr(arm.Body); err != nil {
				return err
			}
			if g.matchReturnType != "" && g.matchReturnType != "interface{}" && g.exprProducesInterface(arm.Body) {
				g.writef(".(%s)", g.matchReturnType)
			}
			g.writef("\n")

			g.indent--
			g.writef("}\n")
		}
	}

	// No matching arm - panic
	g.writef("panic(\"non-exhaustive match\")\n")
	g.indent--
	g.write("}()")
	return nil
}
