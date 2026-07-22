package types

// M-EFFECT-ROW-SHOW-INTERP (#386) helpers for the application-local effect-row
// solver in inferApp. Split out of typechecker_functions.go to keep that file
// under the 800-line AI-maintainability limit.

import (
	"fmt"
	"os"

	"github.com/sunholo-data/ailang/internal/core"
)

// lambdaHasEffectAnnotation reports whether the given lambda carried an explicit
// `! {...}` effect annotation in the surface syntax (with or without budgets).
func lambdaHasEffectAnnotation(tc *CoreTypeChecker, lam *core.Lambda) bool {
	return len(tc.effectAnnotsFull[lam.ID()]) > 0 || len(tc.effectAnnots[lam.ID()]) > 0
}

// DeclaredLambdaEffectRow returns the ORIGINAL source-declared effect row of an
// explicitly-annotated lambda (by NodeID), and whether one was recorded. Used by
// the effect validator to enforce closed inline-lambda annotations against body
// effects.
func (tc *CoreTypeChecker) DeclaredLambdaEffectRow(nodeID uint64) (*Row, bool) {
	if tc.declaredLambdaEffects == nil {
		return nil, false
	}
	r, ok := tc.declaredLambdaEffects[nodeID]
	return r, ok
}

// closeIfResolved returns the locally-substituted effect row `subbed` ONLY when
// the local solve resolved it to a fully CLOSED row (no tail) — the case that
// matters for #386, where a nested pure/known callee's effect row becomes closed
// (`show` -> {}, `println` -> {IO}) and must be visible before combination.
// Otherwise it returns the ORIGINAL `orig` row, leaving any still-open /
// row-polymorphic effect row to be resolved by the enclosing whole-program
// SolveConstraints. This deliberately avoids eagerly binding an open, shared
// effect tail (e.g. a recursive multi-arm function's `! {SharedMem | ρ}`), which
// otherwise caused "same row variable with different extensions".
func closeIfResolved(orig, subbed *Row) *Row {
	if subbed != nil && subbed.Tail == nil {
		return subbed
	}
	return orig
}

// assertSingleTailAfterAppSolve enforces the #386 application-boundary invariant:
// after the application-local equality solver has run, the effect-determining
// operands (substituted argument effects + substituted callee effect row) must
// contain AT MOST ONE distinct unresolved tail. Two distinct tails indicate a
// determining equality escaped the local solve; that is a fail-closed condition
// (a genuinely irreducible multi-tail union would be a new language feature), not
// something to paper over with an implicit join. Runs only in DEBUG_STRICT mode
// as a loud fail; otherwise it is a no-op so production inference degrades to the
// sound open-row over-approximation in combineEffects.
func assertSingleTailAfterAppSolve(rows []*Row, app *core.App) {
	if os.Getenv("DEBUG_STRICT") == "" {
		return
	}
	var seen string
	for _, r := range rows {
		if r == nil || r.Tail == nil {
			continue
		}
		if seen == "" {
			seen = r.Tail.Name
			continue
		}
		if r.Tail.Name != seen {
			panic(fmt.Sprintf("inferApp (#386 distinct-tail invariant): application at %s has two distinct "+
				"unresolved effect tails %q and %q after the application-local solve; a determining equality "+
				"was not solved before publishing the node", app.Span(), seen, r.Tail.Name))
		}
	}
}
