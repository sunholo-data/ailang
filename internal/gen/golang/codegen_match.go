// Package golang provides Go code generation from AILANG Core AST.
package golang

import (
	"fmt"
	"strings"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// extractADTTypeName extracts the ADT type name from an AILANG type.
// M-DX22: Used to determine the correct ADT type when constructor names collide.
// Returns empty string if the type is not an ADT or type name cannot be determined.
func (g *Generator) extractADTTypeName(typ types.Type) string {
	switch t := typ.(type) {
	case *types.TCon:
		// Simple ADT type like SpectralType
		return t.Name
	case *types.TApp:
		// Generic ADT type like Option[T]
		if con, ok := t.Constructor.(*types.TCon); ok {
			return con.Name
		}
	}
	return ""
}

// generateMatch generates a Go switch statement for pattern matching.
// M-DX25.5: Uses typed IIFE return based on CoreTypeInfo.
// M-DX26: In _impl functions (interface{} world), uses interface{} everywhere.
func (g *Generator) generateMatch(match *core.Match) error {
	// M-DX26: In _impl functions, everything is interface{}
	inImplFunc := g.expectedReturnType == "interface{}"

	// M-DX25.5: Look up Match expression's type for IIFE return type
	returnType := "interface{}"
	if !inImplFunc && g.coreTypeInfo != nil {
		if typ, ok := g.coreTypeInfo[match.NodeID]; ok {
			if goType, err := g.TypeMapper.MapType(typ); err == nil {
				returnType = string(goType)
			}
		}
	}
	g.matchReturnType = returnType // Store for arm generation

	// M-CODEGEN-LIST: Check for bool match chain pattern and flatten it
	// Pattern: match <bool> { true => A, false => match <bool> { ... } }
	if chain := extractBoolMatchChain(match); len(chain) > 0 {
		return g.generateFlatBoolMatchChain(chain, returnType)
	}

	// M-DX25.7: Look up scrutinee's type for typed list operations
	// M-DX26: In _impl functions, scrutinee is always interface{}
	g.matchScrutineeType = "interface{}"
	g.matchScrutineeAILANGType = nil // M-DX29: Reset AILANG type

	// M-DX22: ALWAYS look up AILANG type for ADT disambiguation (even in impl functions)
	// This is needed to resolve constructor name collisions like SpectralType.O vs SpectralClass.O
	if g.coreTypeInfo != nil {
		scrutineeNodeID := g.getExprNodeID(match.Scrutinee)
		if typ, ok := g.coreTypeInfo[scrutineeNodeID]; ok {
			g.matchScrutineeAILANGType = typ // M-DX22/M-DX29: Store for ADT type extraction
			// M-DX26: Only map to Go type if NOT in impl function (impl uses interface{})
			if !inImplFunc {
				if goType, err := g.TypeMapper.MapType(typ); err == nil {
					g.matchScrutineeType = string(goType)
				}
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
		// M-DX22: First try to get ADT type from scrutinee's type info (avoids constructor name collisions)
		var adtTypeName string
		if g.matchScrutineeAILANGType != nil {
			adtTypeName = g.extractADTTypeName(g.matchScrutineeAILANGType)
		}
		// Fallback: find the ADT type from constructor patterns (legacy behavior)
		if adtTypeName == "" {
			for _, arm := range match.Arms {
				if cp, ok := arm.Pattern.(*core.ConstructorPattern); ok {
					// M-DX22: Use LookupADTConstructor for backwards-compatible lookup
					if info, ok := g.LookupADTConstructor("", cp.Name); ok {
						adtTypeName = info.TypeName
						break
					}
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

// generateMatchIfElseWithGuards generates match with guards using flat if structure.
// M-PATTERN-GUARDS: Each arm is a separate if block so failed guards fall through.
func (g *Generator) generateMatchIfElseWithGuards(match *core.Match) error {
	for _, arm := range match.Arms {
		// Handle default case (wildcard/var without guard)
		if isWildcardOrVarPattern(arm.Pattern) && arm.Guard == nil {
			// If it's a VarPattern, bind the variable
			if vp, ok := arm.Pattern.(*core.VarPattern); ok {
				g.writef("%s := _scrutinee\n", ToGoVarName(vp.Name))
				g.writef("_ = %s // suppress unused\n", ToGoVarName(vp.Name))
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
			// M-DX17: Handle nested patterns (TuplePattern, etc.) not just VarPattern
			for i, elem := range pat.Elements {
				// Get the current element expression
				var elemExpr string
				if isTypedSlice {
					elemExpr = fmt.Sprintf("%s[%d]", scrutinee, i)
				} else {
					elemExpr = fmt.Sprintf("ListHead(%s)", scrutinee)
				}

				switch elemPat := elem.(type) {
				case *core.VarPattern:
					if elemPat.Name != "_" {
						binding := fmt.Sprintf("%s := %s", ToGoVarName(elemPat.Name), elemExpr)
						bindings = append(bindings, binding)
						bindings = append(bindings, fmt.Sprintf("_ = %s // suppress unused", ToGoVarName(elemPat.Name)))
					}
				case *core.WildcardPattern:
					// No binding needed for wildcards
				case *core.TuplePattern:
					// M-DX17: Nested tuple pattern - extract element and recurse
					tempVar := fmt.Sprintf("_listElem%d", i)
					binding := fmt.Sprintf("%s := %s", tempVar, elemExpr)
					bindings = append(bindings, binding)
					// Recursively get bindings for nested tuple
					_, nestedBindings, err := g.generatePatternCondition(elemPat, tempVar)
					if err != nil {
						return "", nil, err
					}
					bindings = append(bindings, nestedBindings...)
				default:
					// Other pattern types - generate temp and recurse
					tempVar := fmt.Sprintf("_listElem%d", i)
					binding := fmt.Sprintf("%s := %s", tempVar, elemExpr)
					bindings = append(bindings, binding)
					bindings = append(bindings, fmt.Sprintf("_ = %s // suppress unused", tempVar))
					_, nestedBindings, err := g.generatePatternCondition(elem, tempVar)
					if err != nil {
						return "", nil, err
					}
					bindings = append(bindings, nestedBindings...)
				}

				// For untyped lists, advance to next element
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

	case *core.TuplePattern:
		// M-CODEGEN-TUPLE: Tuple patterns extract elements from []interface{}
		// In _impl functions, tuples are represented as []interface{} slices
		cond := fmt.Sprintf("len(%s.([]interface{})) == %d", scrutinee, len(pat.Elements))

		// Generate bindings for each element
		for i, elem := range pat.Elements {
			switch elemPat := elem.(type) {
			case *core.VarPattern:
				if elemPat.Name != "_" {
					binding := fmt.Sprintf("%s := %s.([]interface{})[%d]",
						ToGoVarName(elemPat.Name), scrutinee, i)
					bindings = append(bindings, binding)
					bindings = append(bindings, fmt.Sprintf("_ = %s // suppress unused", ToGoVarName(elemPat.Name)))
				}
			case *core.WildcardPattern:
				// No binding needed for wildcards
			case *core.TuplePattern:
				// Nested tuple - generate temp var and recurse
				tempVar := fmt.Sprintf("_tuple%d", i)
				binding := fmt.Sprintf("%s := %s.([]interface{})[%d]", tempVar, scrutinee, i)
				bindings = append(bindings, binding)
				bindings = append(bindings, fmt.Sprintf("_ = %s // suppress unused", tempVar))
				// Recursively get bindings for nested tuple
				_, nestedBindings, err := g.generatePatternCondition(elemPat, tempVar)
				if err != nil {
					return "", nil, err
				}
				bindings = append(bindings, nestedBindings...)
			default:
				// Other pattern types in tuple - generate temp and recurse
				tempVar := fmt.Sprintf("_elem%d", i)
				binding := fmt.Sprintf("%s := %s.([]interface{})[%d]", tempVar, scrutinee, i)
				bindings = append(bindings, binding)
				_, nestedBindings, err := g.generatePatternCondition(elem, tempVar)
				if err != nil {
					return "", nil, err
				}
				bindings = append(bindings, nestedBindings...)
			}
		}
		return cond, bindings, nil

	case *core.ConstructorPattern:
		// M-CODEGEN-STDLIB-BUILTINS: Handle ADT constructor patterns in if-else chains.
		// This is used when multiple arms match the same constructor with different
		// nested literal patterns (e.g., Some("heading"), Some("text")).
		// Generate: _adt := scrutinee.(*ADT); _adt.Kind == KindSome && _adt.Some.Field0 == "heading"

		// Find ADT type name from constructor
		adtTypeName := ""
		if info, ok := g.LookupADTConstructor("", pat.Name); ok {
			adtTypeName = info.TypeName
		}
		if adtTypeName == "" {
			adtTypeName = "Option" // fallback for common case
		}

		kindConstName := ToKindConstName(adtTypeName, pat.Name)
		variantFieldName := ToPascalCase(pat.Name)

		// Get constructor field info
		var ctorFieldNames []string
		if info, ok := g.LookupADTConstructor("", pat.Name); ok && len(info.FieldNames) > 0 {
			ctorFieldNames = info.FieldNames
		}

		// Condition: check Kind
		adtVar := fmt.Sprintf("_adt_%d", g.varCounter)
		g.varCounter++
		cond := fmt.Sprintf("func() bool { %s := %s.(*%s); return %s.Kind == %s", adtVar, scrutinee, adtTypeName, adtVar, kindConstName)

		// Check nested argument patterns (e.g., the "heading" in Some("heading"))
		for i, arg := range pat.Args {
			fieldAccess := ""
			if i < len(ctorFieldNames) && ctorFieldNames[i] != "" {
				fieldAccess = ToPascalCase(ctorFieldNames[i])
			} else {
				fieldAccess = fmt.Sprintf("Value%d", i)
			}
			fieldExpr := fmt.Sprintf("%s.%s.%s", adtVar, variantFieldName, fieldAccess)

			switch ap := arg.(type) {
			case *core.LitPattern:
				switch v := ap.Value.(type) {
				case string:
					cond += fmt.Sprintf(" && %s == %q", fieldExpr, v)
				case int64:
					cond += fmt.Sprintf(" && %s == int64(%d)", fieldExpr, v)
				case bool:
					cond += fmt.Sprintf(" && %s == %v", fieldExpr, v)
				default:
					cond += fmt.Sprintf(" && %s == %v", fieldExpr, v)
				}
			case *core.VarPattern:
				if ap.Name != "_" {
					bindings = append(bindings, fmt.Sprintf("%s := %s.(*%s).%s.%s", ToGoVarName(ap.Name), scrutinee, adtTypeName, variantFieldName, fieldAccess))
					bindings = append(bindings, fmt.Sprintf("_ = %s // suppress unused", ToGoVarName(ap.Name)))
				}
			case *core.WildcardPattern:
				// No binding or condition needed
			}
		}

		cond += " }()"
		return cond, bindings, nil

	default:
		return "true", nil, nil
	}
}

// patternsNeedIfElse returns true if patterns need if-else (list/tuple patterns can't use switch).
// M-CODEGEN-TUPLE: Added TuplePattern check.
// M-PATTERN-GUARDS: Also returns true if any arm has a guard (switches can't handle guards).
func (g *Generator) patternsNeedIfElse(arms []core.MatchArm) bool {
	// Track constructor names to detect duplicates
	ctorNames := make(map[string]bool)
	for _, arm := range arms {
		// M-PATTERN-GUARDS: Guards require if-else chains
		if arm.Guard != nil {
			return true
		}
		switch p := arm.Pattern.(type) {
		case *core.ListPattern, *core.TuplePattern:
			return true
		case *core.ConstructorPattern:
			// M-CODEGEN-STDLIB-BUILTINS: If multiple arms match the same constructor
			// (e.g., Some("heading"), Some("text")), we need if-else because Go switch
			// can't have duplicate case values. This happens with nested literal patterns.
			if ctorNames[p.Name] {
				return true
			}
			ctorNames[p.Name] = true
			// Also check if any constructor arg is a literal pattern (nested matching)
			for _, arg := range p.Args {
				if _, isLit := arg.(*core.LitPattern); isLit {
					return true
				}
			}
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
		// M-CODEGEN-MULTIMOD: ListPatterns should always go through the if-else path
		// (patternsNeedIfElse returns true for ListPattern). If we reach here, it's
		// a routing bug. Previously emitted `case []interface{}{}:` which is invalid Go
		// (slices are not comparable, can't be switch cases).
		// Fallback: use default case to avoid generating invalid Go syntax.
		g.writef("default: // ListPattern in switch (should use if-else path)\n")
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
			// Look up field names and types from registered constructor info
			var ctorFieldNames []string
			var ctorFieldTypes []string
			// M-DX22: Use LookupADTConstructor for backwards-compatible lookup
			if info, ok := g.LookupADTConstructor("", p.Name); ok {
				if len(info.FieldNames) > 0 {
					ctorFieldNames = info.FieldNames
				}
				if len(info.FieldTypes) > 0 {
					ctorFieldTypes = info.FieldTypes
				}
			}

			// M-DX29: Extract type arguments from generic scrutinee type (e.g., Option[InteractableID])
			// This allows us to add type assertions when extracting ADT values from generic containers
			var typeArgGoTypes []string
			if tapp, ok := g.matchScrutineeAILANGType.(*types.TApp); ok && len(tapp.Args) > 0 {
				typeArgGoTypes = make([]string, len(tapp.Args))
				for i, arg := range tapp.Args {
					if goType, err := g.TypeMapper.MapType(arg); err == nil {
						typeArgGoTypes[i] = string(goType)
					}
				}
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

					// M-DX29: Check if we need to add type assertion for ADT from generic container
					// If the registered field type is interface{} but we have a type argument that's an ADT,
					// add the type assertion to get the concrete type
					fieldGoType := ""
					if i < len(ctorFieldTypes) && ctorFieldTypes[i] != "" {
						fieldGoType = ctorFieldTypes[i]
					}
					needsTypeAssertion := false
					typeAssertionType := ""
					if (fieldGoType == "" || fieldGoType == "interface{}") && i < len(typeArgGoTypes) && typeArgGoTypes[i] != "" {
						// Type argument exists and field is interface{} - check if it's an ADT pointer type
						typeArgType := typeArgGoTypes[i]
						if strings.HasPrefix(typeArgType, "*") && typeArgType != "*struct{}" {
							// It's a pointer type (likely an ADT) - add type assertion
							needsTypeAssertion = true
							typeAssertionType = typeArgType
							fieldGoType = typeArgType
						}
					}

					if needsTypeAssertion {
						g.writef("%s := _adt.%s.%s.(%s)\n", goVarName, variantFieldName, fieldAccess, typeAssertionType)
					} else {
						g.writef("%s := _adt.%s.%s\n", goVarName, variantFieldName, fieldAccess)
					}
					g.writef("_ = %s // suppress unused\n", goVarName)

					// M-DX27: Record the concrete Go type for this local variable
					// This allows exprProducesInterface to know that s is bool, not interface{}
					if fieldGoType != "" && fieldGoType != "interface{}" {
						g.typedLocalVars[goVarName] = fieldGoType
					}
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
		// Bind the variable to the entire ADT (skip binding for wildcard "_")
		if p.Name != "_" {
			goVarName := ToGoVarName(p.Name)
			g.writef("%s := _adt\n", goVarName)
			g.writef("_ = %s // suppress unused\n", goVarName)
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

	default:
		return fmt.Errorf("unsupported pattern type in ADT match: %T", arm.Pattern)
	}
	return nil
}
