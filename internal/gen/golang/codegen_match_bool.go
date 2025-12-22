// Package golang provides Go code generation from AILANG Core AST.
package golang

import (
	"github.com/sunholo/ailang/internal/core"
)

// BoolMatchChainEntry represents one condition-result pair in a bool match chain.
// M-CODEGEN-LIST: Used to flatten nested bool matches into if-else chains.
type BoolMatchChainEntry struct {
	Condition   core.CoreExpr   // The condition (scrutinee of match), nil for final else
	TrueBody    core.CoreExpr   // What to return when condition is true
	LetBindings []core.CoreExpr // ANF let bindings that precede this condition
}

// extractBoolMatchChain detects and extracts a chain of nested bool matches.
// M-CODEGEN-LIST: Pattern: match <bool> { true => A, false => match <bool> { ... } }
//
// Also handles ANF form where false arm contains Let bindings before the nested Match:
// match <bool> { true => A, false => Let x = ... in match <bool> { ... } }
//
// Returns nil if the match is not a bool match chain, or a slice of entries if it is.
// The returned slice has one entry per condition, with the final false body as the last entry
// (with nil Condition to indicate it's the else case).
func extractBoolMatchChain(match *core.Match) []BoolMatchChainEntry {
	var chain []BoolMatchChainEntry
	currentExpr := core.CoreExpr(match)

	for {
		// Unwrap Let bindings to find the Match
		// M-CODEGEN-LIST: In ANF, comparisons are Let-bound before the Match
		var current *core.Match
		var letBindings []core.CoreExpr // Collect bindings to preserve

		switch e := currentExpr.(type) {
		case *core.Match:
			current = e
		case *core.Let:
			// Unwrap let chain to find nested Match
			inner := e
			for {
				letBindings = append(letBindings, inner.Value)
				if nestedMatch, ok := inner.Body.(*core.Match); ok {
					current = nestedMatch
					break
				} else if nestedLet, ok := inner.Body.(*core.Let); ok {
					inner = nestedLet
				} else {
					// Not a Match chain
					return nil
				}
			}
		default:
			return nil
		}

		// Check if this match has exactly 2 arms with true/false literal patterns
		if len(current.Arms) != 2 {
			return nil
		}

		var trueArm, falseArm *core.MatchArm
		for i := range current.Arms {
			arm := &current.Arms[i]
			if lp, ok := arm.Pattern.(*core.LitPattern); ok {
				// LitPattern.Value is the Go value directly (bool), not *core.Lit
				if val, ok := lp.Value.(bool); ok {
					if val {
						trueArm = arm
					} else {
						falseArm = arm
					}
				}
			}
		}

		// Must have both true and false arms
		if trueArm == nil || falseArm == nil {
			return nil
		}

		// Add this condition to the chain
		// M-CODEGEN-LIST: If there's exactly one let binding, use it as the actual condition.
		// In ANF, `Let $cmp = x > 3 in Match $cmp {...}` - the real condition is `x > 3`, not `$cmp`.
		// If there are multiple bindings or none, use the scrutinee directly.
		var actualCondition core.CoreExpr
		if len(letBindings) == 1 {
			// Single let binding - use the comparison expression directly
			actualCondition = letBindings[0]
			letBindings = nil // Don't need to emit separately
		} else if len(letBindings) > 1 {
			// Multiple bindings - use the last one as condition, keep others
			actualCondition = letBindings[len(letBindings)-1]
			letBindings = letBindings[:len(letBindings)-1]
		} else {
			// No bindings - use scrutinee directly
			actualCondition = current.Scrutinee
		}

		chain = append(chain, BoolMatchChainEntry{
			Condition:   actualCondition,
			TrueBody:    trueArm.Body,
			LetBindings: letBindings,
		})

		// Check if false arm is another bool match (continuation of chain)
		// The false arm body could be:
		// 1. Direct Match - nested bool match
		// 2. Let { ... in Match } - ANF form with let-bound comparison
		falseBody := falseArm.Body
		switch fb := falseBody.(type) {
		case *core.Match:
			currentExpr = fb
			// Continue the loop
		case *core.Let:
			currentExpr = fb
			// Continue the loop - will unwrap Let chain
		default:
			// End of chain - add the final else body with nil Condition
			chain = append(chain, BoolMatchChainEntry{
				Condition: nil, // nil indicates else case
				TrueBody:  falseBody,
			})
			goto done
		}
	}

done:
	// Only return chain if we have at least 2 conditions (worth flattening)
	if len(chain) >= 2 {
		return chain
	}
	return nil
}

// generateFlatBoolMatchChain generates a flat if-else chain for bool match chains.
// M-CODEGEN-LIST: Eliminates nested IIFEs by generating flat if-else structure.
//
// Instead of:
//
//	func() interface{} {
//	    switch cond1 {
//	    case true: return A
//	    case false: return func() interface{} {
//	        switch cond2 {
//	        case true: return B
//	        case false: return C
//	        }
//	    }()
//	    }
//	}()
//
// We generate:
//
//	func() interface{} {
//	    if cond1.(bool) { return A }
//	    else if cond2.(bool) { return B }
//	    else { return C }
//	}()
func (g *Generator) generateFlatBoolMatchChain(chain []BoolMatchChainEntry, returnType string) error {
	g.writef("func() %s {\n", returnType)
	g.indent++

	for i, entry := range chain {
		// M-CODEGEN-LIST: Emit any let bindings that weren't inlined
		for _, binding := range entry.LetBindings {
			varName := g.uniqueVarName("cond")
			g.writef("%s := ", varName)
			if err := g.generateExpr(binding); err != nil {
				return err
			}
			g.writef("\n")
			// Note: These bindings are needed if there are multiple let-bindings
			// before a single match. The var isn't used directly but ensures
			// side effects (if any) happen in order.
			_ = varName
		}

		if entry.Condition == nil {
			// This is the else case (last entry)
			g.writef("} else {\n")
			g.indent++
			g.writef("return ")
			if err := g.generateExpr(entry.TrueBody); err != nil {
				return err
			}
			g.writef("\n")
			g.indent--
			g.writef("}\n")
		} else if i == 0 {
			// First condition
			g.writef("if ")
			if err := g.generateExpr(entry.Condition); err != nil {
				return err
			}
			// M-DX27: Only add type assertion if condition produces interface{}
			if g.exprProducesInterface(entry.Condition) {
				g.writef(".(bool)")
			}
			g.writef(" {\n")
			g.indent++
			g.writef("return ")
			if err := g.generateExpr(entry.TrueBody); err != nil {
				return err
			}
			g.writef("\n")
			g.indent--
		} else {
			// Subsequent conditions (else if)
			g.writef("} else if ")
			if err := g.generateExpr(entry.Condition); err != nil {
				return err
			}
			// M-DX27: Only add type assertion if condition produces interface{}
			if g.exprProducesInterface(entry.Condition) {
				g.writef(".(bool)")
			}
			g.writef(" {\n")
			g.indent++
			g.writef("return ")
			if err := g.generateExpr(entry.TrueBody); err != nil {
				return err
			}
			g.writef("\n")
			g.indent--
		}
	}

	g.indent--
	g.writef("}()")
	return nil
}
