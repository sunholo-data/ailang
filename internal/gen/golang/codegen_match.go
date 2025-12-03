// Package golang provides Go code generation from AILANG Core AST.
package golang

import (
	"fmt"

	"github.com/sunholo/ailang/internal/core"
)

// generateMatch generates a Go switch statement for pattern matching.
func (g *Generator) generateMatch(match *core.Match) error {
	g.write("func() interface{} {\n")
	g.indent++

	// Evaluate scrutinee once
	g.writef("_scrutinee := ")
	if err := g.generateExpr(match.Scrutinee); err != nil {
		return err
	}
	g.writef("\n")
	g.writef("_ = _scrutinee // suppress unused\n")

	// Check if any patterns need if-else (list patterns can't use switch)
	needsIfElse := g.patternsNeedIfElse(match.Arms)

	if needsIfElse {
		// Use if-else chain for complex patterns (lists, etc.)
		return g.generateMatchIfElse(match)
	}

	// Determine if we need a type switch or value switch
	needsTypeSwitch := g.patternsNeedTypeSwitch(match.Arms)

	if needsTypeSwitch {
		// ADT constructor matching - use Kind-based switch
		// First, find the ADT type from constructor patterns
		var adtTypeName string
		for _, arm := range match.Arms {
			if cp, ok := arm.Pattern.(*core.ConstructorPattern); ok {
				if info, exists := g.adtConstructors[cp.Name]; exists {
					adtTypeName = info.TypeName
					break
				}
			}
		}
		if adtTypeName == "" {
			return fmt.Errorf("cannot determine ADT type for match expression")
		}

		// Assert scrutinee to ADT pointer type and switch on Kind
		goADTName := ToGoTypeName(adtTypeName)
		g.writef("_adt := _scrutinee.(*%s)\n", goADTName)
		g.writef("switch _adt.Kind {\n")
		hasDefault := false
		for _, arm := range match.Arms {
			if isWildcardOrVarPattern(arm.Pattern) {
				hasDefault = true
			}
			if err := g.generateMatchArmADT(&arm, adtTypeName); err != nil {
				return err
			}
		}
		if !hasDefault {
			g.writef("default:\n")
			g.indent++
			g.writef("panic(\"non-exhaustive match\")\n")
			g.indent--
		}
	} else {
		// Value switch for literals and wildcards
		g.writef("switch _scrutinee {\n")
		hasDefault := false
		for _, arm := range match.Arms {
			if isWildcardOrVarPattern(arm.Pattern) {
				if hasDefault {
					continue // Skip duplicate default
				}
				hasDefault = true
				g.writef("default:\n")
				g.indent++
				g.writef("return ")
				if err := g.generateExpr(arm.Body); err != nil {
					return err
				}
				g.writef("\n")
				g.indent--
			} else {
				if err := g.generateMatchArmValueSwitch(&arm); err != nil {
					return err
				}
			}
		}
		if !hasDefault {
			g.writef("default:\n")
			g.indent++
			g.writef("panic(\"non-exhaustive match\")\n")
			g.indent--
		}
	}
	g.writef("}\n")

	g.indent--
	g.write("}()")
	return nil
}

