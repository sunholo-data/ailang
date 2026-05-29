// Package golang provides Go code generation from AILANG Core AST.
package golang

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
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
