package elaborate

import (
	"fmt"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
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

// elaboratePattern converts surface pattern to core pattern
func (e *Elaborator) elaboratePattern(pat ast.Pattern) (core.CorePattern, error) {
	switch p := pat.(type) {
	case *ast.Identifier:
		return &core.VarPattern{Name: p.Name}, nil
	case *ast.Literal:
		return &core.LitPattern{Value: p.Value}, nil
	case *ast.WildcardPattern:
		return &core.WildcardPattern{}, nil
	case *ast.ConstructorPattern:
		// Elaborate nested patterns
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
