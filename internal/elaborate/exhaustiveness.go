package elaborate

import (
	"fmt"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// ExhaustivenessChecker analyzes match expressions for completeness
type ExhaustivenessChecker struct {
	// Future: type environment for looking up ADT definitions
}

// NewExhaustivenessChecker creates a new checker
func NewExhaustivenessChecker() *ExhaustivenessChecker {
	return &ExhaustivenessChecker{}
}

// CheckExhaustiveness checks if a match expression is exhaustive
// Returns true if exhaustive, false with missing patterns if not
func (ec *ExhaustivenessChecker) CheckExhaustiveness(match *core.Match, scrutineeType types.Type) (bool, []string) {
	// Build the universe of all possible patterns for this type
	universe := ec.buildUniverse(scrutineeType)

	// Track which patterns are covered
	uncovered := universe

	// Process each arm
	for _, arm := range match.Arms {
		// If there's a guard, we can't guarantee the arm will match
		// So we treat guarded patterns as partial coverage
		if arm.Guard != nil {
			// For now, we conservatively assume guarded patterns don't fully cover
			continue
		}

		// Remove patterns covered by this arm
		uncovered = ec.subtract(uncovered, ec.expandPattern(arm.Pattern))
	}

	// If there are uncovered patterns, the match is non-exhaustive
	if len(uncovered) > 0 {
		missing := make([]string, len(uncovered))
		for i, p := range uncovered {
			missing[i] = p.String()
		}
		return false, missing
	}

	return true, nil
}

// PatternSet represents a set of concrete patterns
type PatternSet []core.CorePattern

// buildUniverse constructs all possible patterns for a type
func (ec *ExhaustivenessChecker) buildUniverse(t types.Type) PatternSet {
	switch typ := t.(type) {
	case *types.TCon:
		switch typ.Name {
		case "Bool":
			// Bool has two constructors: true, false
			return PatternSet{
				&core.LitPattern{Value: true},
				&core.LitPattern{Value: false},
			}
		case "Int", "Float", "String":
			// These are infinite, so we represent them as a wildcard
			return PatternSet{&core.WildcardPattern{}}
		default:
			// Unknown type - assume wildcard
			return PatternSet{&core.WildcardPattern{}}
		}

	case *types.TList:
		// Lists are infinite, represent as wildcard
		// (Later: could detect specific list patterns like [], [_], [_, _])
		return PatternSet{&core.WildcardPattern{}}

	case *types.TTuple:
		// Tuple requires matching all elements
		// For now, treat as wildcard
		return PatternSet{&core.WildcardPattern{}}

	case *types.TVar:
		// Type variable - unknown, assume wildcard
		return PatternSet{&core.WildcardPattern{}}

	default:
		// Unknown type - conservative wildcard
		return PatternSet{&core.WildcardPattern{}}
	}
}

// expandPattern converts a pattern into the set of concrete patterns it covers
func (ec *ExhaustivenessChecker) expandPattern(p core.CorePattern) PatternSet {
	switch pat := p.(type) {
	case *core.WildcardPattern:
		// Wildcard matches everything
		return PatternSet{&core.WildcardPattern{}}

	case *core.VarPattern:
		// Variable patterns match everything (like wildcard)
		return PatternSet{&core.WildcardPattern{}}

	case *core.LitPattern:
		// Literal matches only itself
		return PatternSet{pat}

	case *core.ConstructorPattern:
		// Constructor matches only that constructor
		return PatternSet{pat}

	case *core.TuplePattern:
		// Tuple patterns match specific tuple structure
		return PatternSet{pat}

	case *core.ListPattern:
		// List patterns match specific list structure
		return PatternSet{pat}

	case *core.RecordPattern:
		// Record patterns match specific record structure
		return PatternSet{pat}

	default:
		// Unknown pattern type - conservatively assume it covers nothing
		return PatternSet{}
	}
}

// subtract removes patterns in 'covered' from 'universe'
// Returns the remaining uncovered patterns
func (ec *ExhaustivenessChecker) subtract(universe, covered PatternSet) PatternSet {
	// If covered contains a wildcard, everything is covered
	for _, p := range covered {
		if ec.isWildcard(p) {
			return PatternSet{} // Nothing left uncovered
		}
	}

	// If universe is just a wildcard and covered doesn't have wildcard,
	// we can't determine what's left (infinite set)
	if len(universe) == 1 && ec.isWildcard(universe[0]) {
		// For finite types (Bool), we need special handling
		// For now, conservatively assume nothing is subtracted
		return universe
	}

	// For finite sets, remove matching patterns
	var remaining PatternSet
	for _, uPat := range universe {
		matched := false
		for _, cPat := range covered {
			if ec.patternsMatch(uPat, cPat) {
				matched = true
				break
			}
		}
		if !matched {
			remaining = append(remaining, uPat)
		}
	}

	return remaining
}

// isWildcard checks if a pattern is a wildcard or variable (matches everything)
func (ec *ExhaustivenessChecker) isWildcard(p core.CorePattern) bool {
	switch p.(type) {
	case *core.WildcardPattern, *core.VarPattern:
		return true
	default:
		return false
	}
}

// patternsMatch checks if two patterns match the same values
func (ec *ExhaustivenessChecker) patternsMatch(p1, p2 core.CorePattern) bool {
	// If either is wildcard, they match
	if ec.isWildcard(p1) || ec.isWildcard(p2) {
		return true
	}

	// Check specific pattern types
	switch pat1 := p1.(type) {
	case *core.LitPattern:
		if pat2, ok := p2.(*core.LitPattern); ok {
			return pat1.Value == pat2.Value
		}
		return false

	case *core.ConstructorPattern:
		if pat2, ok := p2.(*core.ConstructorPattern); ok {
			if pat1.Name != pat2.Name {
				return false
			}
			if len(pat1.Args) != len(pat2.Args) {
				return false
			}
			for i := range pat1.Args {
				if !ec.patternsMatch(pat1.Args[i], pat2.Args[i]) {
					return false
				}
			}
			return true
		}
		return false

	default:
		// Conservative: patterns don't match
		return false
	}
}

// ExhaustivenessWarning represents a non-exhaustive match warning
type ExhaustivenessWarning struct {
	Location       string   // Source location
	MissingPattern []string // Missing patterns
}

func (w *ExhaustivenessWarning) String() string {
	if len(w.MissingPattern) == 1 {
		return fmt.Sprintf("warning: non-exhaustive match at %s\n  missing pattern: %s",
			w.Location, w.MissingPattern[0])
	}
	return fmt.Sprintf("warning: non-exhaustive match at %s\n  missing patterns: %v",
		w.Location, w.MissingPattern)
}