// generateMatchIfElse generates if-else chains for complex pattern matching.
func (g *Generator) generateMatchIfElse(match *core.Match) error {
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
				g.writef("_ = %s // suppress unused\n", ToGoVarName(vp.Name))
			}

			g.writef("return ")
			if err := g.generateExpr(arm.Body); err != nil {
				return err
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

// generatePatternCondition generates the condition for a pattern and variable bindings.
func (g *Generator) generatePatternCondition(p core.CorePattern, scrutinee string) (string, []string, error) {
	var bindings []string

	switch pat := p.(type) {
	case *core.LitPattern:
		switch v := pat.Value.(type) {
		case int64:
			return fmt.Sprintf("%s == int64(%d)", scrutinee, v), nil, nil
		case int:
			return fmt.Sprintf("%s == int64(%d)", scrutinee, v), nil, nil
		case float64:
			return fmt.Sprintf("%s == float64(%v)", scrutinee, v), nil, nil
		case bool:
			return fmt.Sprintf("%s == %v", scrutinee, v), nil, nil
		case string:
			return fmt.Sprintf("%s == %q", scrutinee, v), nil, nil
		default:
			return fmt.Sprintf("%s == %v", scrutinee, v), nil, nil
		}

	case *core.ListPattern:
		if len(pat.Elements) == 0 && pat.Tail == nil {
			// Empty list: []
			return fmt.Sprintf("ListLen(%s) == 0", scrutinee), nil, nil
		}
		// List with elements or cons pattern
		if pat.Tail != nil {
			// Cons pattern: head :: tail or [a, b, ...rest]
			minLen := len(pat.Elements)
			cond := fmt.Sprintf("ListLen(%s) >= %d", scrutinee, minLen)

			// Bind head elements
			for i, elem := range pat.Elements {
				if vp, ok := elem.(*core.VarPattern); ok {
					binding := fmt.Sprintf("%s := ListHead(%s)", ToGoVarName(vp.Name), scrutinee)
					bindings = append(bindings, binding)
					bindings = append(bindings, fmt.Sprintf("_ = %s // suppress unused", ToGoVarName(vp.Name)))
					// For next element, need to get from tail
					if i < len(pat.Elements)-1 {
						scrutinee = fmt.Sprintf("ListTail(%s)", scrutinee)
					}
				}
			}

			// Bind tail
			if tailPat, ok := (*pat.Tail).(*core.VarPattern); ok {
				// Calculate tail start position
				if len(pat.Elements) > 0 {
					tailExpr := scrutinee
					for range pat.Elements {
						tailExpr = fmt.Sprintf("ListTail(%s)", tailExpr)
					}
					binding := fmt.Sprintf("%s := %s", ToGoVarName(tailPat.Name), tailExpr)
					bindings = append(bindings, binding)
					bindings = append(bindings, fmt.Sprintf("_ = %s // suppress unused", ToGoVarName(tailPat.Name)))
				} else {
					binding := fmt.Sprintf("%s := ListTail(%s)", ToGoVarName(tailPat.Name), scrutinee)
					bindings = append(bindings, binding)
					bindings = append(bindings, fmt.Sprintf("_ = %s // suppress unused", ToGoVarName(tailPat.Name)))
				}
			}

			return cond, bindings, nil
		}
		// Fixed-length list pattern
		cond := fmt.Sprintf("ListLen(%s) == %d", scrutinee, len(pat.Elements))
		return cond, bindings, nil

	case *core.VarPattern:
		// Var pattern always matches, binding is generated separately
		binding := fmt.Sprintf("%s := %s", ToGoVarName(pat.Name), scrutinee)
		bindings = append(bindings, binding)
		bindings = append(bindings, fmt.Sprintf("_ = %s // suppress unused", ToGoVarName(pat.Name)))
		return "true", bindings, nil

	case *core.WildcardPattern:
		return "true", nil, nil

	default:
		return "true", nil, nil
	}
}

// patternsNeedIfElse returns true if patterns need if-else (list patterns can't use switch).
func (g *Generator) patternsNeedIfElse(arms []core.MatchArm) bool {
	for _, arm := range arms {
		if _, ok := arm.Pattern.(*core.ListPattern); ok {
			return true
		}
	}
	return false
}

// patternsNeedTypeSwitch returns true if patterns include ADT constructors.
func (g *Generator) patternsNeedTypeSwitch(arms []core.MatchArm) bool {
	for _, arm := range arms {
		if _, ok := arm.Pattern.(*core.ConstructorPattern); ok {
			return true
		}
	}
	return false
}

// isWildcardOrVarPattern returns true if pattern is wildcard or variable binding.
func isWildcardOrVarPattern(p core.CorePattern) bool {
	switch p.(type) {
	case *core.WildcardPattern, *core.VarPattern:
		return true
	default:
		return false
	}
}

// generateMatchArmValueSwitch generates a case clause for value-based matching.
func (g *Generator) generateMatchArmValueSwitch(arm *core.MatchArm) error {
	switch p := arm.Pattern.(type) {
	case *core.LitPattern:
		switch v := p.Value.(type) {
		case int64:
			g.writef("case int64(%d):\n", v)
		case int:
			g.writef("case int64(%d):\n", v)
		case float64:
			g.writef("case float64(%v):\n", v)
		case bool:
			g.writef("case %v:\n", v)
		case string:
			g.writef("case %q:\n", v)
		default:
			g.writef("case %v:\n", v)
		}
	case *core.ListPattern:
		if len(p.Elements) == 0 {
			// Empty list pattern - check for empty slice
			g.writef("case []interface{}{}:\n")
		} else {
			// Non-empty list - needs more complex handling
			g.writef("default: // TODO: complex list pattern\n")
		}
	default:
		g.writef("default:\n")
	}
	g.indent++
	g.writef("return ")
	if err := g.generateExpr(arm.Body); err != nil {
		return err
	}
	g.writef("\n")
	g.indent--
	return nil
}

// generateMatchArmADT generates a case clause for ADT pattern matching.
// It uses Kind-based switching and binds constructor fields.
func (g *Generator) generateMatchArmADT(arm *core.MatchArm, adtTypeName string) error {
	switch p := arm.Pattern.(type) {
	case *core.ConstructorPattern:
		// Generate Kind case using proper naming convention
		kindConstName := ToKindConstName(adtTypeName, p.Name)
		g.writef("case %s:\n", kindConstName)
		g.indent++

		// Bind fields from the variant struct using proper naming convention
		if len(p.Args) > 0 {
			variantFieldName := ToVariantStructName(adtTypeName, p.Name)
			for i, arg := range p.Args {
				if vp, ok := arg.(*core.VarPattern); ok {
					goVarName := ToGoVarName(vp.Name)
					g.writef("%s := _adt.%s.Value%d\n", goVarName, variantFieldName, i)
					g.writef("_ = %s // suppress unused\n", goVarName)
				}
				// Wildcards don't need binding
			}
		}

		g.writef("return ")
		if err := g.generateExpr(arm.Body); err != nil {
			return err
		}
		g.writef("\n")
		g.indent--

	case *core.WildcardPattern:
		g.writef("default:\n")
		g.indent++
		g.writef("return ")
		if err := g.generateExpr(arm.Body); err != nil {
			return err
		}
		g.writef("\n")
		g.indent--

	case *core.VarPattern:
		g.writef("default:\n")
		g.indent++
		// Bind the variable to the entire ADT
		goVarName := ToGoVarName(p.Name)
		g.writef("%s := _adt\n", goVarName)
		g.writef("_ = %s // suppress unused\n", goVarName)
		g.writef("return ")
		if err := g.generateExpr(arm.Body); err != nil {
			return err
		}
		g.writef("\n")
		g.indent--

	default:
		return fmt.Errorf("unsupported pattern type in ADT match: %T", arm.Pattern)
	}
	return nil
}
