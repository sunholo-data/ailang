// Package golang provides Go code generation from AILANG Core AST.
package golang

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/core"
)

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
						if s := suppressUnusedStr(ToGoVarName(elemPat.Name)); s != "" {
							bindings = append(bindings, s)
						}
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
					if s := suppressUnusedStr(tempVar); s != "" {
						bindings = append(bindings, s)
					}
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
				if s := suppressUnusedStr(ToGoVarName(tailPat.Name)); s != "" {
					bindings = append(bindings, s)
				}
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
		if s := suppressUnusedStr(ToGoVarName(pat.Name)); s != "" {
			bindings = append(bindings, s)
		}
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
					if s := suppressUnusedStr(ToGoVarName(elemPat.Name)); s != "" {
						bindings = append(bindings, s)
					}
				}
			case *core.WildcardPattern:
				// No binding needed for wildcards
			case *core.TuplePattern:
				// Nested tuple - generate temp var and recurse
				tempVar := fmt.Sprintf("_tuple%d", i)
				binding := fmt.Sprintf("%s := %s.([]interface{})[%d]", tempVar, scrutinee, i)
				bindings = append(bindings, binding)
				if s := suppressUnusedStr(tempVar); s != "" {
					bindings = append(bindings, s)
				}
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
					if s := suppressUnusedStr(ToGoVarName(ap.Name)); s != "" {
						bindings = append(bindings, s)
					}
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
