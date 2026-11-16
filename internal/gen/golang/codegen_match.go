// Package golang provides Go code generation from AILANG Core AST.
package golang

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/core"
	ailerrors "github.com/sunholo-data/ailang/internal/errors"
)

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
		// ADT constructor matching - use Kind-based switch, on the ADT the
		// type checker stamped on the constructor patterns (#2244: never the
		// scrutinee's written type, which may be an alias with no Kind
		// constants; #2303: never the first registered ADT declaring the
		// constructor name, whose Kind constants belong to another type).
		var adtTypeName string
		for _, arm := range match.Arms {
			if cp, ok := arm.Pattern.(*core.ConstructorPattern); ok {
				info, err := g.patternCtor(cp)
				if err != nil {
					return err
				}
				adtTypeName = info.TypeName
				break
			}
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
			g.writef("panic(%q)\n", noMatchPanicText(match))
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
			g.writef("panic(%q)\n", noMatchPanicText(match))
			g.indent--
		}
	}
	g.writef("}\n")

	g.indent--
	g.write("}()")
	return nil
}

// patternCtor returns the registered constructor a constructor pattern
// tests: the one of the ADT the type checker stamped on the pattern
// (TypeName). A pattern with no stamp, or a stamp no registered ADT matches,
// is an error, never a lookup by the constructor name alone — two ADTs may
// declare that name (#2303).
func (g *Generator) patternCtor(p *core.ConstructorPattern) (*ADTConstructorInfo, error) {
	if p.TypeName == "" {
		return nil, fmt.Errorf("codegen: constructor pattern %s has no type: the type checker resolves every constructor pattern to its type (TC_MATCH_007 otherwise), so this is a compiler bug", p.Name)
	}
	info, ok := g.LookupADTConstructorByQualifiedName(p.TypeName + "." + p.Name)
	if !ok {
		return nil, fmt.Errorf("codegen: constructor pattern %s.%s names no registered constructor", p.TypeName, p.Name)
	}
	return info, nil
}

// noMatchPanicText is the panic message for a generated match with no
// catch-all: the same RT002 text both interpreters raise, with the match's
// source position (M-V1-MATCH-EXHAUSTIVENESS M0). The scrutinee's shape is a
// run-time fact the static text does not carry.
func noMatchPanicText(match *core.Match) string {
	mf := &ailerrors.MatchFailureError{}
	p := match.OriginalSpan()
	if p.Line == 0 {
		p = match.Span()
	}
	if p.Line > 0 {
		mf.Pos = p.String()
	}
	return mf.Error()
}
