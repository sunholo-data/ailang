package elaborate

import (
	"fmt"
	"unicode"
	"unicode/utf8"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

// normalizeMatch handles pattern matching
func (e *Elaborator) normalizeMatch(match *ast.Match) (core.CoreExpr, error) {
	// Scrutinee must be atomic
	scrutinee, binds, err := e.normalizeToAtomic(match.Expr)
	if err != nil {
		return nil, err
	}

	// Convert arms
	var arms []core.MatchArm
	for _, caseClause := range match.Cases {
		pattern, err := e.elaboratePattern(caseClause.Pattern)
		if err != nil {
			return nil, err
		}

		body, err := e.normalize(caseClause.Body)
		if err != nil {
			return nil, err
		}

		// Elaborate guard if present
		var guard core.CoreExpr
		if caseClause.Guard != nil {
			guard, err = e.normalize(caseClause.Guard)
			if err != nil {
				return nil, fmt.Errorf("failed to elaborate guard: %w", err)
			}
		}

		arms = append(arms, core.MatchArm{
			Pattern: pattern,
			Guard:   guard,
			Body:    body,
		})
	}

	result := &core.Match{
		CoreNode:   e.makeNode(match.Position()),
		Scrutinee:  scrutinee,
		Arms:       arms,
		Exhaustive: false, // Will be checked below
	}

	// Check exhaustiveness (without type info, use simple heuristic)
	// For now, assume Bool type if we see boolean literals
	scrutineeType := e.inferScrutineeType(arms)
	if scrutineeType != nil {
		exhaustive, missing := e.exChecker.CheckExhaustiveness(result, scrutineeType)
		result.Exhaustive = exhaustive

		if !exhaustive {
			// Add warning with source location
			pos := match.Position()
			location := fmt.Sprintf("%s:%d:%d", e.filePath, pos.Line, pos.Column)
			e.warnings = append(e.warnings, &ExhaustivenessWarning{
				Location:       location,
				MissingPattern: missing,
			})
		}
	}

	return e.wrapWithBindings(result, binds), nil
}

// isUpperIdent reports whether an identifier starts with an uppercase letter,
// i.e. is a constructor reference by language convention (the parser rejects
// lowercase variant names with PAR_VARIANT_NEEDS_UIDENT).
func isUpperIdent(name string) bool {
	r, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(r)
}

// elaboratePattern converts surface pattern to core pattern
func (e *Elaborator) elaboratePattern(pat ast.Pattern) (core.CorePattern, error) {
	switch p := pat.(type) {
	case *ast.Identifier:
		// M-DX20: Check for wildcard pattern first
		// "_" is a wildcard that matches anything but binds nothing
		if p.Name == "_" {
			return &core.WildcardPattern{}, nil
		}
		// Check if this identifier is a nullary constructor
		// Nullary constructors appear as bare identifiers (e.g., "None", "Red")
		if ctorInfo, ok := e.constructors[p.Name]; ok {
			if ctorInfo.Arity == 0 {
				// It's a nullary constructor - create ConstructorPattern with no args
				return &core.ConstructorPattern{
					Name: p.Name,
					Args: nil, // Empty args for nullary constructor
				}, nil
			}
			// Bare non-nullary constructor (e.g. `Some` without arguments):
			// previously this silently became a catch-all VarPattern named "Some".
			return nil, fmt.Errorf("constructor %s in pattern expects %d argument(s) — write %s(...) with %d pattern(s), or use a lowercase name for a variable binding", p.Name, ctorInfo.Arity, p.Name, ctorInfo.Arity)
		}
		// Uppercase identifier = constructor by language convention (the parser
		// enforces UpperCamelCase at declaration: PAR_VARIANT_NEEDS_UIDENT).
		// A constructor that isn't in e.constructors (e.g. Option's None when
		// only `std/list (nth)` is imported) previously fell through to
		// VarPattern — a silent catch-all that matched EVERY value, so
		// `match nth(xs, i) { None => ..., Some(v) => ... }` always took the
		// None arm. Elaborate as a nullary constructor pattern instead; the
		// runtime matches by constructor name, same as it already does for
		// unimported constructor patterns with arguments like Some(v). (#323)
		if isUpperIdent(p.Name) {
			return &core.ConstructorPattern{
				Name: p.Name,
				Args: nil,
			}, nil
		}
		// Otherwise, it's a variable pattern
		return &core.VarPattern{Name: p.Name}, nil
	case *ast.Literal:
		return &core.LitPattern{Value: p.Value}, nil
	case *ast.WildcardPattern:
		return &core.WildcardPattern{}, nil
	case *ast.ConstructorPattern:
		// Special case: :: (cons) constructor for lists
		// ::(head, tail) should elaborate to a ListPattern with one element and a tail
		// CRITICAL: Must be ListPattern (not ConstructorPattern) because lists are ListValue at runtime
		// See internal/eval/eval_patterns.go - ListPattern matches ListValue, ConstructorPattern matches TaggedValue
		if p.Name == "::" {
			if len(p.Patterns) != 2 {
				return nil, fmt.Errorf(":: constructor requires exactly 2 arguments (head and tail), got %d", len(p.Patterns))
			}
			// Elaborate head pattern
			headPat, err := e.elaboratePattern(p.Patterns[0])
			if err != nil {
				return nil, err
			}
			// Elaborate tail pattern
			tailPat, err := e.elaboratePattern(p.Patterns[1])
			if err != nil {
				return nil, err
			}
			// Create ListPattern with one element and a tail
			return &core.ListPattern{
				Elements: []core.CorePattern{headPat},
				Tail:     &tailPat,
			}, nil
		}

		// General constructor pattern (ADT constructors)
		var args []core.CorePattern
		for _, argPat := range p.Patterns {
			coreArg, err := e.elaboratePattern(argPat)
			if err != nil {
				return nil, err
			}
			args = append(args, coreArg)
		}
		return &core.ConstructorPattern{
			Name: p.Name,
			Args: args,
		}, nil
	case *ast.TuplePattern:
		// Elaborate tuple element patterns
		var elements []core.CorePattern
		for _, elemPat := range p.Elements {
			coreElem, err := e.elaboratePattern(elemPat)
			if err != nil {
				return nil, err
			}
			elements = append(elements, coreElem)
		}
		return &core.TuplePattern{
			Elements: elements,
		}, nil
	case *ast.ListPattern:
		// Elaborate list element patterns
		var elements []core.CorePattern
		for _, elemPat := range p.Elements {
			coreElem, err := e.elaboratePattern(elemPat)
			if err != nil {
				return nil, err
			}
			elements = append(elements, coreElem)
		}

		// Elaborate rest pattern if present
		var tail *core.CorePattern
		if p.Rest != nil {
			restCore, err := e.elaboratePattern(p.Rest)
			if err != nil {
				return nil, err
			}
			tail = &restCore
		}

		return &core.ListPattern{
			Elements: elements,
			Tail:     tail,
		}, nil
	case *ast.RecordPattern:
		// Elaborate record pattern - convert AST FieldPattern list to Core map
		fields := make(map[string]core.CorePattern)
		for _, fp := range p.Fields {
			corePat, err := e.elaboratePattern(fp.Pattern)
			if err != nil {
				return nil, err
			}
			fields[fp.Name] = corePat
		}
		return &core.RecordPattern{
			Fields: fields,
		}, nil
	default:
		return nil, fmt.Errorf("pattern elaboration not implemented for %T", pat)
	}
}

// inferScrutineeType attempts to infer the type of a scrutinee from its patterns
// This is a simple heuristic - returns Bool if we see boolean literals
func (e *Elaborator) inferScrutineeType(arms []core.MatchArm) types.Type {
	// Look at patterns to infer type
	for _, arm := range arms {
		if litPat, ok := arm.Pattern.(*core.LitPattern); ok {
			switch litPat.Value.(type) {
			case bool:
				return &types.TCon{Name: "Bool"}
			case int, int64:
				return &types.TCon{Name: "Int"}
			case float64:
				return &types.TCon{Name: "Float"}
			case string:
				return &types.TCon{Name: "String"}
			}
		}
	}
	// Can't infer type - return nil
	return nil
}
