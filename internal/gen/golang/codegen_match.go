// Package golang provides Go code generation from AILANG Core AST.
package golang

import (
	"fmt"
	"strings"

	"github.com/sunholo/ailang/internal/core"
)

// generateMatch generates a Go switch statement for pattern matching.
// M-DX25.5: Uses typed IIFE return based on CoreTypeInfo.
func (g *Generator) generateMatch(match *core.Match) error {
	// M-DX25.5: Look up Match expression's type for IIFE return type
	returnType := "interface{}"
	if g.coreTypeInfo != nil {
		if typ, ok := g.coreTypeInfo[match.NodeID]; ok {
			if goType, err := g.TypeMapper.MapType(typ); err == nil {
				returnType = string(goType)
			}
		}
	}
	g.matchReturnType = returnType // Store for arm generation

	// M-DX25.7: Look up scrutinee's type for typed list operations
	g.matchScrutineeType = "interface{}"
	if g.coreTypeInfo != nil {
		scrutineeNodeID := g.getExprNodeID(match.Scrutinee)
		if typ, ok := g.coreTypeInfo[scrutineeNodeID]; ok {
			if goType, err := g.TypeMapper.MapType(typ); err == nil {
				g.matchScrutineeType = string(goType)
			}
		}
	}

	g.writef("func() %s {\n", returnType)
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
		// M-DX25.6: Only type-assert if scrutinee produces interface{}
		if g.exprProducesInterface(match.Scrutinee) {
			g.writef("_adt := _scrutinee.(*%s)\n", goADTName)
		} else {
			// Scrutinee is already typed - no assertion needed
			g.writef("_adt := _scrutinee\n")
		}
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
				// M-DX25.5: Add type assertion if return type is concrete and body produces interface{}
				if g.matchReturnType != "" && g.matchReturnType != "interface{}" && g.exprProducesInterface(arm.Body) {
					g.writef(".(%s)", g.matchReturnType)
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
		// M-DX25.7: Use typed inline operations when scrutinee type is a slice
		isTypedSlice := strings.HasPrefix(g.matchScrutineeType, "[]") && g.matchScrutineeType != "[]interface{}"

		if len(pat.Elements) == 0 && pat.Tail == nil {
			// Empty list: []
			if isTypedSlice {
				return fmt.Sprintf("len(%s) == 0", scrutinee), nil, nil
			}
			return fmt.Sprintf("ListLen(%s) == 0", scrutinee), nil, nil
		}
		// List with elements or cons pattern
		if pat.Tail != nil {
			// Cons pattern: head :: tail or [a, b, ...rest]
			minLen := len(pat.Elements)
			var cond string
			if isTypedSlice {
				cond = fmt.Sprintf("len(%s) >= %d", scrutinee, minLen)
			} else {
				cond = fmt.Sprintf("ListLen(%s) >= %d", scrutinee, minLen)
			}

			// Bind head elements
			for i, elem := range pat.Elements {
				if vp, ok := elem.(*core.VarPattern); ok && vp.Name != "_" {
					// Skip binding for wildcard patterns (name == "_")
					var binding string
					if isTypedSlice {
						// Use indexed access for typed slices
						binding = fmt.Sprintf("%s := %s[%d]", ToGoVarName(vp.Name), scrutinee, i)
					} else {
						binding = fmt.Sprintf("%s := ListHead(%s)", ToGoVarName(vp.Name), scrutinee)
					}
					bindings = append(bindings, binding)
					bindings = append(bindings, fmt.Sprintf("_ = %s // suppress unused", ToGoVarName(vp.Name)))
				}
				// For untyped lists, need to get from tail (always advance, even for wildcards)
				if !isTypedSlice && i < len(pat.Elements)-1 {
					scrutinee = fmt.Sprintf("ListTail(%s)", scrutinee)
				}
			}

			// Bind tail (skip if wildcard)
			if tailPat, ok := (*pat.Tail).(*core.VarPattern); ok && tailPat.Name != "_" {
				var binding string
				if isTypedSlice {
					// Use slice expression for typed slices
					binding = fmt.Sprintf("%s := %s[%d:]", ToGoVarName(tailPat.Name), scrutinee, len(pat.Elements))
				} else {
					// Calculate tail start position
					if len(pat.Elements) > 0 {
						tailExpr := scrutinee
						for range pat.Elements {
							tailExpr = fmt.Sprintf("ListTail(%s)", tailExpr)
						}
						binding = fmt.Sprintf("%s := %s", ToGoVarName(tailPat.Name), tailExpr)
					} else {
						binding = fmt.Sprintf("%s := ListTail(%s)", ToGoVarName(tailPat.Name), scrutinee)
					}
				}
				bindings = append(bindings, binding)
				bindings = append(bindings, fmt.Sprintf("_ = %s // suppress unused", ToGoVarName(tailPat.Name)))
			}

			return cond, bindings, nil
		}
		// Fixed-length list pattern
		var cond string
		if isTypedSlice {
			cond = fmt.Sprintf("len(%s) == %d", scrutinee, len(pat.Elements))
		} else {
			cond = fmt.Sprintf("ListLen(%s) == %d", scrutinee, len(pat.Elements))
		}
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
	// M-DX25.5: Add type assertion if return type is concrete and body produces interface{}
	if g.matchReturnType != "" && g.matchReturnType != "interface{}" && g.exprProducesInterface(arm.Body) {
		g.writef(".(%s)", g.matchReturnType)
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
		// Field names use ToPascalCase(ctor.Name), not ToVariantStructName
		// e.g., for MovementPattern.PatternRandomWalk, field is "PatternRandomWalk" not "MovementPatternPatternRandomWalk"
		if len(p.Args) > 0 {
			variantFieldName := ToPascalCase(p.Name)
			// Look up field names from registered constructor info
			var ctorFieldNames []string
			if info, exists := g.adtConstructors[p.Name]; exists && len(info.FieldNames) > 0 {
				ctorFieldNames = info.FieldNames
			}
			for i, arg := range p.Args {
				if vp, ok := arg.(*core.VarPattern); ok && vp.Name != "_" {
					// Skip binding for wildcard patterns (name == "_")
					goVarName := ToGoVarName(vp.Name)
					// Use named field if available, otherwise fallback to Value0, Value1, ...
					var fieldAccess string
					if i < len(ctorFieldNames) && ctorFieldNames[i] != "" {
						fieldAccess = ToPascalCase(ctorFieldNames[i])
					} else {
						fieldAccess = fmt.Sprintf("Value%d", i)
					}
					g.writef("%s := _adt.%s.%s\n", goVarName, variantFieldName, fieldAccess)
					g.writef("_ = %s // suppress unused\n", goVarName)
				}
				// Wildcards (_) and non-VarPattern args don't need binding
			}
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

	case *core.WildcardPattern:
		g.writef("default:\n")
		g.indent++
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
		// M-DX25.5: Add type assertion if return type is concrete and body produces interface{}
		if g.matchReturnType != "" && g.matchReturnType != "interface{}" && g.exprProducesInterface(arm.Body) {
			g.writef(".(%s)", g.matchReturnType)
		}
		g.writef("\n")
		g.indent--

	default:
		return fmt.Errorf("unsupported pattern type in ADT match: %T", arm.Pattern)
	}
	return nil
}
