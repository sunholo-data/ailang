// Package golang provides Go code generation from AILANG Core AST.
package golang

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

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
					g.writeSuppressUnused(goVarName)

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
			g.writeSuppressUnused(goVarName)
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
